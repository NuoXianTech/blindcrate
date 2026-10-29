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

package rankedadmin

import (
	"context"
	"errors"
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/ranked"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// queryRankedTiers 获取全部段位, 按分数线从低到高
func queryRankedTiers(ctx context.Context) ([]ranked.Tier, error) {
	return ranked.QueryTiers(db.DB(ctx))
}

// doCreateRankedTier 新建段位
func doCreateRankedTier(ctx context.Context, req *RankedTierRequest) (*ranked.Tier, error) {
	target := &ranked.Tier{}
	if err := applyRankedTier(target, req); err != nil {
		return nil, err
	}
	if err := db.DB(ctx).Create(target).Error; err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(ranked.TierRatingTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateRankedTier 修改段位, 返回修改后的段位和本次修改的字段, 没有任何字段变化时不写库
func doUpdateRankedTier(ctx context.Context, tierID uint64, req *RankedTierRequest) (*ranked.Tier, map[string]interface{}, error) {
	var (
		target  ranked.Tier
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockRankedTier(tx, &target, tierID); err != nil {
			return err
		}
		old := target
		if err := applyRankedTier(&target, req); err != nil {
			return err
		}
		changes = rankedTierChanges(&old, &target)
		if len(changes) == 0 {
			return nil
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, nil, errors.New(ranked.TierRatingTaken)
		}
		return nil, nil, err
	}
	return &target, changes, nil
}

// doDeleteRankedTier 删除段位, 段位分不受影响
func doDeleteRankedTier(ctx context.Context, tierID uint64) (*ranked.Tier, error) {
	var target ranked.Tier
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockRankedTier(tx, &target, tierID); err != nil {
			return err
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// lockRankedTier 锁定段位, 防止并发修改和删除互相覆盖
func lockRankedTier(tx *gorm.DB, target *ranked.Tier, tierID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), tierID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(ranked.TierNotFound)
		}
		return err
	}
	return nil
}

// applyRankedTier 写入段位字段并校验, 颜色统一为小写
func applyRankedTier(target *ranked.Tier, req *RankedTierRequest) error {
	target.Name = strings.TrimSpace(req.Name)
	target.MinRating = req.MinRating
	target.Color = strings.ToLower(req.Color)
	return ranked.ValidateTier(target)
}

// rankedTierFields 段位可修改的字段, 用于操作日志
func rankedTierFields(target *ranked.Tier) map[string]interface{} {
	return map[string]interface{}{
		"name":       target.Name,
		"min_rating": target.MinRating,
		"color":      target.Color,
	}
}

// rankedTierChanges 本次修改了哪些字段, 用于操作日志
func rankedTierChanges(old, next *ranked.Tier) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := rankedTierFields(next)
	for key, oldValue := range rankedTierFields(old) {
		if newValue := nextFields[key]; oldValue != newValue {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}
