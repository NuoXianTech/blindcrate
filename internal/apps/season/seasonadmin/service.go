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

package seasonadmin

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// QuerySeasons 获取赛季列表, 按开始时间从晚到早排列
func QuerySeasons(ctx context.Context, req *ListSeasonsRequest) (int64, []season.View, error) {
	offset := (req.Current - 1) * req.Size
	now := time.Now()

	query := db.DB(ctx).Model(&season.Season{})
	if req.Keyword != "" {
		keyword := "%" + db.EscapeLike(req.Keyword) + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", keyword, keyword)
	}
	// 与 season.Season.StatusAt 一致
	switch req.Status {
	case season.StatusUpcoming:
		query = query.Where("settled_at IS NULL AND starts_at > ?", now)
	case season.StatusActive:
		query = query.Where("settled_at IS NULL AND starts_at <= ? AND ends_at > ?", now, now)
	case season.StatusEnded:
		query = query.Where("settled_at IS NULL AND ends_at <= ?", now)
	case season.StatusSettled:
		query = query.Where("settled_at IS NOT NULL")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var seasons []season.Season
	if err := query.Order("starts_at DESC, id DESC").Offset(offset).Limit(req.Size).Find(&seasons).Error; err != nil {
		return 0, nil, err
	}
	views := make([]season.View, len(seasons))
	for i := range seasons {
		views[i] = season.NewView(&seasons[i], now)
	}
	return total, views, nil
}

// doCreateSeason 新建赛季, 时间不能与其他赛季重叠
func doCreateSeason(ctx context.Context, req *CreateSeasonRequest) (*season.View, error) {
	now := time.Now()
	target := &season.Season{
		Code:        req.Code,
		Name:        strings.TrimSpace(req.Name),
		StartsAt:    req.StartsAt.Truncate(time.Minute),
		EndsAt:      req.EndsAt.Truncate(time.Minute),
		RatingCarry: req.RatingCarry,
	}
	if err := season.Validate(target); err != nil {
		return nil, err
	}
	if !target.EndsAt.After(now) {
		return nil, errors.New(season.TimeInvalid)
	}
	rewards, err := season.NormalizeRewards(req.Rewards)
	if err != nil {
		return nil, err
	}
	target.Rewards = rewards
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := season.CheckRewards(tx, target.Rewards); err != nil {
			return err
		}
		if err := season.Lock(tx); err != nil {
			return err
		}
		if err := season.CheckOverlap(tx, target); err != nil {
			return err
		}
		return tx.Create(target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(season.CodeTaken)
		}
		return nil, err
	}
	view := season.NewView(target, now)
	return &view, nil
}

// doUpdateSeason 修改赛季, 返回修改后的赛季和本次修改的字段, 没有任何字段变化时不写库
//
// 时间按分钟比较, 与原来在同一分钟内视为没有修改; 赛季开始后不能修改开始时间和段位分继承比例, 结束后只能修改名称。
func doUpdateSeason(ctx context.Context, seasonID uint64, req *SeasonRequest) (*season.View, map[string]interface{}, error) {
	var (
		target  season.Season
		changes map[string]interface{}
		now     = time.Now()
	)
	rewards, err := season.NormalizeRewards(req.Rewards)
	if err != nil {
		return nil, nil, err
	}
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := season.Lock(tx); err != nil {
			return err
		}
		if err := lockSeason(tx, &target, seasonID); err != nil {
			return err
		}
		old := target
		startsAt, endsAt := req.StartsAt.Truncate(time.Minute), req.EndsAt.Truncate(time.Minute)
		startChanged := !startsAt.Equal(target.StartsAt.Truncate(time.Minute))
		endChanged := !endsAt.Equal(target.EndsAt.Truncate(time.Minute))
		carryChanged := req.RatingCarry != target.RatingCarry
		rewardsChanged := !reflect.DeepEqual(rewards, target.Rewards)
		switch target.StatusAt(now) {
		case season.StatusUpcoming:
		case season.StatusActive:
			if startChanged || carryChanged {
				return errors.New(season.StartLocked)
			}
		default:
			if startChanged || endChanged || carryChanged || rewardsChanged {
				return errors.New(season.SeasonEnded)
			}
		}

		target.Name = strings.TrimSpace(req.Name)
		target.RatingCarry = req.RatingCarry
		if rewardsChanged {
			// 没有修改时不重新检查, 奖励中的物品删除后仍能修改其他字段
			if err := season.CheckRewards(tx, rewards); err != nil {
				return err
			}
			target.Rewards = rewards
		}
		if startChanged {
			target.StartsAt = startsAt
		}
		if endChanged {
			if !endsAt.After(now) {
				return errors.New(season.TimeInvalid)
			}
			target.EndsAt = endsAt
		}
		if err := season.Validate(&target); err != nil {
			return err
		}
		changes = seasonChanges(&old, &target)
		if len(changes) == 0 {
			return nil
		}
		if startChanged || endChanged {
			if err := season.CheckOverlap(tx, &target); err != nil {
				return err
			}
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		return nil, nil, err
	}
	view := season.NewView(&target, now)
	return &view, changes, nil
}

// doDeleteSeason 删除未开始的赛季
func doDeleteSeason(ctx context.Context, seasonID uint64) (*season.Season, error) {
	var target season.Season
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSeason(tx, &target, seasonID); err != nil {
			return err
		}
		if target.StatusAt(time.Now()) != season.StatusUpcoming {
			return errors.New(season.DeleteStarted)
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// doEndSeason 提前结束进行中的赛季, 结束时间改为当前时间, 返回结束后的赛季和修改的字段
func doEndSeason(ctx context.Context, seasonID uint64) (*season.View, map[string]interface{}, error) {
	var (
		target  season.Season
		changes map[string]interface{}
		now     = time.Now()
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSeason(tx, &target, seasonID); err != nil {
			return err
		}
		if target.StatusAt(now) != season.StatusActive {
			return errors.New(season.EndNotActive)
		}
		old := target
		target.EndsAt = now
		changes = seasonChanges(&old, &target)
		return tx.Save(&target).Error
	})
	if err != nil {
		return nil, nil, err
	}
	view := season.NewView(&target, now)
	return &view, changes, nil
}

// lockSeason 锁定赛季, 防止并发修改、删除和结束互相覆盖
func lockSeason(tx *gorm.DB, target *season.Season, seasonID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), seasonID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(season.SeasonNotFound)
		}
		return err
	}
	return nil
}

// seasonFields 赛季可修改的字段, 用于操作日志; 时间统一为 UTC 的 RFC3339 格式, 便于比较
func seasonFields(target *season.Season) map[string]interface{} {
	return map[string]interface{}{
		"code":         target.Code,
		"name":         target.Name,
		"starts_at":    target.StartsAt.UTC().Format(time.RFC3339),
		"ends_at":      target.EndsAt.UTC().Format(time.RFC3339),
		"rating_carry": target.RatingCarry,
		"rewards":      target.Rewards,
	}
}

// seasonChanges 本次修改了哪些字段, 用于操作日志; 奖励是切片, 按内容比较
func seasonChanges(old, next *season.Season) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := seasonFields(next)
	for key, oldValue := range seasonFields(old) {
		if newValue := nextFields[key]; !reflect.DeepEqual(oldValue, newValue) {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}
