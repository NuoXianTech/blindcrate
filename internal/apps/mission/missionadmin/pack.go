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
	"fmt"

	"github.com/NuoXianTech/blindcrate/internal/apps/mission"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"gorm.io/gorm"
)

// 配置包中的任务
const (
	PackMissions  = "missions"
	OrderMissions = 75 // 排在地图和榜单之后, 范围和条件引用地图代码
	packLabel     = "任务"
)

func init() {
	pack.RegisterSection(pack.Section{
		Code: PackMissions, Label: packLabel, Hint: "按代码对应", Order: OrderMissions, Renamable: true,
		Export: func(_ context.Context, tx *gorm.DB, _ func(string) string) (any, int, error) {
			var rows []mission.Mission
			if err := tx.Order("sort_order, id").Find(&rows).Error; err != nil {
				return nil, 0, err
			}
			list := make([]MissionRequest, len(rows))
			for i := range rows {
				list[i] = missionPack(&rows[i])
			}
			return list, len(list), nil
		},
		Decode: pack.DecodeList(missionEntry, func(req *MissionRequest, section, old, next string) {
			if section == pack.PackVenues {
				req.Scope = pack.RenameScope(req.Scope, old, next)
				pack.RenameConditionScopes(req.Requirements, old, next)
			}
		}),
		Invalid: func(err error) bool {
			var invalid *mission.InvalidError
			return errors.As(err, &invalid)
		},
	})
}

// missionPack 任务写入配置包的内容
func missionPack(m *mission.Mission) MissionRequest {
	return MissionRequest{
		Code:         m.Code,
		Name:         m.Name,
		Description:  m.Description,
		Cycle:        m.Cycle,
		Scope:        m.Scope,
		Metric:       m.Metric,
		Target:       m.Target,
		Reward:       m.Reward,
		Requirements: m.Requirements,
		SortOrder:    m.SortOrder,
		IsEnabled:    m.IsEnabled,
	}
}

// missionEntry 包中的一个任务, 按代码对应已有任务; 同代码的任务已删除时只能改名导入
func missionEntry(_ *pack.Run, req *MissionRequest) *pack.Entry {
	var current mission.Mission
	return &pack.Entry{
		Key:    req.Code,
		Name:   req.Name,
		Taken:  mission.CodeTaken,
		Rename: func(code string) { req.Code = code },
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Unscoped().Where("code = ?", req.Code).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			found := current.ID != 0
			if found && current.DeletedAt.Valid {
				return pack.PackStatusConflict, fmt.Sprintf(pack.PackCodeDeleted, packLabel), nil
			}
			return pack.CompareStatus(found, missionPack(&current), req), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			if err := pack.ValidateFields(req); err != nil {
				return err
			}
			target := &mission.Mission{}
			if exists {
				if err := LockMission(tx, &current, current.ID); err != nil {
					return err
				}
				if err := CheckCycle(&current, req); err != nil {
					return err
				}
				target = &current
			}
			ApplyMission(target, req)
			if err := mission.Validate(tx, target); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	}
}
