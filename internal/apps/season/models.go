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

package season

import (
	"time"

	"gorm.io/gorm"
)

// Season 赛季, 时间与其他赛季不重叠
type Season struct {
	ID          uint64     `gorm:"primaryKey" json:"id"`
	Code        string     `gorm:"size:24;not null;uniqueIndex:uniq_seasons_code" json:"code"` // 自然键, 拼进统计周期 season:<code>, 创建后不能修改
	Name        string     `gorm:"size:64;not null" json:"name"`
	StartsAt    time.Time  `gorm:"not null;index:idx_seasons_starts_at" json:"starts_at"`                                      // 包含
	EndsAt      time.Time  `gorm:"not null" json:"ends_at"`                                                                    // 不包含
	SettledAt   *time.Time `json:"settled_at"`                                                                                 // 赛季奖励发完的时间, 没有发完时为空
	RatingCarry int16      `gorm:"not null;check:chk_seasons_rating_carry,rating_carry BETWEEN 0 AND 100" json:"rating_carry"` // 段位分继承上一个赛季的百分比, 初始分 = 基础分 + (上赛季的段位分 - 基础分) × 比例; 0 有效, 模型中不设默认值
	CreatedAt   time.Time  `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Season) TableName() string { return "seasons" }

// Exact 按 ID 获取赛季
func (s *Season) Exact(tx *gorm.DB, id uint64) error {
	if err := tx.Where("id = ?", id).First(s).Error; err != nil {
		return err
	}
	return nil
}

// StatusAt 赛季在 now 时的状态
func (s *Season) StatusAt(now time.Time) string {
	switch {
	case s.SettledAt != nil:
		return StatusSettled
	case now.Before(s.StartsAt):
		return StatusUpcoming
	case now.Before(s.EndsAt):
		return StatusActive
	default:
		return StatusEnded
	}
}

// View 赛季和当前的状态
type View struct {
	Season
	Status string `json:"status"` // upcoming、active、ended 或 settled
}

// NewView 赛季和 now 时的状态
func NewView(s *Season, now time.Time) View {
	return View{Season: *s, Status: s.StatusAt(now)}
}
