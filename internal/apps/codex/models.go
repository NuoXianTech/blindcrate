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

package codex

import (
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"gorm.io/gorm"
)

// Set 收集组, 集齐 ItemIDs 中全部启用的藏品后可以领取 Reward
//
// 删除为软删除, 领取记录仍然对应得上; 代码在已删除的收集组中也唯一。
type Set struct {
	ID          uint64         `gorm:"primaryKey" json:"id"`
	Code        string         `gorm:"size:32;not null;uniqueIndex:uniq_codex_sets_code" json:"code"` // 自然键, 创建后不能修改
	Name        string         `gorm:"size:64;not null" json:"name"`
	Description string         `gorm:"size:255;not null;default:''" json:"description"`
	ItemIDs     []uint64       `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"item_ids"` // 组内的藏品, 去重后从小到大排列
	Reward      reward.Bundle  `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"reward"`   // 数量为正数, 不能为空
	SortOrder   int32          `gorm:"not null;default:0" json:"sort_order"`                             // 从小到大排列
	IsEnabled   bool           `gorm:"not null;default:false" json:"is_enabled"`                         // 停用后玩家看不到, 也不能领取
	CreatedAt   time.Time      `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 自定义表名
func (Set) TableName() string { return "codex_sets" }

// Exact 按 ID 获取收集组
func (s *Set) Exact(tx *gorm.DB, id uint64) error {
	return tx.Where("id = ?", id).First(s).Error
}

// ClaimRecord 一次领取, 同一收集组同一用户只有一行
type ClaimRecord struct {
	SetID     uint64    `gorm:"primaryKey;autoIncrement:false" json:"set_id"`
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (ClaimRecord) TableName() string { return "codex_claims" }
