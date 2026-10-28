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

package wallet

import (
	"cmp"
	"context"
	"maps"
	"slices"

	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/ws"
)

// BalanceChange 一种货币变动后的余额
type BalanceChange struct {
	Currency      string `json:"currency"`
	Balance       int64  `json:"balance"`
	Frozen        int64  `json:"frozen"`
	TransactionID uint64 `json:"transaction_id"` // 同一用户同一货币的流水 ID 越大越新, 客户端只采用 ID 更大的推送
}

// BalanceChangedMessage MessageBalanceChanged 的 data
type BalanceChangedMessage struct {
	Balances []BalanceChange `json:"balances"` // 按货币代码排序
}

// PushChanges 把流水变动后的余额推送给对应用户, 需要在事务提交后调用
//
// 同一用户的流水合并为一条消息, 同一货币只推送最新的一条。
// 推送不保证送达, 多个实例推送时还可能乱序, 失败只记录日志。
func PushChanges(ctx context.Context, txns ...*Transaction) {
	byUser := map[uint64]map[string]BalanceChange{}
	for _, txn := range txns {
		balances := byUser[txn.UserID]
		if balances == nil {
			balances = map[string]BalanceChange{}
			byUser[txn.UserID] = balances
		}
		// 同一货币的变动在余额行锁上排队, 流水 ID 的顺序就是提交顺序
		if current, ok := balances[txn.Currency]; ok && current.TransactionID > txn.ID {
			continue
		}
		balances[txn.Currency] = BalanceChange{
			Currency:      txn.Currency,
			Balance:       txn.BalanceAfter,
			Frozen:        txn.FrozenAfter,
			TransactionID: txn.ID,
		}
	}

	for _, userID := range slices.Sorted(maps.Keys(byUser)) {
		changes := slices.SortedFunc(maps.Values(byUser[userID]), func(a, b BalanceChange) int {
			return cmp.Compare(a.Currency, b.Currency)
		})
		if err := ws.SendToUser(ctx, userID, MessageBalanceChanged, BalanceChangedMessage{Balances: changes}); err != nil {
			logger.ErrorF(ctx, "[Wallet] push balance changes to %d failed: %v", userID, err)
		}
	}
}
