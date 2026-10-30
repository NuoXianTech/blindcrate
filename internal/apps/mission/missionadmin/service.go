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

package missionadmin

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/mission"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// QueryMissions 获取任务列表, 按周期、排序值和 ID 排列
func QueryMissions(ctx context.Context, req *ListMissionsRequest) (int64, []mission.Mission, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&mission.Mission{})
	if req.Keyword != "" {
		keyword := "%" + db.EscapeLike(req.Keyword) + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", keyword, keyword)
	}
	if req.Cycle != "" {
		query = query.Where("cycle = ?", req.Cycle)
	}
	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	missions := []mission.Mission{}
	// 周期按玩家页面的页签顺序排列
	order := clause.OrderBy{Expression: clause.Expr{SQL: "CASE cycle WHEN ? THEN 0 WHEN ? THEN 1 ELSE 2 END, sort_order, id", Vars: []any{mission.CycleDaily, mission.CycleWeekly}}}
	if err := query.Clauses(order).Offset(offset).Limit(req.Size).Find(&missions).Error; err != nil {
		return 0, nil, err
	}
	return total, missions, nil
}

// doCreateMission 校验并新建任务
func doCreateMission(ctx context.Context, req *MissionRequest) (*mission.Mission, error) {
	target := &mission.Mission{}
	ApplyMission(target, req)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := mission.Validate(tx, target); err != nil {
			return err
		}
		return tx.Create(target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(mission.CodeTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateMission 校验并修改任务, 返回修改后的任务和本次修改的字段, 没有任何字段变化时不写库
func doUpdateMission(ctx context.Context, missionID uint64, req *MissionRequest) (*mission.Mission, map[string]interface{}, error) {
	var (
		target  mission.Mission
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockMission(tx, &target, missionID); err != nil {
			return err
		}
		if req.Code != target.Code {
			return errors.New(mission.CodeImmutable)
		}
		if err := CheckCycle(&target, req); err != nil {
			return err
		}
		old := target
		ApplyMission(&target, req)
		if err := mission.Validate(tx, &target); err != nil {
			return err
		}
		changes = missionChanges(&old, &target)
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

// doDeleteMission 软删除任务, 返回被删除的任务
func doDeleteMission(ctx context.Context, missionID uint64) (*mission.Mission, error) {
	var target mission.Mission
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockMission(tx, &target, missionID); err != nil {
			return err
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// LockMission 锁定任务, 防止并发修改、删除和领取互相覆盖
func LockMission(tx *gorm.DB, target *mission.Mission, missionID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), missionID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(mission.MissionNotFound)
		}
		return err
	}
	return nil
}

// CheckCycle 已有任务的周期不能修改: 领取记录和幂等键按周期对应的统计周期区分, 改周期后已领取的会被当作未领取
func CheckCycle(target *mission.Mission, req *MissionRequest) error {
	if req.Cycle != target.Cycle {
		return &mission.InvalidError{Reason: mission.CycleImmutable}
	}
	return nil
}

// ApplyMission 写入任务字段, 由 mission.Validate 校验
func ApplyMission(target *mission.Mission, req *MissionRequest) {
	target.Code = req.Code
	target.Name = strings.TrimSpace(req.Name)
	target.Description = strings.TrimSpace(req.Description)
	target.Cycle = req.Cycle
	target.Scope = req.Scope
	target.Metric = req.Metric
	target.Target = req.Target
	target.Reward = req.Reward
	target.Requirements = req.Requirements
	target.SortOrder = req.SortOrder
	target.IsEnabled = req.IsEnabled
}

// missionFields 任务可修改的字段, 用于操作日志
func missionFields(target *mission.Mission) map[string]interface{} {
	return map[string]interface{}{
		"code":         target.Code,
		"name":         target.Name,
		"description":  target.Description,
		"cycle":        target.Cycle,
		"scope":        target.Scope,
		"metric":       target.Metric,
		"target":       target.Target,
		"reward":       target.Reward,
		"requirements": target.Requirements,
		"sort_order":   target.SortOrder,
		"is_enabled":   target.IsEnabled,
	}
}

// missionChanges 本次修改了哪些字段, 用于操作日志; 奖励和条件中有切片, 按内容比较
func missionChanges(old, next *mission.Mission) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := missionFields(next)
	for key, oldValue := range missionFields(old) {
		if newValue := nextFields[key]; !reflect.DeepEqual(oldValue, newValue) {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}
