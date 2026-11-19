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

package shop

import (
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"gorm.io/gorm"
)

type purchaseCurrency struct {
	Code      string
	Balance   int64
	IsEnabled bool
}

type purchaseResources struct {
	currencies map[string]purchaseCurrency
	quantities map[uint64]int64
	grantable  map[uint64]bool
}

// queryPurchaseResources 批量读取商品用到的资源, 不为每件商品单独查询。
func queryPurchaseResources(tx *gorm.DB, userID uint64, items []Item) (*purchaseResources, error) {
	r := &purchaseResources{currencies: map[string]purchaseCurrency{}, quantities: map[uint64]int64{}, grantable: map[uint64]bool{}}
	var codes []string
	var priceIDs, goodsIDs []uint64
	for _, it := range items {
		for _, line := range it.Price.Currencies {
			codes = append(codes, line.Currency)
		}
		for _, line := range it.Goods.Currencies {
			codes = append(codes, line.Currency)
		}
		for _, line := range it.Price.Items {
			priceIDs = append(priceIDs, line.ItemID)
		}
		for _, line := range it.Goods.Items {
			goodsIDs = append(goodsIDs, line.ItemID)
		}
	}
	if len(codes) > 0 {
		var rows []purchaseCurrency
		if err := tx.Model(&wallet.Currency{}).
			Select("currencies.code, currencies.is_enabled, COALESCE(wallet_balances.balance, 0) AS balance").
			Joins("LEFT JOIN wallet_balances ON wallet_balances.currency = currencies.code AND wallet_balances.user_id = ?", userID).
			Where("currencies.code IN ?", codes).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			r.currencies[row.Code] = row
		}
	}
	if len(priceIDs) > 0 {
		var rows []inventory.UserItem
		if err := tx.Select("item_id, quantity").Where("user_id = ? AND item_id IN ?", userID, priceIDs).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			r.quantities[row.ItemID] = row.Quantity
		}
	}
	if len(goodsIDs) > 0 {
		var ids []uint64
		if err := tx.Model(&item.Item{}).Where("id IN ? AND is_enabled", goodsIDs).Pluck("id", &ids).Error; err != nil {
			return nil, err
		}
		for _, id := range ids {
			r.grantable[id] = true
		}
	}
	return r, nil
}

// maxQuantity 只提供当前可购买份数的预览, Purchase 仍在事务内重新检查并扣款。
// 价格先扣除, 商品后发放; 不能用本次将获得的资源支付本次订单。
func (r *purchaseResources) maxQuantity(it *Item, view *ItemView, hasSeason bool) int64 {
	if !view.Eligible || (it.Limits.Season > 0 && !hasSeason) {
		return 0
	}
	maximum := int64(MaxQuantity)
	if it.Stock != nil {
		maximum = min(maximum, *it.Stock)
	}
	for _, limit := range view.Limits {
		maximum = min(maximum, limit.Limit-limit.Bought)
	}
	for _, line := range it.Price.Currencies {
		currency := r.currencies[line.Currency]
		if !currency.IsEnabled || line.Amount <= 0 {
			return 0
		}
		maximum = min(maximum, currency.Balance/line.Amount, wallet.MaxAmount/line.Amount)
	}
	for _, line := range it.Price.Items {
		if line.Quantity <= 0 {
			return 0
		}
		// 与 inventory.Change 的消耗规则一致, 已下架或删除的支付物品仍可消耗。
		maximum = min(maximum, r.quantities[line.ItemID]/line.Quantity, inventory.MaxQuantity/line.Quantity)
	}
	for _, line := range it.Goods.Currencies {
		if !r.currencies[line.Currency].IsEnabled || line.Amount <= 0 {
			return 0
		}
		maximum = min(maximum, wallet.MaxAmount/line.Amount)
	}
	for _, line := range it.Goods.Items {
		if !r.grantable[line.ItemID] || line.Quantity <= 0 {
			return 0
		}
		maximum = min(maximum, inventory.MaxQuantity/line.Quantity)
	}
	return max(maximum, 0)
}
