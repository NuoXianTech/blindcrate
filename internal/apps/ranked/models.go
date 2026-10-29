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

package ranked

import (
	"time"

	"gorm.io/gorm"
)

// Rating 用户在一个赛季中的段位分, 第一次结算排位时创建
type Rating struct {
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
	SeasonID  uint64    `gorm:"primaryKey;autoIncrement:false;index:idx_ranked_ratings_board,priority:1" json:"season_id"` // 赛季之外为 0
	Rating    int32     `gorm:"not null;index:idx_ranked_ratings_board,priority:2,sort:desc" json:"rating"`
	Peak      int32     `gorm:"not null" json:"peak"`              // 本赛季的最高分
	Matches   int32     `gorm:"not null;default:0" json:"matches"` // 本赛季计分的排位局数
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Rating) TableName() string { return "ranked_ratings" }

// Tier 段位, 段位分不低于分数线时属于分数线最高的那个段位; 低于最低的分数线时没有段位
type Tier struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:32;not null" json:"name"`
	MinRating int32     `gorm:"not null;uniqueIndex:uniq_ranked_tiers_min_rating;check:chk_ranked_tiers_min_rating,min_rating >= 0" json:"min_rating"` // 分数线, 包含
	Color     string    `gorm:"size:7;not null" json:"color"`                                                                                          // #rrggbb
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Tier) TableName() string { return "ranked_tiers" }

// Exact 按 ID 获取段位
func (t *Tier) Exact(tx *gorm.DB, id uint64) error {
	if err := tx.Where("id = ?", id).First(t).Error; err != nil {
		return err
	}
	return nil
}
