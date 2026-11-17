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

package lootadmin

import (
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"gorm.io/gorm"
)

// PoolPack 配置包中的一个掉落池, 含条目, 按代码对应
type PoolPack struct {
	LootPoolRequest
	Entries []LootPoolEntryRequest `json:"entries" binding:"dive"`
}

func init() {
	pack.RegisterTable(pack.Table[loot.Pool, PoolPack]{
		Code: pack.PackLootPools, Label: "掉落池", Hint: "按代码对应, 含条目; 已删除物品的条目不导出", Order: pack.OrderLootPools,
		Export: func(tx *gorm.DB, _ func(string) string) ([]PoolPack, error) {
			var (
				rows    []loot.Pool
				entries []loot.Entry
			)
			if err := tx.Order("id").Find(&rows).Error; err != nil {
				return nil, err
			}
			if err := liveEntries(tx).Order("pool_id, id").Find(&entries).Error; err != nil {
				return nil, err
			}
			byPool := map[uint64][]loot.Entry{}
			for _, entry := range entries {
				byPool[entry.PoolID] = append(byPool[entry.PoolID], entry)
			}
			list := make([]PoolPack, len(rows))
			for i := range rows {
				list[i] = poolPack(&rows[i], byPool[rows[i].ID])
			}
			return list, nil
		},
		Current: func(tx *gorm.DB, pool *loot.Pool) (PoolPack, error) {
			var entries []loot.Entry
			if err := liveEntries(tx).Where("pool_id = ?", pool.ID).Order("id").Find(&entries).Error; err != nil {
				return PoolPack{}, err
			}
			return poolPack(pool, entries), nil
		},
		Key:    func(req *PoolPack) string { return req.Code },
		Name:   func(req *PoolPack) string { return req.Name },
		Taken:  loot.PoolCodeTaken,
		Rename: func(req *PoolPack, code string) { req.Code = code },
		Find: func(tx *gorm.DB, req *PoolPack, current *loot.Pool) (bool, error) {
			err := tx.Where("code = ?", req.Code).Limit(1).Find(current).Error
			return current.ID != 0, err
		},
		Lock: func(tx *gorm.DB, current *loot.Pool) error { return LockLootPool(tx, current, current.ID) },
		// 条目整体替换, 与后台保存条目的规则相同
		Save: func(tx *gorm.DB, _ *pack.Run, current *loot.Pool, req *PoolPack) error {
			target := current
			if target == nil {
				target = &loot.Pool{}
			}
			if err := ApplyLootPool(target, &req.LootPoolRequest); err != nil {
				return err
			}
			if err := tx.Save(target).Error; err != nil {
				return err
			}
			entries, err := BuildLootEntries(target.ID, req.Entries)
			if err != nil {
				return err
			}
			if err := CheckLootTargets(tx, entries); err != nil {
				return err
			}
			if err := tx.Where("pool_id = ?", target.ID).Delete(&loot.Entry{}).Error; err != nil {
				return err
			}
			if len(entries) == 0 {
				return nil
			}
			return tx.Create(&entries).Error
		},
	})
}

// poolPack 掉落池和它的条目写入配置包的内容
func poolPack(pool *loot.Pool, entries []loot.Entry) PoolPack {
	result := PoolPack{
		LootPoolRequest: LootPoolRequest{Code: pool.Code, Name: pool.Name, Description: pool.Description, IsEnabled: pool.IsEnabled},
		Entries:         make([]LootPoolEntryRequest, len(entries)),
	}
	for i, entry := range entries {
		result.Entries[i] = LootPoolEntryRequest{
			ItemID:      entry.ItemID,
			Currency:    entry.Currency,
			Weight:      entry.Weight,
			MinQuantity: entry.MinQuantity,
			MaxQuantity: entry.MaxQuantity,
		}
	}
	return result
}

// liveEntries 掉落池中的条目, 不含已删除物品的条目; 这些条目不参与抽取, 也不导出, 否则导入时找不到物品
func liveEntries(tx *gorm.DB) *gorm.DB {
	return tx.Where("item_id = 0 OR item_id IN (?)", tx.Session(&gorm.Session{NewDB: true}).Model(&item.Item{}).Select("id"))
}
