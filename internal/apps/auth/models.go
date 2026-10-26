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
	ID              uint64     `json:"id" gorm:"primaryKey"`
	Username        string     `json:"username" gorm:"size:64;not null;index:uniq_users_username_lower,unique,expression:lower(username)"`
	PasswordHash    string     `json:"-" gorm:"size:255;not null;default:''"`
	Email           string     `json:"email" gorm:"size:255;not null;default:'';index:uniq_users_email_lower,unique,expression:lower(email),where:email <> ''"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	Nickname        string     `json:"nickname" gorm:"size:64;not null;default:''"`
	AvatarURL       string     `json:"avatar_url" gorm:"size:1024;not null;default:''"`
	IsActive        bool       `json:"is_active" gorm:"not null;default:true"`
	IsAdmin         bool       `json:"is_admin" gorm:"not null;default:false"`
	BannedUntil     *time.Time `json:"banned_until" gorm:"index"`
	BanReason       string     `json:"ban_reason" gorm:"size:255;not null;default:''"`
	LastLoginAt     *time.Time `json:"last_login_at" gorm:"index"`
	CreatedAt       time.Time  `json:"created_at" gorm:"not null;autoCreateTime;index"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

func (u *User) TableName() string {
	return "users"
}

// HasPassword 是否设置了密码, 仅通过第三方登录注册的用户没有密码
func (u *User) HasPassword() bool {
	return u.PasswordHash != ""
}

// UserIdentity 用户绑定的外部身份(第三方登录), 一个用户可绑定多个
type UserIdentity struct {
	ID             uint64    `json:"id" gorm:"primaryKey"`
	UserID         uint64    `json:"user_id" gorm:"not null;index"`
	Provider       string    `json:"provider" gorm:"size:32;not null;uniqueIndex:uniq_user_identities_provider_uid"`
	ProviderUserID string    `json:"provider_user_id" gorm:"size:255;not null;uniqueIndex:uniq_user_identities_provider_uid"`
	CreatedAt      time.Time `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

func (i *UserIdentity) TableName() string {
	return "user_identities"
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
