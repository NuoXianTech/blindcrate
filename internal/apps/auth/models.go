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
	"errors"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint64     `json:"id" gorm:"primaryKey"`
	Username     string     `json:"username" gorm:"size:255;unique"`
	PasswordHash string     `json:"-" gorm:"size:255"`
	Nickname     string     `json:"nickname" gorm:"size:255"`
	AvatarUrl    string     `json:"avatar_url" gorm:"size:255"`
	IsActive     bool       `json:"is_active" gorm:"default:true"`
	IsAdmin      bool       `json:"is_admin" gorm:"default:false"`
	BannedUntil  *time.Time `json:"banned_until" gorm:"index"`
	BanReason    string     `json:"ban_reason" gorm:"size:255"`
	LastLoginAt  *time.Time `json:"last_login_at" gorm:"index"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"autoUpdateTime;index"`
}

func (u *User) TableName() string {
	return "users"
}

func (u *User) Exact(tx *gorm.DB, id uint64) error {
	if err := tx.Where("id = ?", id).First(u).Error; err != nil {
		return err
	}
	return nil
}

// UpdateLastLogin 更新最后登录时间
func (u *User) UpdateLastLogin(tx *gorm.DB) error {
	now := time.Now()
	u.LastLoginAt = &now
	return tx.Model(u).Update("last_login_at", now).Error
}

// CheckActive 检查用户账户是否激活,未激活则返回错误
func (u *User) CheckActive() error {
	if !u.IsActive {
		return errors.New(BannedAccount)
	}
	return nil
}

// CheckBan 检查用户是否处于封禁期,仍在封禁则返回错误
func (u *User) CheckBan() error {
	if u.IsBanned(time.Now()) {
		return errors.New(BannedAccount)
	}
	return nil
}

// IsBanned 当前是否处于封禁状态
//
// BannedUntil 为 nil 表示没封过,过去时间视为已解封
func (u *User) IsBanned(now time.Time) bool {
	if u.BannedUntil == nil {
		return false
	}
	return u.BannedUntil.After(now)
}
