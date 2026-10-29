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

package reward

import (
	"errors"
	"maps"
	"slices"

	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
)

// CurrencyLine 一种货币的变动
type CurrencyLine struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"` // 正数入账, 负数出账
}

// ItemLine 一种物品的变动
type ItemLine struct {
	ItemID   uint64 `json:"item_id"`
	Quantity int64  `json:"quantity"` // 正数获得, 负数消耗
}

// Bundle 一组货币和物品的变动, 如开箱的花费和掉落、任务奖励、邮件附件
//
// 可以直接存进 JSONB 配置; 同一货币或物品可以出现多次, 发放前由 Normalize 合并。
type Bundle struct {
	Currencies []CurrencyLine `json:"currencies,omitempty"`
	Items      []ItemLine     `json:"items,omitempty"`
}

// Line 一种货币或物品的数量, 带名称便于展示, 见 Lines
type Line struct {
	Currency string `json:"currency,omitempty"` // 与 item_id 二选一
	ItemID   uint64 `json:"item_id,omitempty"`
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
	Quality  int16  `json:"quality"`  // 物品的品质等级, 货币为 0
	Decimals int16  `json:"decimals"` // 货币的展示用小数位, 物品为 0
	Amount   int64  `json:"amount"`   // 货币金额或物品数量
}

// Add 把 other 追加到 b 之后, 不合并同类项
func (b *Bundle) Add(other Bundle) {
	b.Currencies = append(b.Currencies, other.Currencies...)
	b.Items = append(b.Items, other.Items...)
}

// IsEmpty 是否没有任何变动
func (b Bundle) IsEmpty() bool {
	return len(b.Currencies) == 0 && len(b.Items) == 0
}

// Normalize 合并同一货币和同一物品的变动, 去掉合计为 0 的行, 货币按代码排序, 物品按 ID 排序
//
// 合计超出单笔变动上限时返回 wallet.AmountInvalid 或 inventory.QuantityInvalid。
func (b Bundle) Normalize() (Bundle, error) {
	amounts := map[string]int64{}
	for _, line := range b.Currencies {
		if line.Currency == "" || !inRange(line.Amount, wallet.MaxAmount) || !inRange(amounts[line.Currency]+line.Amount, wallet.MaxAmount) {
			return Bundle{}, errors.New(wallet.AmountInvalid)
		}
		amounts[line.Currency] += line.Amount
	}
	quantities := map[uint64]int64{}
	for _, line := range b.Items {
		if line.ItemID == 0 || !inRange(line.Quantity, inventory.MaxQuantity) || !inRange(quantities[line.ItemID]+line.Quantity, inventory.MaxQuantity) {
			return Bundle{}, errors.New(inventory.QuantityInvalid)
		}
		quantities[line.ItemID] += line.Quantity
	}

	var normalized Bundle
	for _, currency := range slices.Sorted(maps.Keys(amounts)) {
		if amounts[currency] != 0 {
			normalized.Currencies = append(normalized.Currencies, CurrencyLine{Currency: currency, Amount: amounts[currency]})
		}
	}
	for _, itemID := range slices.Sorted(maps.Keys(quantities)) {
		if quantities[itemID] != 0 {
			normalized.Items = append(normalized.Items, ItemLine{ItemID: itemID, Quantity: quantities[itemID]})
		}
	}
	return normalized, nil
}

// Negate 取反, 把奖励变成花费或反过来
func (b Bundle) Negate() Bundle {
	negated := Bundle{
		Currencies: make([]CurrencyLine, len(b.Currencies)),
		Items:      make([]ItemLine, len(b.Items)),
	}
	for i, line := range b.Currencies {
		negated.Currencies[i] = CurrencyLine{Currency: line.Currency, Amount: -line.Amount}
	}
	for i, line := range b.Items {
		negated.Items[i] = ItemLine{ItemID: line.ItemID, Quantity: -line.Quantity}
	}
	return negated
}

// inRange 绝对值不超过 limit, 两个都不超过 limit 的数相加不会溢出
func inRange(value, limit int64) bool {
	return value >= -limit && value <= limit
}
