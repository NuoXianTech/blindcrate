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
	"context"
	"errors"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"gorm.io/gorm"
)

// ApplyRequest 一次发放或扣除
type ApplyRequest struct {
	UserID         uint64
	Bundle         Bundle
	BizType        string // 业务类型, 需要同时在 wallet 和 inventory 的 constants.go 中定义, 最长 32
	BizID          string // 业务单号, 最长 64
	IdempotencyKey string // 非空时同一个 key 只生效一次, 需带业务前缀保证全局唯一, 最长 MaxIdempotencyKeyLength
	OperatorID     uint64 // 操作人, 0 表示系统
	Remark         string // 最长 255 个字符
}

// Result 发放或扣除产生的流水
//
// 流水中带有幂等键和操作人, 不要直接返回给用户, 需要时取 Bundle 或流水中的余额和数量。
type Result struct {
	Bundle                Bundle                   // 合并后实际变动的内容
	WalletTransactions    []*wallet.Transaction    // 按货币代码排序
	InventoryTransactions []*inventory.Transaction // 按物品 ID 排序
}

// Apply 在一个事务内变动多种货币和物品, 全部成功或全部失败, 玩法中的花费和奖励都应走这里
//
// tx 必须是调用方开启的事务, 返回错误时调用方应回滚整个事务; 事务提交后 replayed 为 false 时调用 Result.Push 推送给用户。
// Bundle 会先合并同类项, 合并后为空时不做任何变动; 扣除不足时返回 wallet.InsufficientBalance 或 inventory.InsufficientQuantity。
//
// 先变动物品再变动货币, 各自按 ID 和代码排序, 与出售的加锁顺序一致, 同一用户的并发请求不会互相死锁。
//
// 幂等键会加上货币代码或物品 ID 作为每一行的幂等键; 每一行都命中已有流水时 replayed 为 true,
// 部分命中说明同一个 key 被用于不同的内容, 返回 IdempotencyConflict。
// 重放时少了几行无法发现, 同一个 key 需要始终对应同一个 Bundle。
func Apply(tx *gorm.DB, req *ApplyRequest) (*Result, bool, error) {
	if len(req.IdempotencyKey) > MaxIdempotencyKeyLength {
		return nil, false, errors.New("reward: idempotency_key too long")
	}
	bundle, err := req.Bundle.Normalize()
	if err != nil {
		return nil, false, err
	}

	result := &Result{
		Bundle:                bundle,
		WalletTransactions:    make([]*wallet.Transaction, 0, len(bundle.Currencies)),
		InventoryTransactions: make([]*inventory.Transaction, 0, len(bundle.Items)),
	}
	replayedLines := 0
	for _, line := range bundle.Items {
		txn, replayed, err := inventory.Change(tx, &inventory.ChangeRequest{
			UserID:         req.UserID,
			ItemID:         line.ItemID,
			Quantity:       line.Quantity,
			BizType:        req.BizType,
			BizID:          req.BizID,
			IdempotencyKey: lineKey(req.IdempotencyKey, strconv.FormatUint(line.ItemID, 10)),
			OperatorID:     req.OperatorID,
			Remark:         req.Remark,
		})
		if err != nil {
			return nil, false, err
		}
		if replayed {
			replayedLines++
		}
		result.InventoryTransactions = append(result.InventoryTransactions, txn)
	}
	for _, line := range bundle.Currencies {
		txn, replayed, err := wallet.Change(tx, &wallet.ChangeRequest{
			UserID:         req.UserID,
			Currency:       line.Currency,
			Amount:         line.Amount,
			BizType:        req.BizType,
			BizID:          req.BizID,
			IdempotencyKey: lineKey(req.IdempotencyKey, line.Currency),
			OperatorID:     req.OperatorID,
			Remark:         req.Remark,
		})
		if err != nil {
			return nil, false, err
		}
		if replayed {
			replayedLines++
		}
		result.WalletTransactions = append(result.WalletTransactions, txn)
	}

	lines := len(bundle.Items) + len(bundle.Currencies)
	if replayedLines > 0 && replayedLines < lines {
		return nil, false, errors.New(IdempotencyConflict)
	}
	return result, lines > 0 && replayedLines == lines, nil
}

// InvalidError 配置中的奖励或价格不合法, Error 返回的提示可以直接展示给管理员
type InvalidError struct {
	Reason string
}

func (e *InvalidError) Error() string { return e.Reason }

// Check 保存配置前校验一组奖励或价格: 货币和物品各自最多 MaxLines 行, 数量为正数, 货币需存在, 物品需未删除;
// 返回合并同类项后的结果
//
// 不合法时返回 *InvalidError, 其他错误为查库失败等内部错误。
// 已停用的货币和已下架的物品可以保存, 便于提前配置; 发放时已下架的物品用 Deliverable 折成货币。
func Check(tx *gorm.DB, b Bundle) (Bundle, error) {
	if len(b.Currencies) > MaxLines || len(b.Items) > MaxLines {
		return Bundle{}, &InvalidError{Reason: LinesTooMany}
	}
	for _, line := range b.Currencies {
		if line.Amount <= 0 {
			return Bundle{}, &InvalidError{Reason: AmountInvalid}
		}
	}
	for _, line := range b.Items {
		if line.Quantity <= 0 {
			return Bundle{}, &InvalidError{Reason: AmountInvalid}
		}
	}
	normalized, err := b.Normalize()
	if err != nil {
		return Bundle{}, &InvalidError{Reason: err.Error()}
	}
	if len(normalized.Currencies) > 0 {
		codes := make([]string, len(normalized.Currencies))
		for i, line := range normalized.Currencies {
			codes[i] = line.Currency
		}
		var count int64
		if err := tx.Model(&wallet.Currency{}).Where("code IN ?", codes).Count(&count).Error; err != nil {
			return Bundle{}, err
		}
		if count != int64(len(codes)) {
			return Bundle{}, &InvalidError{Reason: CurrencyNotFound}
		}
	}
	if len(normalized.Items) > 0 {
		itemIDs := make([]uint64, len(normalized.Items))
		for i, line := range normalized.Items {
			itemIDs[i] = line.ItemID
		}
		var count int64
		if err := tx.Model(&item.Item{}).Where("id IN ?", itemIDs).Count(&count).Error; err != nil {
			return Bundle{}, err
		}
		if count != int64(len(itemIDs)) {
			return Bundle{}, &InvalidError{Reason: ItemsNotFound}
		}
	}
	return normalized, nil
}

// Deliverable 发放前把已下架或已删除的物品换成货币, 这些物品不能再发放
//
// prices 为物品 ID -> 折算的货币和单价, 没有时按物品当前的回收价; 单价为 0 的物品不折算。
func Deliverable(tx *gorm.DB, bundle Bundle, prices map[uint64]CurrencyLine) (Bundle, error) {
	itemIDs := make([]uint64, len(bundle.Items))
	for i, line := range bundle.Items {
		itemIDs[i] = line.ItemID
	}
	items, err := item.Resolve(tx, itemIDs)
	if err != nil {
		return Bundle{}, err
	}
	result := Bundle{Currencies: append([]CurrencyLine{}, bundle.Currencies...)}
	for _, line := range bundle.Items {
		target := items[line.ItemID]
		if !target.IsPlaceholder && target.IsEnabled {
			result.Items = append(result.Items, line)
			continue
		}
		price, ok := prices[line.ItemID]
		if !ok {
			price = CurrencyLine{Currency: target.SellCurrency, Amount: target.SellPrice}
		}
		if price.Amount > 0 {
			result.Currencies = append(result.Currencies, CurrencyLine{Currency: price.Currency, Amount: price.Amount * line.Quantity})
		}
	}
	return result, nil
}

// Lines 把每个 Bundle 换成带名称的行, 按 bundles 的顺序返回, 每个 Bundle 内货币在前、物品在后
//
// 不存在的货币以代码为名称, 已删除的物品显示为占位物品。
func Lines(tx *gorm.DB, bundles ...Bundle) ([][]Line, error) {
	var (
		codes   []string
		itemIDs []uint64
	)
	for _, b := range bundles {
		for _, line := range b.Currencies {
			codes = append(codes, line.Currency)
		}
		for _, line := range b.Items {
			itemIDs = append(itemIDs, line.ItemID)
		}
	}
	currencies := make(map[string]wallet.Currency, len(codes))
	if len(codes) > 0 {
		var found []wallet.Currency
		if err := tx.Where("code IN ?", codes).Find(&found).Error; err != nil {
			return nil, err
		}
		for _, c := range found {
			currencies[c.Code] = c
		}
	}
	items, err := item.Resolve(tx, itemIDs)
	if err != nil {
		return nil, err
	}

	result := make([][]Line, len(bundles))
	for i, b := range bundles {
		lines := make([]Line, 0, len(b.Currencies)+len(b.Items))
		for _, line := range b.Currencies {
			c, ok := currencies[line.Currency]
			if !ok {
				c = wallet.Currency{Code: line.Currency, Name: line.Currency}
			}
			lines = append(lines, Line{Currency: c.Code, Name: c.Name, Decimals: c.Decimals, Amount: line.Amount})
		}
		for _, line := range b.Items {
			target := items[line.ItemID]
			lines = append(lines, Line{ItemID: line.ItemID, Name: target.Name, ImageURL: target.ImageURL, Quality: target.Quality, Amount: line.Quantity})
		}
		result[i] = lines
	}
	return result, nil
}

// lineKey 每一行的幂等键, 货币和物品的流水在不同的表中, 不会互相冲突
func lineKey(key, suffix string) string {
	if key == "" {
		return ""
	}
	return key + ":" + suffix
}

// Push 把变动后的余额和物品数量推送给用户, 需要在事务提交后调用, 重放时不要调用; 推送失败只记录日志
func (r *Result) Push(ctx context.Context) {
	wallet.PushChanges(ctx, r.WalletTransactions...)
	inventory.PushChanges(ctx, r.InventoryTransactions...)
}
