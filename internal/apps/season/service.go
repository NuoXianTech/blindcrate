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
	"cmp"
	"errors"
	"regexp"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"gorm.io/gorm"
)

func init() {
	// 进行中的赛季在结算对局时额外写入赛季周期
	stats.RegisterPeriod(func(tx *gorm.DB, at time.Time) (string, error) {
		current, err := Current(tx, at)
		if err != nil || current == nil {
			return "", err
		}
		return Period(current.Code), nil
	})
}

// codePattern 赛季代码, 小写字母、数字和下划线; 拼上前缀后不能超过 stats.period 的长度
var codePattern = regexp.MustCompile(`^[a-z0-9_]{1,24}$`)

// Period 赛季的统计周期, 如 season:s1
func Period(code string) string {
	return periodPrefix + code
}

// Validate 保存前校验赛季, 名称需已去掉首尾空白; 不检查与其他赛季是否重叠, 见 CheckOverlap
func Validate(s *Season) error {
	if !codePattern.MatchString(s.Code) {
		return errors.New(CodeInvalid)
	}
	if n := utf8.RuneCountInString(s.Name); n < 1 || n > 64 {
		return errors.New(NameInvalid)
	}
	if !s.EndsAt.After(s.StartsAt) {
		return errors.New(TimeInvalid)
	}
	if s.RatingCarry < 0 || s.RatingCarry > 100 {
		return errors.New(CarryInvalid)
	}
	return nil
}

// NormalizeRewards 校验赛季奖励的分数线、档数和数量, 返回按分数线从低到高排列、合并同类项后的奖励
//
// 不查库, 货币和物品是否存在见 CheckRewards; rewards 为 nil 时返回空列表。
func NormalizeRewards(rewards []Reward) ([]Reward, error) {
	if len(rewards) > maxRewards {
		return nil, errors.New(RewardsTooMany)
	}
	normalized := make([]Reward, len(rewards))
	for i, r := range rewards {
		if r.MinRating < 0 || r.MinRating > maxRewardRating {
			return nil, errors.New(RewardRatingInvalid)
		}
		if r.Bundle.IsEmpty() {
			return nil, errors.New(RewardEmpty)
		}
		if len(r.Bundle.Currencies)+len(r.Bundle.Items) > maxRewardLines {
			return nil, errors.New(RewardLinesTooMany)
		}
		for _, line := range r.Bundle.Currencies {
			if line.Amount <= 0 {
				return nil, errors.New(RewardAmountInvalid)
			}
		}
		for _, line := range r.Bundle.Items {
			if line.Quantity <= 0 {
				return nil, errors.New(RewardAmountInvalid)
			}
		}
		bundle, err := r.Bundle.Normalize()
		if err != nil {
			return nil, errors.New(RewardAmountInvalid)
		}
		normalized[i] = Reward{MinRating: r.MinRating, Bundle: bundle}
	}
	slices.SortFunc(normalized, func(a, b Reward) int { return cmp.Compare(a.MinRating, b.MinRating) })
	for i := 1; i < len(normalized); i++ {
		if normalized[i].MinRating == normalized[i-1].MinRating {
			return nil, errors.New(RewardRatingTaken)
		}
	}
	return normalized, nil
}

// CheckRewards 赛季奖励中的货币需存在, 物品需未删除; 已下架的物品在发放时按回收价折成货币
func CheckRewards(tx *gorm.DB, rewards []Reward) error {
	var (
		codes   []string
		itemIDs []uint64
	)
	for _, r := range rewards {
		for _, line := range r.Bundle.Currencies {
			codes = append(codes, line.Currency)
		}
		for _, line := range r.Bundle.Items {
			itemIDs = append(itemIDs, line.ItemID)
		}
	}
	if len(codes) > 0 {
		codes = slices.Compact(slices.Sorted(slices.Values(codes)))
		var count int64
		if err := tx.Model(&wallet.Currency{}).Where("code IN ?", codes).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(codes)) {
			return errors.New(RewardNotFound)
		}
	}
	if len(itemIDs) > 0 {
		itemIDs = slices.Compact(slices.Sorted(slices.Values(itemIDs)))
		var count int64
		if err := tx.Model(&item.Item{}).Where("id IN ?", itemIDs).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(itemIDs)) {
			return errors.New(RewardNotFound)
		}
	}
	return nil
}

// ResolveRewards 赛季奖励中的货币和物品带上名称, 已删除的物品显示为占位物品
func ResolveRewards(tx *gorm.DB, rewards []Reward) ([]RewardView, error) {
	var (
		codes   []string
		itemIDs []uint64
	)
	for _, r := range rewards {
		for _, line := range r.Bundle.Currencies {
			codes = append(codes, line.Currency)
		}
		for _, line := range r.Bundle.Items {
			itemIDs = append(itemIDs, line.ItemID)
		}
	}
	currencies := make(map[string]wallet.Currency, len(codes))
	if len(codes) > 0 {
		var found []wallet.Currency
		if err := tx.Where("code IN ?", codes).Find(&found).Error; err != nil {
			return nil, err
		}
		for _, c := range found {
			currencies[c.Code] = c
		}
	}
	items, err := item.Resolve(tx, itemIDs)
	if err != nil {
		return nil, err
	}

	views := make([]RewardView, len(rewards))
	for i, r := range rewards {
		lines := make([]RewardLine, 0, len(r.Bundle.Currencies)+len(r.Bundle.Items))
		for _, line := range r.Bundle.Currencies {
			c, ok := currencies[line.Currency]
			if !ok {
				c = wallet.Currency{Code: line.Currency, Name: line.Currency}
			}
			lines = append(lines, RewardLine{Currency: c.Code, Name: c.Name, Decimals: c.Decimals, Amount: line.Amount})
		}
		for _, line := range r.Bundle.Items {
			target := items[line.ItemID]
			lines = append(lines, RewardLine{ItemID: line.ItemID, Name: target.Name, ImageURL: target.ImageURL, Amount: line.Quantity})
		}
		views[i] = RewardView{MinRating: r.MinRating, Lines: lines}
	}
	return views, nil
}

// Lock 锁定赛季表直到事务结束, 新建和修改赛季时间前调用, 防止并发保存的两个赛季互相重叠
//
// 只阻塞其他写入, 不影响读取。
func Lock(tx *gorm.DB) error {
	return tx.Exec("LOCK TABLE seasons IN SHARE ROW EXCLUSIVE MODE").Error
}

// CheckOverlap 检查赛季的时间是否与其他赛季重叠, 需要先调用 Lock
func CheckOverlap(tx *gorm.DB, s *Season) error {
	var count int64
	if err := tx.Model(&Season{}).
		Where("id <> ? AND starts_at < ? AND ends_at > ?", s.ID, s.EndsAt, s.StartsAt).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New(TimeOverlapped)
	}
	return nil
}

// Current 获取 at 时进行中的赛季, 没有时返回 nil
func Current(tx *gorm.DB, at time.Time) (*Season, error) {
	return first(tx.Where("starts_at <= ? AND ends_at > ?", at, at).Order("starts_at DESC"))
}

// Latest 获取 at 时最近一个已开始的赛季, 可能已经结束; 没有时返回 nil
//
// 用于赛季排行榜: 赛季之间的间隔中展示刚结束的赛季。
func Latest(tx *gorm.DB, at time.Time) (*Season, error) {
	return first(tx.Where("starts_at <= ?", at).Order("starts_at DESC"))
}

// Next 获取 at 之后最早开始的赛季, 没有时返回 nil
func Next(tx *gorm.DB, at time.Time) (*Season, error) {
	return first(tx.Where("starts_at > ?", at).Order("starts_at"))
}

// Previous 获取 s 之前最近的一个赛季, 没有时返回 nil
func Previous(tx *gorm.DB, s *Season) (*Season, error) {
	return first(tx.Where("starts_at < ?", s.StartsAt).Order("starts_at DESC"))
}

// first 获取查询到的第一个赛季, 没有时返回 nil
func first(query *gorm.DB) (*Season, error) {
	var seasons []Season
	if err := query.Limit(1).Find(&seasons).Error; err != nil {
		return nil, err
	}
	if len(seasons) == 0 {
		return nil, nil
	}
	return &seasons[0], nil
}
