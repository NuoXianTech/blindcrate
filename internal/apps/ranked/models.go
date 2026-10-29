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

import "time"

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
