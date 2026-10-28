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

package loot

import (
	"context"
	"errors"

	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
)

// EntryDetail 掉落池条目, 附带物品或货币信息和当前的概率
type EntryDetail struct {
	Entry
	Name        string  `json:"name"`         // 物品或货币的名称, 已删除的物品为占位名称
	Quality     int16   `json:"quality"`      // 物品品质, 货币为 0
	ImageURL    string  `json:"image_url"`    // 物品图片, 货币为空
	IsAvailable bool    `json:"is_available"` // 物品已上架或货币已启用, 否则不参与抽取
	Probability float64 `json:"probability"`  // 当前每次抽中的概率, 不参与抽取时为 0
}

// QueryEntries 获取掉落池的全部条目并计算概率, 按条目 ID 排序
//
// 已下架或删除的物品、已停用或不存在的货币不参与抽取, 概率按其余条目的权重计算, 与 Roll 一致。
func QueryEntries(tx *gorm.DB, poolID uint64) ([]EntryDetail, error) {
	var entries []Entry
	if err := tx.Where("pool_id = ?", poolID).Order("id").Find(&entries).Error; err != nil {
		return nil, err
	}

	var itemIDs []uint64
	for _, entry := range entries {
		if entry.ItemID != 0 {
			itemIDs = append(itemIDs, entry.ItemID)
		}
	}
	items, err := item.Resolve(tx, itemIDs)
	if err != nil {
		return nil, err
	}
	var currencyList []wallet.Currency
	if err := tx.Find(&currencyList).Error; err != nil {
		return nil, err
	}
	currencies := make(map[string]*wallet.Currency, len(currencyList))
	for i := range currencyList {
		currencies[currencyList[i].Code] = &currencyList[i]
	}

	details := make([]EntryDetail, len(entries))
	var totalWeight int64
	for i, entry := range entries {
		detail := EntryDetail{Entry: entry}
		if entry.ItemID != 0 {
			target := items[entry.ItemID]
			detail.Name = target.Name
			detail.Quality = target.Quality
			detail.ImageURL = target.ImageURL
			detail.IsAvailable = target.IsEnabled && !target.IsPlaceholder
		} else if currency := currencies[entry.Currency]; currency != nil {
			detail.Name = currency.Name
			detail.IsAvailable = currency.IsEnabled
		} else {
			detail.Name = entry.Currency
		}
		if detail.IsAvailable {
			totalWeight += int64(entry.Weight)
		}
		details[i] = detail
	}
	for i := range details {
		if details[i].IsAvailable {
			details[i].Probability = float64(details[i].Weight) / float64(totalWeight)
		}
	}
	return details, nil
}

// QueryAvailable 按代码获取已启用的掉落池和其中可抽取的条目, 概率与 Roll 一致
//
// 需要多次抽取时(如仓库生成)先读出条目再在内存中抽取, 避免每次抽取都查库。
func QueryAvailable(tx *gorm.DB, code string) (*Pool, []EntryDetail, error) {
	var pool Pool
	if err := pool.ExactCode(tx, code); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New(PoolNotFound)
		}
		return nil, nil, err
	}
	if !pool.IsEnabled {
		return nil, nil, errors.New(PoolDisabled)
	}

	entries, err := QueryEntries(tx, pool.ID)
	if err != nil {
		return nil, nil, err
	}
	available := make([]EntryDetail, 0, len(entries))
	for _, entry := range entries {
		if entry.IsAvailable {
			available = append(available, entry)
		}
	}
	if len(available) == 0 {
		return nil, nil, errors.New(PoolEmpty)
	}
	return &pool, available, nil
}

// Drop 一次抽取的结果
type Drop struct {
	EntryID  uint64 `json:"entry_id"`
	ItemID   uint64 `json:"item_id,omitempty"`  // 掉落的物品, 与 Currency 二选一
	Currency string `json:"currency,omitempty"` // 掉落的货币
	Quantity int64  `json:"quantity"`           // 物品数量或货币金额
}

// RollResult 从掉落池中抽取的结果
type RollResult struct {
	PoolID uint64        `json:"pool_id"`
	Drops  []Drop        `json:"drops"`  // 按抽取顺序
	Bundle reward.Bundle `json:"bundle"` // 合并后的掉落, 交给 reward.Apply 发放
}

// Roll 从已启用的掉落池中按权重独立抽取 times 次, 只抽取不发放
//
// 抽取范围和概率与 QueryEntries 一致; 抽取时不加锁, 发放前物品被下架时 reward.Apply 返回 item.ItemDisabled。
// 抽取结果不会自动保存, 需要可追溯的玩法应把结果和发放写在同一个事务里。
// 同一物品或货币的掉落合计超过单笔变动上限时返回 inventory.QuantityInvalid 或 wallet.AmountInvalid, 配置条目时需考虑 times。
func Roll(tx *gorm.DB, code string, times int) (*RollResult, error) {
	if times < 1 || times > MaxRollTimes {
		return nil, errors.New(RollTimesInvalid)
	}
	pool, entries, err := QueryAvailable(tx, code)
	if err != nil {
		return nil, err
	}

	weights := make([]int64, len(entries))
	for i := range entries {
		weights[i] = int64(entries[i].Weight)
	}
	result := &RollResult{PoolID: pool.ID, Drops: make([]Drop, times)}
	var bundle reward.Bundle
	for i := range times {
		entry := entries[PickIndex(weights)]
		quantity := entry.MinQuantity + RandInt64N(entry.MaxQuantity-entry.MinQuantity+1)
		result.Drops[i] = Drop{EntryID: entry.ID, ItemID: entry.ItemID, Currency: entry.Currency, Quantity: quantity}
		if entry.ItemID != 0 {
			bundle.Items = append(bundle.Items, reward.ItemLine{ItemID: entry.ItemID, Quantity: quantity})
		} else {
			bundle.Currencies = append(bundle.Currencies, reward.CurrencyLine{Currency: entry.Currency, Amount: quantity})
		}
	}
	if result.Bundle, err = bundle.Normalize(); err != nil {
		return nil, err
	}
	return result, nil
}

// PoolOdds 掉落池的公示概率
type PoolOdds struct {
	Code        string        `json:"code"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Entries     []EntryDetail `json:"entries"` // 只含可抽取的条目
}

// QueryOdds 获取已启用的掉落池中可抽取的条目和概率, 用于向玩家公示
func QueryOdds(ctx context.Context, code string) (*PoolOdds, error) {
	pool, entries, err := QueryAvailable(db.DB(ctx), code)
	if err != nil {
		return nil, err
	}
	return &PoolOdds{Code: pool.Code, Name: pool.Name, Description: pool.Description, Entries: entries}, nil
}
