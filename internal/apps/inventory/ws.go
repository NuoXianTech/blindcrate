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

package inventory

import (
	"cmp"
	"context"
	"maps"
	"slices"

	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/ws"
)

// QuantityChange 一种物品变动后的数量
type QuantityChange struct {
	ItemID        uint64 `json:"item_id"`
	Quantity      int64  `json:"quantity"`
	TransactionID uint64 `json:"transaction_id"` // 同一用户同一物品的流水 ID 越大越新, 客户端只采用 ID 更大的推送
}

// ItemsChangedMessage MessageItemsChanged 的 data
type ItemsChangedMessage struct {
	Items []QuantityChange `json:"items"` // 按物品 ID 排序
}

// PushChanges 把流水变动后的数量推送给对应用户, 需要在事务提交后调用
//
// 只推送数量, 背包中没有的新物品需要客户端重新拉取物品信息。
// 同一用户的流水合并为一条消息, 同一物品只推送最新的一条。
// 推送不保证送达, 多个实例推送时还可能乱序, 失败只记录日志。
func PushChanges(ctx context.Context, txns ...*Transaction) {
	byUser := map[uint64]map[uint64]QuantityChange{}
	for _, txn := range txns {
		items := byUser[txn.UserID]
		if items == nil {
			items = map[uint64]QuantityChange{}
			byUser[txn.UserID] = items
		}
		// 同一物品的变动在背包行锁上排队, 流水 ID 的顺序就是提交顺序
		if current, ok := items[txn.ItemID]; ok && current.TransactionID > txn.ID {
			continue
		}
		items[txn.ItemID] = QuantityChange{
			ItemID:        txn.ItemID,
			Quantity:      txn.QuantityAfter,
			TransactionID: txn.ID,
		}
	}

	for _, userID := range slices.Sorted(maps.Keys(byUser)) {
		changes := slices.SortedFunc(maps.Values(byUser[userID]), func(a, b QuantityChange) int {
			return cmp.Compare(a.ItemID, b.ItemID)
		})
		if err := ws.SendToUser(ctx, userID, MessageItemsChanged, ItemsChangedMessage{Items: changes}); err != nil {
			logger.ErrorF(ctx, "[Inventory] push item changes to %d failed: %v", userID, err)
		}
	}
}
