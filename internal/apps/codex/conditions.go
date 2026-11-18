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

package codex

import (
	"errors"
	"fmt"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/params"
	"gorm.io/gorm"
)

// 本文件是按图鉴进度检查的条件: 收集过某件物品、集齐某个收集组、收集了多少种藏品; 收集的算法与图鉴一致, 见包注释

func init() {
	params.RegisterSource(SourceSet, params.Source{Options: setOptions})
	condition.Register(TypeCollected, "图鉴: 收集过物品", "曾经获得过指定的物品, 之后卖掉或用掉也算, 只检查不消耗; 用于高级地图的门槛、收藏家才能买的商品", checkCollected)
	condition.Register(TypeSetCompleted, "图鉴: 集齐收集组", "集齐指定收集组中全部启用的藏品, 不要求领取过奖励; 收集组停用或删除后不再满足", checkSetCompleted)
	condition.Register(TypeCollectedCount, "图鉴: 收集数量", "图鉴中已收集的藏品不少于指定的种数, 只算启用的藏品", checkCollectedCount)
}

// setOptions 可以选的收集组, 按排序值和 ID; 停用的不能新选
func setOptions(tx *gorm.DB) ([]params.Option, error) {
	var sets []Set
	if err := tx.Select("code", "name", "is_enabled").Order("sort_order, id").Find(&sets).Error; err != nil {
		return nil, err
	}
	options := make([]params.Option, len(sets))
	for i, s := range sets {
		options[i] = params.Option{Value: s.Code, Label: s.Name, Hint: s.Code, Disabled: !s.IsEnabled}
	}
	return options, nil
}

// collectedParams 收集过物品的参数
type collectedParams struct {
	ItemID uint64 `json:"item_id" label:"物品" binding:"required" input:"item"`
}

// Validate 物品必须存在
func (p *collectedParams) Validate(tx *gorm.DB) (string, error) {
	var it item.Item
	if err := it.Exact(tx, p.ItemID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item.ItemNotFound, nil
		}
		return "", err
	}
	return "", nil
}

func checkCollected(tx *gorm.DB, userID uint64, p *collectedParams) (*condition.Progress, error) {
	collected, err := countCollected(tx, userID, []uint64{p.ItemID})
	if err != nil {
		return nil, err
	}
	// 物品被删除时按占位物品展示
	items, err := item.Resolve(tx, []uint64{p.ItemID})
	if err != nil {
		return nil, err
	}
	return &condition.Progress{Met: collected > 0, Current: collected, Target: 1, Text: "收集过" + items[p.ItemID].Name}, nil
}

// setCompletedParams 集齐收集组的参数
type setCompletedParams struct {
	Set string `json:"set" label:"收集组" binding:"required,max=32" input:"codex_set"`
}

// Validate 收集组必须存在
func (p *setCompletedParams) Validate(tx *gorm.DB) (string, error) {
	var count int64
	if err := tx.Model(&Set{}).Where("code = ?", p.Set).Count(&count).Error; err != nil {
		return "", err
	}
	if count == 0 {
		return SetNotFound, nil
	}
	return "", nil
}

func checkSetCompleted(tx *gorm.DB, userID uint64, p *setCompletedParams) (*condition.Progress, error) {
	var sets []Set
	if err := tx.Where("code = ? AND is_enabled", p.Set).Limit(1).Find(&sets).Error; err != nil {
		return nil, err
	}
	// 收集组已停用或删除时玩家看不到它, 按代码展示, 不算满足
	if len(sets) == 0 {
		return &condition.Progress{Text: "集齐收集组 " + p.Set}, nil
	}
	ids, err := enabledMembers(tx, sets[0].ItemIDs)
	if err != nil {
		return nil, err
	}
	collected, err := countCollected(tx, userID, ids)
	if err != nil {
		return nil, err
	}
	total := int64(len(ids))
	return &condition.Progress{
		Met:     total > 0 && collected == total,
		Current: collected,
		Target:  total,
		Text:    fmt.Sprintf("集齐%s(%d/%d)", sets[0].Name, collected, total),
	}, nil
}

// collectedCountParams 收集数量的参数
type collectedCountParams struct {
	Count int64 `json:"count" label:"种数" binding:"min=1,max=100000"`
}

func checkCollectedCount(tx *gorm.DB, userID uint64, p *collectedCountParams) (*condition.Progress, error) {
	var collected int64
	if err := tx.Model(&inventory.UserItem{}).
		Joins("JOIN items ON items.id = inventory_items.item_id AND items.deleted_at IS NULL").
		Where("inventory_items.user_id = ? AND items.category = ? AND items.is_enabled", userID, item.CategoryCollectible).
		Count(&collected).Error; err != nil {
		return nil, err
	}
	return &condition.Progress{
		Met:     collected >= p.Count,
		Current: collected,
		Target:  p.Count,
		Text:    fmt.Sprintf("图鉴收集 %d 种藏品(%d/%d)", p.Count, collected, p.Count),
	}, nil
}

// enabledMembers 物品中启用的藏品, 只有它们计入收集组的进度
func enabledMembers(tx *gorm.DB, itemIDs []uint64) ([]uint64, error) {
	var ids []uint64
	if len(itemIDs) == 0 {
		return ids, nil
	}
	err := tx.Model(&item.Item{}).
		Where("id IN ? AND category = ? AND is_enabled", itemIDs, item.CategoryCollectible).
		Pluck("id", &ids).Error
	return ids, err
}

// countCollected 用户收集过 itemIDs 中的几种: 背包中有这一行就算, 数量扣到 0 也保留
func countCollected(tx *gorm.DB, userID uint64, itemIDs []uint64) (int64, error) {
	var collected int64
	if len(itemIDs) == 0 {
		return 0, nil
	}
	err := tx.Model(&inventory.UserItem{}).Where("user_id = ? AND item_id IN ?", userID, itemIDs).Count(&collected).Error
	return collected, err
}
