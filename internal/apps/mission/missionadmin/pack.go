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
	"github.com/NuoXianTech/blindcrate/internal/apps/mission"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"gorm.io/gorm"
)

// 配置包中的任务
const (
	PackMissions  = "missions"
	OrderMissions = 75 // 排在地图和榜单之后, 范围和条件引用地图代码
)

func init() {
	// 任务按代码对应; 同代码的任务已删除时只能改名导入
	pack.RegisterTable(pack.Table[mission.Mission, MissionRequest]{
		Code: PackMissions, Label: "任务", Hint: "按代码对应", Order: OrderMissions,
		Query: func(tx *gorm.DB) *gorm.DB { return tx.Order("sort_order, id") },
		Pack: func(m *mission.Mission, _ func(string) string) MissionRequest {
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
		},
		Key:    func(req *MissionRequest) string { return req.Code },
		Name:   func(req *MissionRequest) string { return req.Name },
		Taken:  mission.CodeTaken,
		Rename: func(req *MissionRequest, code string) { req.Code = code },
		Renamed: func(req *MissionRequest, section, old, next string) {
			if section == pack.PackVenues {
				req.Scope = pack.RenameScope(req.Scope, old, next)
				pack.RenameConditionScopes(req.Requirements, old, next)
			}
		},
		Find: func(tx *gorm.DB, req *MissionRequest, current *mission.Mission) (bool, error) {
			err := tx.Unscoped().Where("code = ?", req.Code).Limit(1).Find(current).Error
			return current.ID != 0, err
		},
		Deleted: func(m *mission.Mission) bool { return m.DeletedAt.Valid },
		Lock:    func(tx *gorm.DB, current *mission.Mission) error { return LockMission(tx, current, current.ID) },
		Save: func(tx *gorm.DB, _ *pack.Run, current *mission.Mission, req *MissionRequest) error {
			target := current
			if target == nil {
				target = &mission.Mission{}
			} else if err := CheckCycle(current, req); err != nil {
				return err
			}
			ApplyMission(target, req)
			if err := mission.Validate(tx, target); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	})
}
