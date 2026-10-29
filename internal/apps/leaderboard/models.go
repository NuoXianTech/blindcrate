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

package leaderboard

import (
	"time"

	"gorm.io/gorm"
)

// Board 排行榜, 按某个统计范围、周期和指标排名
type Board struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"size:64;not null;uniqueIndex:uniq_leaderboards_code" json:"code"` // 自然键, 如 ranked_wins
	Name        string    `gorm:"size:64;not null" json:"name"`
	Description string    `gorm:"size:255;not null;default:''" json:"description"`
	Metric      string    `gorm:"size:64;not null" json:"metric"`        // 排名的指标, 如 wins、profit:chip
	Scope       string    `gorm:"size:64;not null" json:"scope"`         // 统计范围, 如 all、mode:ranked
	Period      string    `gorm:"size:16;not null" json:"period"`        // 统计周期, total、day 或 season
	Size        int32     `gorm:"not null;default:100" json:"size"`      // 展示前多少名
	MinMatches  int64     `gorm:"not null;default:0" json:"min_matches"` // 同一范围和周期内至少完成多少局才上榜
	SortOrder   int32     `gorm:"not null;default:0" json:"sort_order"`
	IsEnabled   bool      `gorm:"not null;default:false" json:"is_enabled"` // 停用后玩家看不到
	CreatedAt   time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Board) TableName() string { return "leaderboards" }

// Exact 按 ID 获取榜单
func (b *Board) Exact(tx *gorm.DB, id uint64) error {
	if err := tx.Where("id = ?", id).First(b).Error; err != nil {
		return err
	}
	return nil
}

// ExactCode 按代码获取榜单
func (b *Board) ExactCode(tx *gorm.DB, code string) error {
	if err := tx.Where("code = ?", code).First(b).Error; err != nil {
		return err
	}
	return nil
}
