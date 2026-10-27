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

// User 用户账号, 用户名和邮箱(非空时)都不区分大小写唯一
type User struct {
	ID              uint64     `gorm:"primaryKey" json:"id"`
	Username        string     `gorm:"size:64;not null;index:uniq_users_username_lower,unique,expression:lower(username)" json:"username"`
	PasswordHash    string     `gorm:"size:255;not null;default:''" json:"-"`
	Email           string     `gorm:"size:255;not null;default:'';index:uniq_users_email_lower,unique,expression:lower(email),where:email <> ''" json:"email"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	Nickname        string     `gorm:"size:64;not null;default:''" json:"nickname"`
	AvatarURL       string     `gorm:"size:1024;not null;default:''" json:"avatar_url"`
	IsDisabled      bool       `gorm:"not null;default:false" json:"is_disabled"`
	IsAdmin         bool       `gorm:"not null;default:false" json:"is_admin"`
	BannedUntil     *time.Time `gorm:"index" json:"banned_until"`
	BanReason       string     `gorm:"size:255;not null;default:''" json:"ban_reason"`
	LastLoginAt     *time.Time `gorm:"index" json:"last_login_at"`
	CreatedAt       time.Time  `gorm:"not null;default:now();autoCreateTime;index" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (User) TableName() string { return "users" }

// Exact 按 ID 查询用户
func (u *User) Exact(tx *gorm.DB, id uint64) error {
	if err := tx.Where("id = ?", id).First(u).Error; err != nil {
		return err
	}
	return nil
}

// HasPassword 是否设置了密码, 仅通过第三方登录注册的用户没有密码
func (u *User) HasPassword() bool {
	return u.PasswordHash != ""
}

// UpdateLastLogin 更新最后登录时间
func (u *User) UpdateLastLogin(tx *gorm.DB) error {
	now := time.Now()
	u.LastLoginAt = &now
	return tx.Model(u).Update("last_login_at", now).Error
}

// CheckDisabled 检查用户账户是否被停用,已停用则返回错误
func (u *User) CheckDisabled() error {
	if u.IsDisabled {
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

// UserIdentity 用户绑定的外部身份(第三方登录), 一个用户可绑定多个
//
// 联合索引：
//   - uniq_user_identities_provider_uid (provider, provider_user_id)：同一个外部账号只能绑定一个用户
type UserIdentity struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	UserID         uint64    `gorm:"not null;index" json:"user_id"`
	Provider       string    `gorm:"size:32;not null;uniqueIndex:uniq_user_identities_provider_uid" json:"provider"`
	ProviderUserID string    `gorm:"size:255;not null;uniqueIndex:uniq_user_identities_provider_uid" json:"provider_user_id"`
	CreatedAt      time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (UserIdentity) TableName() string { return "user_identities" }
