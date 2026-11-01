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
	"fmt"
	"slices"

	"github.com/NuoXianTech/blindcrate/internal/apps/codex"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"gorm.io/gorm"
)

// 配置包中的收集组
const (
	PackCodexSets  = "codex_sets"
	OrderCodexSets = 85 // 排在物品之后, 收集组按 ID 引用物品
	packLabel      = "收集组"
)

func init() {
	pack.RegisterSection(pack.Section{
		Code: PackCodexSets, Label: packLabel, Hint: "按代码对应, 物品按 ID 引用; 已删除或不是藏品的物品不导出", Order: OrderCodexSets, Renamable: true,
		Export: func(_ context.Context, tx *gorm.DB, _ func(string) string) (any, int, error) {
			var rows []codex.Set
			if err := tx.Order("sort_order, id").Find(&rows).Error; err != nil {
				return nil, 0, err
			}
			var all []uint64
			for i := range rows {
				all = append(all, rows[i].ItemIDs...)
			}
			slices.Sort(all)
			collectible, err := collectibleIDs(tx, slices.Compact(all))
			if err != nil {
				return nil, 0, err
			}
			list := make([]SetRequest, len(rows))
			for i := range rows {
				list[i] = setPack(&rows[i], collectible)
			}
			return list, len(list), nil
		},
		Decode: pack.DecodeList(setEntry, nil),
		Invalid: func(err error) bool {
			var invalid *codex.InvalidError
			return errors.As(err, &invalid)
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

// setEntry 包中的一个收集组, 按代码对应已有收集组; 同代码的收集组已删除时只能改名导入
func setEntry(_ *pack.Run, req *SetRequest) *pack.Entry {
	var current codex.Set
	return &pack.Entry{
		Key:    req.Code,
		Name:   req.Name,
		Taken:  codex.CodeTaken,
		Rename: func(code string) { req.Code = code },
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Unscoped().Where("code = ?", req.Code).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			found := current.ID != 0
			if found && current.DeletedAt.Valid {
				return pack.PackStatusConflict, fmt.Sprintf(pack.PackCodeDeleted, packLabel), nil
			}
			// 与导出的内容比较, 本环境导出的包再导入时为未变化
			collectible, err := collectibleIDs(tx, current.ItemIDs)
			if err != nil {
				return "", "", err
			}
			return pack.CompareStatus(found, setPack(&current, collectible), req), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			if err := pack.ValidateFields(req); err != nil {
				return err
			}
			target := &codex.Set{}
			var kept []uint64
			if exists {
				if err := LockSet(tx, &current, current.ID); err != nil {
					return err
				}
				target = &current
				kept = current.ItemIDs
			}
			ApplySet(target, req)
			if err := codex.Validate(tx, target, kept); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	}
}
