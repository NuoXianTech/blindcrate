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

package migrator

import (
	"context"
	"crypto/rand"
	"log"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
)

const (
	defaultAdminUsername = "admin"
	defaultAdminNickname = "Administrator"
)

// seedAdmin 首次初始化(users 表为空)时创建默认管理员, 并把账号密码打印到日志
//
// 表里只要已有任意用户就跳过, 因此重启不会重复创建, 也不会重置密码。
func seedAdmin() {
	err := db.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		// 事务级咨询锁, 防止多实例同时启动时重复创建
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext('blindcrate:seed_admin'))").Error; err != nil {
			return err
		}

		var count int64
		if err := tx.Model(&auth.User{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}

		password := rand.Text()
		hash, err := auth.HashPassword(password)
		if err != nil {
			return err
		}
		admin := auth.User{
			Username:     defaultAdminUsername,
			PasswordHash: hash,
			Nickname:     defaultAdminNickname,
			IsActive:     true,
			IsAdmin:      true,
		}
		if err = tx.Create(&admin).Error; err != nil {
			return err
		}

		log.Printf("[PostgreSQL] default admin created, username=%s password=%s\n", admin.Username, password)
		return nil
	})
	if err != nil {
		log.Fatalf("[PostgreSQL] seed admin failed: %v\n", err)
	}
}
