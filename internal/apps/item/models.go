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
	Quality       int16          `gorm:"not null;default:1" json:"quality"`                      // 品质等级, 见 Quality
	Category      string         `gorm:"size:32;not null;default:'collectible'" json:"category"` // 类别代码, 见 RegisterCategory
	ImageURL      string         `gorm:"size:1024;not null;default:''" json:"image_url"`
	SellCurrency  string         `gorm:"size:32;not null;default:'chip'" json:"sell_currency"`
	SellPrice     int64          `gorm:"not null;default:0;check:chk_items_sell_price,sell_price >= 0" json:"sell_price"` // 0 表示不可卖出
	Width         int16          `gorm:"not null;default:1;check:chk_items_width,width >= 1" json:"width"`                // 在仓库中占的格数
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

// Category 物品类别, 由代码注册, 见 RegisterCategory
type Category struct {
	Code  string `json:"code"`
	Label string `json:"label"` // 后台展示的名称, 如 "藏品"
}

// Quality 物品品质, 新增品质只需插入一行
//
// 等级越大越稀有, 物品、仓库藏品按等级引用品质; 颜色和名称由前端从接口读取。
type Quality struct {
	Level     int16     `gorm:"primaryKey;autoIncrement:false" json:"level"`
	Code      string    `gorm:"size:32;not null;uniqueIndex:uniq_qualities_code" json:"code"` // 自然键, 导入导出时用来对应
	Name      string    `gorm:"size:32;not null" json:"name"`
	Color     string    `gorm:"size:7;not null" json:"color"`              // #rrggbb
	Effect    string    `gorm:"size:16;not null;default:''" json:"effect"` // 特效, 空表示纯色, 见 constants.go
	ScanMs    int32     `gorm:"not null;default:0" json:"scan_ms"`         // 开箱时扫描这一品质藏品的时长, 毫秒
	SortOrder int32     `gorm:"not null;default:0" json:"sort_order"`      // 筛选和图例中从小到大排列
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Quality) TableName() string { return "qualities" }

// Exact 按等级查询
func (q *Quality) Exact(tx *gorm.DB, level int16) error {
	if err := tx.Where("level = ?", level).First(q).Error; err != nil {
		return err
	}
	return nil
}
