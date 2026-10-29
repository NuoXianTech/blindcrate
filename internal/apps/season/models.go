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

	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
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
	Rewards     []Reward   `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"rewards"`                            // 赛季奖励, 按分数线从低到高, 见 NormalizeRewards
	CreatedAt   time.Time  `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// Reward 一档赛季奖励, 赛季中的最高段位分不低于分数线时可以领取; 每人只发满足条件的最高一档
type Reward struct {
	MinRating int32         `json:"min_rating"` // 分数线, 包含
	Bundle    reward.Bundle `json:"bundle"`     // 数量为正数
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

// RewardOf 最高段位分 peak 可以领取的那一档奖励, 一档都不满足时返回 nil; Rewards 需按分数线从低到高排列
func (s *Season) RewardOf(peak int32) *Reward {
	for i := len(s.Rewards) - 1; i >= 0; i-- {
		if s.Rewards[i].MinRating <= peak {
			return &s.Rewards[i]
		}
	}
	return nil
}

// View 赛季和当前的状态
type View struct {
	Season
	Status string `json:"status"` // upcoming、active、ended 或 settled
}

// RewardView 玩家看到的一档赛季奖励
type RewardView struct {
	MinRating int32        `json:"min_rating"`
	Lines     []RewardLine `json:"lines"` // 货币在前, 物品在后
}

// RewardLine 奖励中的一种货币或物品, 带名称便于展示
type RewardLine struct {
	Currency string `json:"currency,omitempty"` // 与 item_id 二选一
	ItemID   uint64 `json:"item_id,omitempty"`
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
	Decimals int16  `json:"decimals"` // 货币的展示用小数位, 物品为 0
	Amount   int64  `json:"amount"`   // 货币金额或物品数量
}

// NewView 赛季和 now 时的状态
func NewView(s *Season, now time.Time) View {
	return View{Season: *s, Status: s.StatusAt(now)}
}
