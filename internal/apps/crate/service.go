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

package crate

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/effect"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/ledger"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
)

func init() {
	item.RegisterCategory(Category, "箱子")
	effect.Register(EffectOpen, effect.SceneInventory, "开启箱子",
		"在背包中开启, 从掉落池中按概率抽取一项货币或物品, 每次消耗 1 个", contentsOf)
	ledger.RegisterBizType(ledger.KindWallet, BizDrop, "箱子掉落")
	ledger.RegisterBizType(ledger.KindInventory, BizDrop, "箱子掉落")
}

// poolPattern 掉落池代码, 与掉落池的校验一致
var poolPattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

// OpenParams 开启箱子的参数
type OpenParams struct {
	Pool string `json:"pool" label:"掉落池" help:"开启一次从这个掉落池中抽取一项; 掉落池不存在、已停用或没有可抽取的条目时箱子暂时无法开启" binding:"required,max=64" input:"pool"`
}

// Validate 只校验掉落池代码的格式, 不要求掉落池已经存在: 配置包中物品先于掉落池导入, 掉落池也可以之后再建
func (p *OpenParams) Validate(_ *gorm.DB) (string, error) {
	if !poolPattern.MatchString(p.Pool) {
		return PoolInvalid, nil
	}
	return "", nil
}

// RenamePool 掉落池改名时把箱子效果参数中的掉落池代码一起改为新代码, 用于配置包的改名导入; 不是箱子或用的不是这个掉落池时不变
func RenamePool(effectCode string, params map[string]any, old, next string) {
	if effectCode == EffectOpen && params[paramPool] == old {
		params[paramPool] = next
	}
}

// UseContext 在背包中使用物品时的上下文, 效果只读不写
type UseContext struct {
	Tx *gorm.DB
}

// Contents 箱子的内容: 掉落池和其中可抽取的条目
type Contents struct {
	Pool    *loot.Pool
	Entries []loot.EntryDetail // 不为空, 概率之和为 1
}

// contentsOf 读取箱子的掉落池中可抽取的条目; 掉落池不存在、已停用或没有可抽取的条目时返回 loot 的提示
func contentsOf(ctx *UseContext, params *OpenParams) (*Contents, error) {
	pool, entries, err := loot.QueryAvailable(ctx.Tx, params.Pool)
	if err != nil {
		return nil, err
	}
	return &Contents{Pool: pool, Entries: entries}, nil
}

// pick 按权重抽取一项, 数量在条目的范围内均匀随机, 与 loot.Roll 抽取一次相同
func (c *Contents) pick() reward.Bundle {
	weights := make([]int64, len(c.Entries))
	for i := range c.Entries {
		weights[i] = int64(c.Entries[i].Weight)
	}
	entry := c.Entries[loot.PickIndex(weights)]
	quantity := entry.MinQuantity + loot.RandInt64N(entry.MaxQuantity-entry.MinQuantity+1)
	if entry.ItemID != 0 {
		return reward.Bundle{Items: []reward.ItemLine{{ItemID: entry.ItemID, Quantity: quantity}}}
	}
	return reward.Bundle{Currencies: []reward.CurrencyLine{{Currency: entry.Currency, Amount: quantity}}}
}

// load 获取箱子和它的内容
//
// 物品不存在、已删除或不能在背包中使用时返回 CrateNotFound; 掉落池不可用时返回 CrateUnavailable。
// 已下架的箱子照常开启, 下架只是不再发放。
func load(tx *gorm.DB, itemID uint64) (*item.Item, *Contents, error) {
	// 物品不存在时不算查询错误, 用 Find 查询
	var target item.Item
	if err := tx.Where("id = ?", itemID).Limit(1).Find(&target).Error; err != nil {
		return nil, nil, err
	}
	if target.ID == 0 || effect.SceneOf(target.Effect) != effect.SceneInventory {
		return nil, nil, errors.New(CrateNotFound)
	}
	contents, err := effect.Apply[*UseContext, *Contents](effect.SceneInventory, target.Effect, &UseContext{Tx: tx}, target.EffectParams)
	if err != nil {
		switch err.Error() {
		case loot.PoolNotFound, loot.PoolDisabled, loot.PoolEmpty:
			return nil, nil, errors.New(CrateUnavailable)
		}
		return nil, nil, err
	}
	return &target, contents, nil
}

// quantityOf 用户背包中某种物品的数量, 没有时为 0
func quantityOf(tx *gorm.DB, userID, itemID uint64) (int64, error) {
	var row inventory.UserItem
	if err := tx.Where("user_id = ? AND item_id = ?", userID, itemID).Limit(1).Find(&row).Error; err != nil {
		return 0, err
	}
	return row.Quantity, nil
}

// Prize 箱子中的一项, 开启一次抽中其中一项
type Prize struct {
	reward.Line         // amount 为每次掉落的最小数量
	MaxAmount   int64   `json:"max_amount"`  // 每次掉落的最大数量, 与 amount 相同时数量固定
	Probability float64 `json:"probability"` // 开启一次抽中这一项的概率
}

// View 玩家看到的箱子
type View struct {
	ItemID      uint64  `json:"item_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Quality     int16   `json:"quality"`
	ImageURL    string  `json:"image_url"`
	Quantity    int64   `json:"quantity"` // 背包中的数量
	Prizes      []Prize `json:"prizes"`   // 按品质从高到低, 货币排在最后; 概率之和为 1
}

// Query 获取箱子、用户持有的数量和箱子中可以开出的全部内容, 用于公示概率和播放开箱动画
//
// 不要求用户持有这个箱子; 已下架或已删除的物品、已停用的货币不参与抽取, 也不在内容中。
func Query(ctx context.Context, userID, itemID uint64) (*View, error) {
	tx := db.DB(ctx)
	target, contents, err := load(tx, itemID)
	if err != nil {
		return nil, err
	}
	quantity, err := quantityOf(tx, userID, itemID)
	if err != nil {
		return nil, err
	}

	bundles := make([]reward.Bundle, len(contents.Entries))
	for i, entry := range contents.Entries {
		if entry.ItemID != 0 {
			bundles[i].Items = []reward.ItemLine{{ItemID: entry.ItemID, Quantity: entry.MinQuantity}}
		} else {
			bundles[i].Currencies = []reward.CurrencyLine{{Currency: entry.Currency, Amount: entry.MinQuantity}}
		}
	}
	lines, err := reward.Lines(tx, bundles...)
	if err != nil {
		return nil, err
	}
	prizes := make([]Prize, len(contents.Entries))
	for i, entry := range contents.Entries {
		prizes[i] = Prize{Line: lines[i][0], MaxAmount: entry.MaxQuantity, Probability: entry.Probability}
	}
	// 稳定排序, 同一品质的保持掉落池中的顺序
	slices.SortStableFunc(prizes, func(a, b Prize) int {
		return cmp.Compare(b.Quality, a.Quality)
	})

	return &View{
		ItemID:      target.ID,
		Name:        target.Name,
		Description: target.Description,
		Quality:     target.Quality,
		ImageURL:    target.ImageURL,
		Quantity:    quantity,
		Prizes:      prizes,
	}, nil
}

// Result 开启一次的结果
type Result struct {
	ID       uint64      `json:"id"`       // 开启记录 ID
	Prize    reward.Line `json:"prize"`    // 抽中并已经发放的货币或物品, amount 为这次得到的数量
	Quantity int64       `json:"quantity"` // 开启后背包中剩余的箱子数
}

// Open 开启一个箱子: 从掉落池中抽取一项, 扣除 1 个箱子并发放抽中的内容; 同一个幂等键重复提交时返回原来的结果, replayed 为 true
//
// 一个事务内完成, 任何一步失败整体回滚, 箱子不会被扣除; 同一用户的开启在咨询锁上排队, 幂等键不会被并发请求绕过。
// 背包中没有这个箱子时返回 inventory.InsufficientQuantity; 掉落池不可用, 或抽中的物品、货币在发放前被停用时返回 CrateUnavailable。
func Open(ctx context.Context, userID, itemID uint64, idempotencyKey string) (*Result, bool, error) {
	var (
		record   Record
		replayed bool
		results  []*reward.Result
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", fmt.Sprintf("blindcrate:crate:%d", userID)).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND idempotency_key = ?", userID, idempotencyKey).Limit(1).Find(&record).Error; err != nil {
			return err
		}
		if record.ID != 0 {
			if record.ItemID != itemID {
				return errors.New(IdempotencyConflict)
			}
			replayed = true
			return nil
		}

		var err error
		results, err = open(tx, userID, itemID, idempotencyKey, &record)
		return err
	})
	if err != nil {
		return nil, false, err
	}
	for _, result := range results {
		result.Push(ctx)
	}

	tx := db.DB(ctx)
	lines, err := reward.Lines(tx, record.Reward)
	if err != nil {
		return nil, false, err
	}
	if len(lines[0]) != 1 {
		return nil, false, fmt.Errorf("crate: record %d has %d reward lines", record.ID, len(lines[0]))
	}
	quantity, err := quantityOf(tx, userID, itemID)
	if err != nil {
		return nil, false, err
	}
	return &Result{ID: record.ID, Prize: lines[0][0], Quantity: quantity}, replayed, nil
}

// open 在已加锁的事务中完成一次开启, 把开启记录写入 record, 返回扣除箱子和发放内容的结果, 用于提交后推送
func open(tx *gorm.DB, userID, itemID uint64, idempotencyKey string, record *Record) ([]*reward.Result, error) {
	target, contents, err := load(tx, itemID)
	if err != nil {
		return nil, err
	}
	drop := contents.pick()

	*record = Record{
		UserID:         userID,
		IdempotencyKey: idempotencyKey,
		ItemID:         target.ID,
		ItemName:       target.Name,
		Pool:           contents.Pool.Code,
		Reward:         drop,
	}
	if err := tx.Create(record).Error; err != nil {
		// 咨询锁之外的兜底, 正常不会走到
		if db.IsUniqueViolation(err) {
			return nil, errors.New(DuplicateRequest)
		}
		return nil, err
	}
	bizID := strconv.FormatUint(record.ID, 10)
	used, _, err := effect.Consume(tx, &effect.UseRequest{
		UserID:         userID,
		ItemID:         target.ID,
		BizID:          bizID,
		IdempotencyKey: "crate:" + bizID + ":use",
		Remark:         useRemark,
		Scopes:         []string{stats.ScopeAll},
	})
	if err != nil {
		return nil, err
	}
	granted, _, err := reward.Apply(tx, &reward.ApplyRequest{
		UserID:         userID,
		Bundle:         drop,
		BizType:        BizDrop,
		BizID:          bizID,
		IdempotencyKey: "crate:" + bizID + ":drop",
		Remark:         target.Name,
	})
	if err != nil {
		switch err.Error() {
		case item.ItemNotFound, item.ItemDisabled, wallet.CurrencyNotFound, wallet.CurrencyDisabled, wallet.AmountInvalid, inventory.QuantityInvalid:
			return nil, errors.New(CrateUnavailable)
		}
		return nil, err
	}
	return []*reward.Result{used, granted}, nil
}
