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

package item

import (
	"time"

	"gorm.io/gorm"
)

// Item 物品定义
//
// 删除为软删除, ID 不会被新物品复用; 已删除的物品在背包中按占位物品处理, 见 Resolve。
type Item struct {
	ID            uint64         `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"size:64;not null;uniqueIndex:uniq_items_name,where:deleted_at IS NULL" json:"name"`
	Description   string         `gorm:"size:255;not null;default:''" json:"description"`
	Quality       int16          `gorm:"not null;default:1;check:chk_items_quality,quality BETWEEN 1 AND 7" json:"quality"` // 见 constants.go
	ImageURL      string         `gorm:"size:1024;not null;default:''" json:"image_url"`
	SellCurrency  string         `gorm:"size:32;not null;default:'chip'" json:"sell_currency"`
	SellPrice     int64          `gorm:"not null;default:0;check:chk_items_sell_price,sell_price >= 0" json:"sell_price"` // 0 表示不可卖出
	Weight        int32          `gorm:"not null;default:0;check:chk_items_weight,weight >= 0" json:"weight"`             // 掉落权重, 0 表示不会掉落
	Width         int16          `gorm:"not null;default:1;check:chk_items_width,width >= 1" json:"width"`                // 在保险柜中占的格数
	Height        int16          `gorm:"not null;default:1;check:chk_items_height,height >= 1" json:"height"`
	IsEnabled     bool           `gorm:"not null;default:false" json:"is_enabled"` // 下架后不会掉落, 也不能再发放
	IsPlaceholder bool           `gorm:"-" json:"is_placeholder,omitempty"`        // 是否为占位物品, 只由 Resolve 生成
	CreatedAt     time.Time      `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 自定义表名
func (Item) TableName() string { return "items" }

// Exact 按 ID 获取未删除的物品
func (i *Item) Exact(tx *gorm.DB, id uint64) error {
	if err := tx.Where("id = ?", id).First(i).Error; err != nil {
		return err
	}
	return nil
}
