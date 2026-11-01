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

package codexadmin

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/codex"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// QuerySets 获取收集组列表, 按排序值和 ID 排列
func QuerySets(ctx context.Context, req *ListSetsRequest) (int64, []codex.Set, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&codex.Set{})
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
	sets := []codex.Set{}
	if err := query.Order("sort_order, id").Offset(offset).Limit(req.Size).Find(&sets).Error; err != nil {
		return 0, nil, err
	}
	return total, sets, nil
}

// doCreateSet 校验并新建收集组
func doCreateSet(ctx context.Context, req *SetRequest) (*codex.Set, error) {
	target := &codex.Set{}
	ApplySet(target, req)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := codex.Validate(tx, target, nil); err != nil {
			return err
		}
		return tx.Create(target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(codex.CodeTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateSet 校验并修改收集组, 返回修改后的收集组和本次修改的字段, 没有任何字段变化时不写库
func doUpdateSet(ctx context.Context, setID uint64, req *SetRequest) (*codex.Set, map[string]interface{}, error) {
	var (
		target  codex.Set
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockSet(tx, &target, setID); err != nil {
			return err
		}
		if req.Code != target.Code {
			return errors.New(codex.CodeImmutable)
		}
		old := target
		ApplySet(&target, req)
		if err := codex.Validate(tx, &target, old.ItemIDs); err != nil {
			return err
		}
		changes = setChanges(&old, &target)
		if len(changes) == 0 {
			return nil
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return &target, changes, nil
}

// doDeleteSet 软删除收集组, 返回被删除的收集组
func doDeleteSet(ctx context.Context, setID uint64) (*codex.Set, error) {
	var target codex.Set
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockSet(tx, &target, setID); err != nil {
			return err
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// LockSet 锁定收集组, 防止并发修改、删除和领取互相覆盖
func LockSet(tx *gorm.DB, target *codex.Set, setID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), setID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(codex.SetNotFound)
		}
		return err
	}
	return nil
}

// ApplySet 写入收集组字段, 由 codex.Validate 校验
func ApplySet(target *codex.Set, req *SetRequest) {
	target.Code = req.Code
	target.Name = strings.TrimSpace(req.Name)
	target.Description = strings.TrimSpace(req.Description)
	target.ItemIDs = req.ItemIDs
	target.Reward = req.Reward
	target.SortOrder = req.SortOrder
	target.IsEnabled = req.IsEnabled
}

// setFields 收集组可修改的字段, 用于操作日志
func setFields(target *codex.Set) map[string]interface{} {
	return map[string]interface{}{
		"code":        target.Code,
		"name":        target.Name,
		"description": target.Description,
		"item_ids":    target.ItemIDs,
		"reward":      target.Reward,
		"sort_order":  target.SortOrder,
		"is_enabled":  target.IsEnabled,
	}
}

// setChanges 本次修改了哪些字段, 用于操作日志; 物品和奖励中有切片, 按内容比较
func setChanges(old, next *codex.Set) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := setFields(next)
	for key, oldValue := range setFields(old) {
		if newValue := nextFields[key]; !reflect.DeepEqual(oldValue, newValue) {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}
