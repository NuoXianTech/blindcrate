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
	"slices"

	"github.com/NuoXianTech/blindcrate/internal/apps/codex"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"gorm.io/gorm"
)

func init() {
	// 条件里按代码引用收集组(如集齐收集组), 收集组改名导入时一起改
	pack.RegisterReference(pack.PackCodexSets, codex.SourceSet)
	// 收集组按代码对应; 同代码的收集组已删除时只能改名导入
	pack.RegisterTable(pack.Table[codex.Set, SetRequest]{
		Code: pack.PackCodexSets, Label: "收集组", Hint: "按代码对应, 物品按 ID 引用; 已删除或不是藏品的物品不导出", Order: pack.OrderCodexSets,
		Export: func(tx *gorm.DB, _ func(string) string) ([]SetRequest, error) {
			var rows []codex.Set
			if err := tx.Order("sort_order, id").Find(&rows).Error; err != nil {
				return nil, err
			}
			var all []uint64
			for i := range rows {
				all = append(all, rows[i].ItemIDs...)
			}
			slices.Sort(all)
			collectible, err := collectibleIDs(tx, slices.Compact(all))
			if err != nil {
				return nil, err
			}
			list := make([]SetRequest, len(rows))
			for i := range rows {
				list[i] = setPack(&rows[i], collectible)
			}
			return list, nil
		},
		// 与导出的内容比较, 本环境导出的包再导入时为未变化
		Current: func(tx *gorm.DB, s *codex.Set) (SetRequest, error) {
			collectible, err := collectibleIDs(tx, s.ItemIDs)
			if err != nil {
				return SetRequest{}, err
			}
			return setPack(s, collectible), nil
		},
		Key:    func(req *SetRequest) string { return req.Code },
		Name:   func(req *SetRequest) string { return req.Name },
		Taken:  codex.CodeTaken,
		Rename: func(req *SetRequest, code string) { req.Code = code },
		Find: func(tx *gorm.DB, req *SetRequest, current *codex.Set) (bool, error) {
			err := tx.Unscoped().Where("code = ?", req.Code).Limit(1).Find(current).Error
			return current.ID != 0, err
		},
		Deleted: func(s *codex.Set) bool { return s.DeletedAt.Valid },
		Lock:    func(tx *gorm.DB, current *codex.Set) error { return LockSet(tx, current, current.ID) },
		Save: func(tx *gorm.DB, _ *pack.Run, current *codex.Set, req *SetRequest) error {
			target := current
			var kept []uint64
			if target == nil {
				target = &codex.Set{}
			} else {
				kept = current.ItemIDs
			}
			ApplySet(target, req)
			if err := codex.Validate(tx, target, kept); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	})
}

// collectibleIDs ids 中未删除的藏品
func collectibleIDs(tx *gorm.DB, ids []uint64) (map[uint64]bool, error) {
	found := map[uint64]bool{}
	if len(ids) == 0 {
		return found, nil
	}
	var rows []uint64
	if err := tx.Model(&item.Item{}).Where("id IN ? AND category = ?", ids, item.CategoryCollectible).Pluck("id", &rows).Error; err != nil {
		return nil, err
	}
	for _, id := range rows {
		found[id] = true
	}
	return found, nil
}

// setPack 收集组写入配置包的内容; 组内已删除或不是藏品的物品导入时不能加入, 不写入包中, 比较时也不算
func setPack(s *codex.Set, collectible map[uint64]bool) SetRequest {
	return SetRequest{
		Code:        s.Code,
		Name:        s.Name,
		Description: s.Description,
		ItemIDs:     slices.DeleteFunc(slices.Clone(s.ItemIDs), func(id uint64) bool { return !collectible[id] }),
		Reward:      s.Reward,
		SortOrder:   s.SortOrder,
		IsEnabled:   s.IsEnabled,
	}
}
