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

package leaderboardadmin

import (
	"context"
	"errors"
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// QueryLeaderboards 获取榜单列表, 按排序值排列
func QueryLeaderboards(ctx context.Context, req *ListLeaderboardsRequest) (int64, []leaderboard.Board, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&leaderboard.Board{})
	if req.Keyword != "" {
		keyword := "%" + db.EscapeLike(req.Keyword) + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", keyword, keyword)
	}
	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	boards := []leaderboard.Board{}
	if err := query.Order("sort_order, id").Offset(offset).Limit(req.Size).Find(&boards).Error; err != nil {
		return 0, nil, err
	}
	return total, boards, nil
}

// doCreateLeaderboard 新建榜单
func doCreateLeaderboard(ctx context.Context, req *LeaderboardRequest) (*leaderboard.Board, error) {
	target := &leaderboard.Board{}
	tx := db.DB(ctx)
	if err := ApplyLeaderboard(tx, target, req); err != nil {
		return nil, err
	}
	if err := tx.Create(target).Error; err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(leaderboard.CodeTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateLeaderboard 修改榜单, 返回修改后的榜单和本次修改的字段, 没有任何字段变化时不写库
//
// 修改后缓存的前 N 名随即失效。
func doUpdateLeaderboard(ctx context.Context, boardID uint64, req *LeaderboardRequest) (*leaderboard.Board, map[string]interface{}, error) {
	var (
		target  leaderboard.Board
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockLeaderboard(tx, &target, boardID); err != nil {
			return err
		}
		old := target
		if err := ApplyLeaderboard(tx, &target, req); err != nil {
			return err
		}
		changes = leaderboardChanges(&old, &target)
		if len(changes) == 0 {
			return nil
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, nil, errors.New(leaderboard.CodeTaken)
		}
		return nil, nil, err
	}
	return &target, changes, nil
}

// doDeleteLeaderboard 删除榜单, 统计数据不受影响
func doDeleteLeaderboard(ctx context.Context, boardID uint64) (*leaderboard.Board, error) {
	var target leaderboard.Board
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockLeaderboard(tx, &target, boardID); err != nil {
			return err
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// LockLeaderboard 锁定榜单, 防止并发修改和删除互相覆盖
func LockLeaderboard(tx *gorm.DB, target *leaderboard.Board, boardID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), boardID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(leaderboard.BoardNotFound)
		}
		return err
	}
	return nil
}

// ApplyLeaderboard 写入榜单字段并校验
func ApplyLeaderboard(tx *gorm.DB, target *leaderboard.Board, req *LeaderboardRequest) error {
	target.Code = req.Code
	target.Name = strings.TrimSpace(req.Name)
	target.Description = strings.TrimSpace(req.Description)
	target.Metric = req.Metric
	target.Scope = req.Scope
	target.Period = req.Period
	target.Size = req.Size
	target.MinMatches = req.MinMatches
	target.SortOrder = req.SortOrder
	target.IsEnabled = req.IsEnabled
	return leaderboard.Validate(tx, target)
}

// leaderboardFields 榜单可修改的字段, 用于操作日志
func leaderboardFields(target *leaderboard.Board) map[string]interface{} {
	return map[string]interface{}{
		"code":        target.Code,
		"name":        target.Name,
		"description": target.Description,
		"metric":      target.Metric,
		"scope":       target.Scope,
		"period":      target.Period,
		"size":        target.Size,
		"min_matches": target.MinMatches,
		"sort_order":  target.SortOrder,
		"is_enabled":  target.IsEnabled,
	}
}

// leaderboardChanges 本次修改了哪些字段, 用于操作日志
func leaderboardChanges(old, next *leaderboard.Board) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := leaderboardFields(next)
	for key, oldValue := range leaderboardFields(old) {
		if newValue := nextFields[key]; oldValue != newValue {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}
