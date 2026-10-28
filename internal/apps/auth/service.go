/*
 * MIT License
 *
 * Copyright (c) 2026 nuoxi4n, NuoXianTech
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/mail"
	"gorm.io/gorm"
)

// RegisterHook 在注册的事务内、创建用户之后执行, 返回错误时注册失败并回滚
type RegisterHook func(tx *gorm.DB, user *User) error

// registerHooks 已注册的注册钩子, 只在服务启动前写入
var registerHooks []RegisterHook

// OnRegister 添加注册钩子, 需要在服务启动前调用, 按添加顺序执行
//
// 其他模块依赖 auth, auth 不能反过来依赖它们, 注册时要做的事通过钩子接入。
func OnRegister(hook RegisterHook) {
	registerHooks = append(registerHooks, hook)
}

// doLogin 用户名或已验证邮箱 + 密码登录
func doLogin(ctx context.Context, identifier, password string) (*User, error) {
	// query user, 用户名不允许包含 "@", 据此区分邮箱
	var user User
	query := db.DB(ctx)
	if strings.Contains(identifier, "@") {
		authSettings, err := settings.Auth.Get(ctx)
		if err != nil {
			return nil, err
		}
		if !authSettings.EmailLoginEnabled {
			return nil, errors.New(EmailLoginDisabled)
		}
		query = query.Where("LOWER(email) = LOWER(?) AND email_verified_at IS NOT NULL", identifier)
	} else {
		query = query.Where("LOWER(username) = LOWER(?)", identifier)
	}
	tx := query.First(&user)
	if tx.Error != nil {
		if !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, tx.Error
		}
		// 用户不存在时也走一次 bcrypt,让两种失败的耗时接近
		compareDummyPassword(password)
		return nil, errors.New(InvalidCredentials)
	}

	// verify password
	if !user.HasPassword() {
		compareDummyPassword(password)
		return nil, errors.New(InvalidCredentials)
	}
	if err := comparePassword(user.PasswordHash, password); err != nil {
		return nil, errors.New(InvalidCredentials)
	}

	// check status
	if err := user.CheckDisabled(); err != nil {
		return nil, err
	}
	if err := user.CheckBan(); err != nil {
		return nil, err
	}

	// update last login
	if err := user.UpdateLastLogin(db.DB(ctx)); err != nil {
		logger.ErrorF(ctx, "[Login] update last login failed: %v", err)
	}

	return &user, nil
}

// doRegister 注册新用户, 系统中第一个注册的用户自动成为管理员
func doRegister(ctx context.Context, username, password string) (*User, error) {
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	authSettings, err := settings.Auth.Get(ctx)
	if err != nil {
		return nil, err
	}
	user := User{Username: username, PasswordHash: hash, Nickname: username}
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 事务级咨询锁: 并发注册时保证只有一个人成为第一个用户(管理员)
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext('blindcrate:register'))").Error; err != nil {
			return err
		}

		var count int64
		if err := tx.Model(&User{}).Where("LOWER(username) = LOWER(?)", username).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New(UsernameTaken)
		}

		var hasUser bool
		if err := tx.Raw("SELECT EXISTS (SELECT 1 FROM users)").Scan(&hasUser).Error; err != nil {
			return err
		}
		// 系统中还没有用户时不受注册开关限制, 保证总能创建出第一个管理员
		if hasUser && !authSettings.RegisterEnabled {
			return errors.New(RegisterClosed)
		}
		user.IsAdmin = !hasUser

		if err := tx.Create(&user).Error; err != nil {
			if db.IsUniqueViolation(err) {
				return errors.New(UsernameTaken)
			}
			return err
		}
		for _, hook := range registerHooks {
			if err := hook(tx, &user); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if user.IsAdmin {
		logger.InfoF(ctx, "[Register] first user %d %s registered as admin", user.ID, user.Username)
	}
	return &user, nil
}

// doChangePassword 修改密码, 未设置过密码(仅第三方登录)的用户无需提供原密码
func doChangePassword(ctx context.Context, user *User, oldPassword, newPassword string) error {
	if user.HasPassword() {
		if err := comparePassword(user.PasswordHash, oldPassword); err != nil {
			return errors.New(WrongPassword)
		}
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return db.DB(ctx).Model(user).Update("password_hash", hash).Error
}

// trackSession 记录用户的 session ID, 集合有效期与 session 一致
func trackSession(ctx context.Context, userID uint64, sessionID string) error {
	key := userSessionsKey(userID)
	pipe := db.Redis.TxPipeline()
	pipe.SAdd(ctx, key, sessionID)
	pipe.Expire(ctx, key, time.Duration(config.Config.App.SessionAge)*time.Second)
	_, err := pipe.Exec(ctx)
	return err
}

// untrackSession 移除用户的某个 session ID(登出时调用)
func untrackSession(ctx context.Context, userID uint64, sessionID string) error {
	return db.Redis.SRem(ctx, userSessionsKey(userID), sessionID).Err()
}

// RevokeUserSessions 让用户的 session 全部失效, keepSessionID 非空时保留该 session(当前设备)
func RevokeUserSessions(ctx context.Context, userID uint64, keepSessionID string) error {
	key := userSessionsKey(userID)
	sessionIDs, err := db.Redis.SMembers(ctx, key).Result()
	if err != nil {
		return err
	}

	pipe := db.Redis.TxPipeline()
	for _, sessionID := range sessionIDs {
		if sessionID == keepSessionID {
			continue
		}
		pipe.Del(ctx, sessionKeyPrefix+sessionID)
		pipe.SRem(ctx, key, sessionID)
	}
	_, err = pipe.Exec(ctx)
	return err
}

// IsSessionActive session 是否仍然有效: 没有过期, 也没有因登出、改密码、封禁或停用被移除
func IsSessionActive(ctx context.Context, userID uint64, sessionID string) (bool, error) {
	pipe := db.Redis.Pipeline()
	tracked := pipe.SIsMember(ctx, userSessionsKey(userID), sessionID)
	exists := pipe.Exists(ctx, sessionKeyPrefix+sessionID)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	return tracked.Val() && exists.Val() > 0, nil
}

// doSendEmailCode 向待绑定的邮箱发送验证码
//
// 邮箱在验证通过前只暂存在 Redis, 不写入 users 表,
// 避免他人先填写你的邮箱占住唯一索引。
func doSendEmailCode(ctx context.Context, userID uint64, rawEmail string) error {
	email, err := mail.NormalizeAddress(rawEmail)
	if err != nil {
		return errors.New(EmailInvalid)
	}

	// 邮箱已被其他用户绑定
	var count int64
	if err = db.DB(ctx).Model(&User{}).
		Where("LOWER(email) = ? AND id <> ?", email, userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New(EmailTaken)
	}
	site, err := settings.Site.Get(ctx)
	if err != nil {
		return err
	}

	// 发送冷却
	ok, err := db.Redis.SetNX(ctx, emailCodeCooldownKey(userID), 1, emailCodeCooldown).Result()
	if err != nil {
		return err
	}
	if !ok {
		return errors.New(CodeTooFrequent)
	}

	code, err := generateEmailCode()
	if err != nil {
		return err
	}
	key := emailCodeKey(userID)
	pipe := db.Redis.TxPipeline()
	pipe.Del(ctx, key)
	pipe.HSet(ctx, key, "email", email, "code", code, "attempts", 0)
	pipe.Expire(ctx, key, emailCodeTTL)
	if _, err = pipe.Exec(ctx); err != nil {
		return err
	}

	subject := fmt.Sprintf("%s 邮箱验证码", site.Name)
	body := fmt.Sprintf("您的验证码是 %s, %d 分钟内有效。\n如非本人操作, 请忽略此邮件。", code, int(emailCodeTTL.Minutes()))
	if err = mail.Send(ctx, email, subject, body); err != nil {
		logger.ErrorF(ctx, "[EmailCode] send to user %d failed: %v", userID, err)
		db.Redis.Del(ctx, key, emailCodeCooldownKey(userID))
		return errors.New(MailUnavailable)
	}
	return nil
}

// doVerifyEmail 校验验证码, 通过后把邮箱写入 users 并标记已验证
func doVerifyEmail(ctx context.Context, userID uint64, rawEmail, code string) error {
	email, err := mail.NormalizeAddress(rawEmail)
	if err != nil {
		return errors.New(EmailInvalid)
	}

	key := emailCodeKey(userID)
	stored, err := db.Redis.HGetAll(ctx, key).Result()
	if err != nil {
		return err
	}
	if stored["code"] == "" {
		return errors.New(CodeInvalid)
	}
	// 超过尝试次数直接作废, 防止暴力枚举 6 位验证码
	attempts, err := db.Redis.HIncrBy(ctx, key, "attempts", 1).Result()
	if err != nil {
		return err
	}
	if attempts > emailCodeMaxAttempts {
		db.Redis.Del(ctx, key)
		return errors.New(CodeInvalid)
	}
	if stored["email"] != email || subtle.ConstantTimeCompare([]byte(stored["code"]), []byte(strings.TrimSpace(code))) != 1 {
		return errors.New(CodeInvalid)
	}

	if err = db.DB(ctx).Model(&User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{"email": email, "email_verified_at": time.Now()}).Error; err != nil {
		if db.IsUniqueViolation(err) {
			return errors.New(EmailTaken)
		}
		return err
	}
	db.Redis.Del(ctx, key)
	return nil
}

// ProfileChanges 资料修改的操作日志内容, 只记录本次传入的字段
func ProfileChanges(username, nickname, avatarURL *string) map[string]interface{} {
	changes := map[string]interface{}{}
	if username != nil {
		changes["username"] = *username
	}
	if nickname != nil {
		changes["nickname"] = *nickname
	}
	if avatarURL != nil {
		changes["avatar_url"] = *avatarURL
	}
	return changes
}

// UpdateUserProfile 更新用户资料, 字段为 nil 表示不修改; username 仅管理员可改
func UpdateUserProfile(ctx context.Context, userID uint64, username, nickname, avatarURL *string) (*User, error) {
	values := map[string]interface{}{}
	if username != nil {
		if err := validateUsername(*username); err != nil {
			return nil, err
		}
		values["username"] = *username
	}
	if nickname != nil {
		*nickname = strings.TrimSpace(*nickname)
		if err := validateNickname(*nickname); err != nil {
			return nil, err
		}
		values["nickname"] = *nickname
	}
	if avatarURL != nil {
		*avatarURL = strings.TrimSpace(*avatarURL)
		if err := validateAvatarURL(*avatarURL); err != nil {
			return nil, err
		}
		values["avatar_url"] = *avatarURL
	}

	var user User
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := user.Exact(tx, userID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(UserNotFound)
			}
			return err
		}
		if len(values) == 0 {
			return nil
		}
		if err := tx.Model(&user).Updates(values).Error; err != nil {
			if db.IsUniqueViolation(err) {
				return errors.New(UsernameTaken)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}
