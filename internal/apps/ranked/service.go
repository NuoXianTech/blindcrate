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
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	condition.Register(TypeMinRating, "段位分", "当前赛季的段位分不少于指定值, 本赛季还没打过排位时按初始分计算", checkMinRating)
}

// colorPattern 段位颜色, #rrggbb 小写
var colorPattern = regexp.MustCompile(`^#[0-9a-f]{6}$`)

// MaxTierRating 分数线的上限
const MaxTierRating = 1000000

// Result 一局排位中一个真人的名次
type Result struct {
	UserID uint64
	Rank   int16
}

// Change 一个真人本局段位分的变化
type Change struct {
	Rating int32 // 结算后的段位分
	Delta  int32 // 实际的变化, 扣到 0 为止时小于配置的扣分
}

// Settle 按名次结算一局排位的段位分, 在结算对局的事务中调用, 返回用户 ID -> 变化
//
// at 为结算时间, 按它所在的赛季记分; 没有配置加减分的名次不加不减, 但仍计一局。
func Settle(tx *gorm.DB, at time.Time, results []Result) (map[uint64]Change, error) {
	cfg, err := settings.Ranked.Get(tx.Statement.Context)
	if err != nil {
		return nil, err
	}
	current, err := season.Current(tx, at)
	if err != nil {
		return nil, err
	}
	// 按用户 ID 加锁, 避免并发结算时互相等待
	results = slices.Clone(results)
	slices.SortFunc(results, func(a, b Result) int { return cmp.Compare(a.UserID, b.UserID) })

	changes := make(map[uint64]Change, len(results))
	for _, r := range results {
		rating, err := lock(tx, r.UserID, current, int32(cfg.BaseRating))
		if err != nil {
			return nil, err
		}
		next := apply(rating.Rating, deltaOf(cfg.RankDeltas, r.Rank))
		changes[r.UserID] = Change{Rating: next, Delta: next - rating.Rating}
		if err := tx.Model(&Rating{}).
			Where("user_id = ? AND season_id = ?", rating.UserID, rating.SeasonID).
			Updates(map[string]interface{}{
				"rating":  next,
				"peak":    max(rating.Peak, next),
				"matches": gorm.Expr("matches + 1"),
			}).Error; err != nil {
			return nil, err
		}
	}
	return changes, nil
}

// View 用户在当前赛季的段位分
type View struct {
	Season   *season.View `json:"season"` // 进行中的赛季, 赛季之外为空
	Rating   int32        `json:"rating"`
	Peak     int32        `json:"peak"`
	Matches  int32        `json:"matches"`   // 本赛季计分的排位局数, 0 表示还没打过, 段位分为初始分
	Tier     *Tier        `json:"tier"`      // 当前的段位, 低于最低的分数线时为空
	NextTier *Tier        `json:"next_tier"` // 下一个段位, 已是最高段位时为空
}

// Query 查询用户在 now 时所在赛季的段位分, 还没打过排位时为初始分
func Query(tx *gorm.DB, userID uint64, now time.Time) (*View, error) {
	cfg, err := settings.Ranked.Get(tx.Statement.Context)
	if err != nil {
		return nil, err
	}
	current, err := season.Current(tx, now)
	if err != nil {
		return nil, err
	}
	view := &View{}
	if current != nil {
		seasonView := season.NewView(current, now)
		view.Season = &seasonView
	}
	rating, err := find(tx, userID, seasonIDOf(current))
	if err != nil {
		return nil, err
	}
	if rating != nil {
		view.Rating, view.Peak, view.Matches = rating.Rating, rating.Peak, rating.Matches
	} else {
		if view.Rating, err = initialRating(tx, userID, current, int32(cfg.BaseRating)); err != nil {
			return nil, err
		}
		view.Peak = view.Rating
	}
	tiers, err := QueryTiers(tx)
	if err != nil {
		return nil, err
	}
	view.Tier, view.NextTier = TierOf(tiers, view.Rating)
	return view, nil
}

// ValidateTier 保存前校验段位, 名称需已去掉首尾空白, 颜色需已转为小写
func ValidateTier(t *Tier) error {
	if n := utf8.RuneCountInString(t.Name); n < 1 || n > 32 {
		return errors.New(TierNameInvalid)
	}
	if t.MinRating < 0 || t.MinRating > MaxTierRating {
		return errors.New(TierRatingInvalid)
	}
	if !colorPattern.MatchString(t.Color) {
		return errors.New(TierColorInvalid)
	}
	return nil
}

// QueryTiers 获取全部段位, 按分数线从低到高
func QueryTiers(tx *gorm.DB) ([]Tier, error) {
	tiers := []Tier{}
	if err := tx.Order("min_rating").Find(&tiers).Error; err != nil {
		return nil, err
	}
	return tiers, nil
}

// TierOf 段位分所在的段位和下一个段位, tiers 需按分数线从低到高排列
//
// 低于最低的分数线时当前段位为 nil, 已是最高段位时下一个段位为 nil。
func TierOf(tiers []Tier, rating int32) (*Tier, *Tier) {
	// 第一个分数线高于段位分的段位是下一个段位, 它前面的是当前段位
	i, _ := slices.BinarySearchFunc(tiers, rating+1, func(t Tier, target int32) int { return cmp.Compare(t.MinRating, target) })
	var current, next *Tier
	if i > 0 {
		current = &tiers[i-1]
	}
	if i < len(tiers) {
		next = &tiers[i]
	}
	return current, next
}

// lock 锁定用户在赛季中的段位分, 还没有时按初始分创建
func lock(tx *gorm.DB, userID uint64, current *season.Season, base int32) (*Rating, error) {
	seasonID := seasonIDOf(current)
	locking := tx.Clauses(clause.Locking{Strength: "UPDATE"})
	rating, err := find(locking, userID, seasonID)
	if err != nil || rating != nil {
		return rating, err
	}
	initial, err := initialRating(tx, userID, current, base)
	if err != nil {
		return nil, err
	}
	created := &Rating{UserID: userID, SeasonID: seasonID, Rating: initial, Peak: initial}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(created).Error; err != nil {
		return nil, err
	}
	return find(locking, userID, seasonID)
}

// find 获取用户在赛季中的段位分, 没有时返回 nil
func find(tx *gorm.DB, userID, seasonID uint64) (*Rating, error) {
	var ratings []Rating
	if err := tx.Where("user_id = ? AND season_id = ?", userID, seasonID).Limit(1).Find(&ratings).Error; err != nil {
		return nil, err
	}
	if len(ratings) == 0 {
		return nil, nil
	}
	return &ratings[0], nil
}

// initialRating 用户在赛季中的初始分: 在上一个赛季有段位分时按赛季的继承比例向基础分靠拢, 否则为基础分; 赛季之外为基础分
func initialRating(tx *gorm.DB, userID uint64, current *season.Season, base int32) (int32, error) {
	if current == nil {
		return base, nil
	}
	previous, err := season.Previous(tx, current)
	if err != nil || previous == nil {
		return base, err
	}
	last, err := find(tx, userID, previous.ID)
	if err != nil || last == nil {
		return base, err
	}
	return carry(base, last.Rating, current.RatingCarry), nil
}

// seasonIDOf 记分的赛季 ID, 赛季之外为 0
func seasonIDOf(current *season.Season) uint64 {
	if current == nil {
		return 0
	}
	return current.ID
}

// carry 按百分比继承上赛季的段位分: 基础分 + (上赛季的段位分 - 基础分) × 比例, 不低于 0
func carry(base, last int32, percent int16) int32 {
	return max(0, base+(last-base)*int32(percent)/100)
}

// deltaOf 名次对应的加减分, 没有配置的名次为 0
func deltaOf(deltas []int, rank int16) int32 {
	if rank < 1 || int(rank) > len(deltas) {
		return 0
	}
	return int32(deltas[rank-1])
}

// apply 加减后的段位分, 扣到 0 为止
func apply(rating, delta int32) int32 {
	return max(0, rating+delta)
}

// minRatingParams 段位分的参数
type minRatingParams struct {
	Rating int64 `json:"rating" label:"段位分" binding:"min=1,max=1000000"`
}

func checkMinRating(tx *gorm.DB, userID uint64, p *minRatingParams) (*condition.Progress, error) {
	view, err := Query(tx, userID, time.Now())
	if err != nil {
		return nil, err
	}
	current := int64(view.Rating)
	return &condition.Progress{
		Met:     current >= p.Rating,
		Current: current,
		Target:  p.Rating,
		Text:    fmt.Sprintf("段位分达到 %d(当前 %d)", p.Rating, current),
	}, nil
}
