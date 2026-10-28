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

package loot

import (
	"time"

	"gorm.io/gorm"
)

// Pool 掉落池, 玩法按代码引用
type Pool struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"size:64;not null;uniqueIndex:uniq_loot_pools_code" json:"code"` // 玩法中引用的代码, 如 crate_basic
	Name        string    `gorm:"size:64;not null" json:"name"`
	Description string    `gorm:"size:255;not null;default:''" json:"description"`
	IsEnabled   bool      `gorm:"not null;default:false" json:"is_enabled"` // 停用后不能再抽取
	CreatedAt   time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Pool) TableName() string { return "loot_pools" }

// Exact 按 ID 获取掉落池
func (p *Pool) Exact(tx *gorm.DB, id uint64) error {
	if err := tx.Where("id = ?", id).First(p).Error; err != nil {
		return err
	}
	return nil
}

// ExactCode 按代码获取掉落池
func (p *Pool) ExactCode(tx *gorm.DB, code string) error {
	if err := tx.Where("code = ?", code).First(p).Error; err != nil {
		return err
	}
	return nil
}

// Entry 掉落池条目, 每行掉落一种物品或货币; 修改时整池替换, 不单独更新
type Entry struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	PoolID      uint64    `gorm:"not null;index:idx_loot_pool_entries_pool_id" json:"pool_id"`
	ItemID      uint64    `gorm:"not null;default:0;check:chk_loot_pool_entries_target,(item_id <> 0) <> (currency <> '')" json:"item_id"`                        // 掉落的物品, 与 Currency 二选一
	Currency    string    `gorm:"size:32;not null;default:''" json:"currency"`                                                                                    // 掉落的货币
	Weight      int32     `gorm:"not null;check:chk_loot_pool_entries_weight,weight > 0" json:"weight"`                                                           // 抽中的概率为权重 / 池中可抽取条目的权重之和
	MinQuantity int64     `gorm:"not null;default:1;check:chk_loot_pool_entries_quantity,min_quantity >= 1 AND max_quantity >= min_quantity" json:"min_quantity"` // 每次掉落的数量在该范围内均匀随机, 货币为金额
	MaxQuantity int64     `gorm:"not null;default:1" json:"max_quantity"`
	CreatedAt   time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (Entry) TableName() string { return "loot_pool_entries" }
