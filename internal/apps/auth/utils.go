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
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func GetUserIDFromSession(s sessions.Session) uint64 {
	userID, ok := s.Get(UserIDKey).(uint64)
	if !ok {
		return 0
	}
	return userID
}

func GetUserIDFromContext(c *gin.Context) uint64 {
	session := sessions.Default(c)
	return GetUserIDFromSession(session)
}

// GetUserFromContext 从Context中获取User对象
func GetUserFromContext(c *gin.Context) (*User, bool) {
	user, exists := c.Get(UserObjKey)
	if !exists {
		return nil, false
	}
	u, ok := user.(*User)
	return u, ok
}

// SetUserToContext 将User对象存储到Context中
func SetUserToContext(c *gin.Context, user *User) {
	c.Set(UserObjKey, user)
}

func doLogin(ctx context.Context, username, password string) (*User, error) {
	// query user
	var user User
	tx := db.DB(ctx).Where("LOWER(username) = LOWER(?)", username).First(&user)
	if tx.Error != nil {
		if !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			logger.ErrorF(ctx, "[Login] query user failed: %s", tx.Error.Error())
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
		logger.ErrorF(ctx, "[Login] update last login failed: %s", err.Error())
	}

	return &user, nil
}

// usernamePattern 用户名只允许字母、数字和下划线
var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

func validateUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return errors.New(UsernameInvalid)
	}
	return nil
}

// validatePassword 校验密码长度, bcrypt 最多只处理 72 字节
func validatePassword(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return errors.New(PasswordInvalid)
	}
	return nil
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
		user.IsAdmin = !hasUser

		if err := tx.Create(&user).Error; err != nil {
			if isUniqueViolation(err) {
				return errors.New(UsernameTaken)
			}
			return err
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

// isUniqueViolation 是否为唯一约束冲突
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
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

func userSessionsKey(userID uint64) string {
	return userSessionsKeyPrefix + strconv.FormatUint(userID, 10)
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

// HashPassword 生成密码哈希
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// comparePassword 校验密码
func comparePassword(passwordHash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
}

// dummyPasswordHash 陪跑比对用的固定哈希, 启动时生成一次
var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("blindcrate"), bcrypt.DefaultCost)

// compareDummyPassword 用户名不存在时陪跑一次比对
//
// 目的是让"用户不存在"与"密码错误"两条路径的耗时接近,
// 否则攻击者可以靠响应时间枚举出哪些用户名有效。
// 哈希必须预先生成, 否则这条路径会多一次 bcrypt 计算, 反而更慢。
func compareDummyPassword(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
}
