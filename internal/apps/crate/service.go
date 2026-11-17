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
	"regexp"

	"github.com/NuoXianTech/blindcrate/internal/apps/effect"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"gorm.io/gorm"
)

func init() {
	item.RegisterCategory(Category, "箱子")
	effect.Register(EffectOpen, effect.SceneInventory, "开启箱子",
		"在背包中开启, 从掉落池中按概率抽取一项货币或物品, 每次消耗 1 个", open)
	reward.RegisterBizType(BizDrop, "箱子掉落")
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

// open 开启箱子: 从掉落池中可抽取的条目里按概率得到一项; 已下架或已删除的物品、已停用的货币不参与抽取
//
// 掉落池不存在、已停用或没有可抽取的条目时箱子暂时无法开启。
func open(ctx *effect.InventoryContext, params *OpenParams) (*effect.Plan, error) {
	pool, entries, err := loot.QueryAvailable(ctx.Tx, params.Pool)
	if err != nil {
		switch err.Error() {
		case loot.PoolNotFound, loot.PoolDisabled, loot.PoolEmpty:
			return nil, effect.Unusable(effect.ItemUnavailable)
		}
		return nil, err
	}
	return &effect.Plan{
		Mode:    effect.ModeRandom,
		Summary: "每次消耗 1 个, 按公示的概率开出其中一项, 直接放进背包或钱包",
		Options: options(entries),
		Detail:  pool.Code,
		BizType: BizDrop,
		Draw:    func() reward.Bundle { return loot.Draw(entries).Bundle() },
	}, nil
}

// options 掉落池的条目换成可以开出的内容, 每个条目一项, 数量为每次掉落的最小数量
func options(entries []loot.EntryDetail) []effect.Option {
	out := make([]effect.Option, len(entries))
	for i, entry := range entries {
		out[i] = effect.Option{MaxAmount: entry.MaxQuantity, Probability: entry.Probability}
		if entry.ItemID != 0 {
			out[i].Bundle.Items = []reward.ItemLine{{ItemID: entry.ItemID, Quantity: entry.MinQuantity}}
		} else {
			out[i].Bundle.Currencies = []reward.CurrencyLine{{Currency: entry.Currency, Amount: entry.MinQuantity}}
		}
	}
	return out
}
