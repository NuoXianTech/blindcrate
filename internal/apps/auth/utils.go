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
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/fault"
	"github.com/NuoXianTech/blindcrate/internal/storage"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// GetUserIDFromSession 从 session 中获取用户 ID, 未登录返回 0
func GetUserIDFromSession(s sessions.Session) uint64 {
	userID, ok := s.Get(UserIDKey).(uint64)
	if !ok {
		return 0
	}
	return userID
}

// GetUserIDFromContext 从当前请求的 session 中获取用户 ID, 未登录返回 0
func GetUserIDFromContext(c *gin.Context) uint64 {
	session := sessions.Default(c)
	return GetUserIDFromSession(session)
}

// sessionStamp 账号的标识: 注册时间, 精确到微秒(数据库保存的精度)
//
// session 存在 Redis, 用户存在数据库。数据库重建、恢复备份或换库而 Redis 没有清空时,
// 新注册的账号会用到旧账号的 ID, 只按 ID 认人的话旧 session 会直接进入新账号。
// ID 相同的两个账号注册时间不同, 所以登录时把它写入 session, 之后每次请求都核对。
func sessionStamp(user *User) int64 {
	return user.CreatedAt.UnixMicro()
}

// setSessionUser 把登录的用户写入 session, 调用方负责保存
func setSessionUser(s sessions.Session, user *User) {
	s.Set(UserIDKey, user.ID)
	s.Set(UserNameKey, user.Username)
	s.Set(UserStampKey, sessionStamp(user))
}

// sessionMatchesUser session 是不是登录这个账号时建立的; 没有账号标识的旧 session 也不算
func sessionMatchesUser(s sessions.Session, user *User) bool {
	stamp, ok := s.Get(UserStampKey).(int64)
	return ok && stamp == sessionStamp(user)
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

// CurrentUser 当前登录的用户, 只能在 LoginRequired 之后的接口中调用
//
// 没有用户说明接口没有注册在需要登录的分组下, 属于代码错误, 直接 panic, 由 Recovery 返回 500。
func CurrentUser(c *gin.Context) *User {
	user, ok := GetUserFromContext(c)
	if !ok {
		panic("auth: CurrentUser called outside LoginRequired")
	}
	return user
}

// SetUserToContext 将User对象存储到Context中
func SetUserToContext(c *gin.Context, user *User) {
	c.Set(UserObjKey, user)
}

// usernamePattern 用户名只允许字母、数字和下划线
var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

// validateUsername 校验用户名格式
func validateUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return fault.Invalid(UsernameInvalid)
	}
	return nil
}

// validatePassword 校验密码长度, bcrypt 最多只处理 72 字节
func validatePassword(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return fault.Invalid(PasswordInvalid)
	}
	return nil
}

// validateNickname 昵称去掉首尾空白后需为 1-32 个字符
func validateNickname(nickname string) error {
	n := utf8.RuneCountInString(nickname)
	if n < 1 || n > 32 {
		return fault.Invalid(NicknameInvalid)
	}
	return nil
}

// validateAvatarURL 头像地址允许为空(清除头像), 否则必须是 http(s) 链接或上传的图片, 见 storage.ValidURL
func validateAvatarURL(avatarURL string) error {
	if avatarURL == "" {
		return nil
	}
	if len(avatarURL) > 1024 || !storage.ValidURL(avatarURL) {
		return fault.Invalid(AvatarURLInvalid)
	}
	return nil
}

// userSessionsKey 用户 session ID 集合的 key
func userSessionsKey(userID uint64) string {
	return userSessionsKeyPrefix + strconv.FormatUint(userID, 10)
}

// emailCodeKey 用户邮箱验证码的 key
func emailCodeKey(userID uint64) string {
	return emailCodeKeyPrefix + strconv.FormatUint(userID, 10)
}

// emailCodeCooldownKey 用户邮箱验证码发送冷却的 key
func emailCodeCooldownKey(userID uint64) string {
	return emailCodeCooldownKeyPrefix + strconv.FormatUint(userID, 10)
}

// registerCodeKey 注册验证码的 key, email 为规范化后的邮箱
func registerCodeKey(email string) string {
	return registerCodeKeyPrefix + email
}

// registerCodeCooldownKey 注册验证码发送冷却的 key
func registerCodeCooldownKey(email string) string {
	return registerCodeCooldownKeyPrefix + email
}

// generateEmailCode 生成 6 位数字验证码
func generateEmailCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
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
