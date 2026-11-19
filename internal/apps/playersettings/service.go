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
package playersettings

import (
	"context"
	"fmt"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/fault"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
)

func init() { audit.RegisterAction(ActionUpdate, "修改游戏设置") }

// Read 未保存过的账号使用默认值, 读取不创建数据库行。
func Read(tx *gorm.DB, userID uint64) (*Settings, error) {
	var result Settings
	if err := tx.Where("user_id = ?", userID).Limit(1).Find(&result).Error; err != nil {
		return nil, err
	}
	result.UserID = userID
	return &result, nil
}

// Lock 修改设置和出售共享这把事务锁, 包括还没有设置行的账号。
// 必须在物品、背包和钱包锁之前取得, 保证开启保护与出售具有确定顺序。
func Lock(tx *gorm.DB, userID uint64) error {
	return db.Lock(tx, fmt.Sprintf("player_settings:%d", userID))
}

func Update(ctx context.Context, userID uint64, req *UpdateRequest) (*Settings, bool, error) {
	var result *Settings
	changed := false
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := Lock(tx, userID); err != nil {
			return err
		}
		current, err := Read(tx, userID)
		if err != nil {
			return err
		}
		if current.Version != *req.Version {
			return fault.Conflict(Changed)
		}
		result = current
		if current.KeepOneCollectible == *req.KeepOneCollectible {
			return nil
		}
		current.KeepOneCollectible = *req.KeepOneCollectible
		current.Version++
		now := time.Now()
		current.UpdatedAt = &now
		changed = true
		return tx.Save(current).Error
	})
	return result, changed, err
}
