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

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
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
