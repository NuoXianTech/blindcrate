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
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
)

func init() {
	settings.RegisterRef(RefCurrency, currencyOptions)
}

// ChangeRequest 一次余额变动
//
// 常见用法:
//   - 入账: Amount = +x
//   - 出账: Amount = -x
//   - 冻结(如竞拍押金): Amount = -x, FrozenAmount = +x
//   - 解冻(如流拍退回): Amount = +x, FrozenAmount = -x
//   - 扣除冻结(如成交): FrozenAmount = -x
type ChangeRequest struct {
	UserID         uint64
	Currency       string
	Amount         int64  // 可用余额变动, 正数入账, 负数出账
	FrozenAmount   int64  // 冻结余额变动, 正数冻结, 负数解冻或扣除
	BizType        string // 业务类型, 见 constants.go, 最长 32
	BizID          string // 业务单号, 便于按业务回查流水, 最长 64
	IdempotencyKey string // 非空时同一个 key 只生效一次, 需带业务前缀保证全局唯一, 最长 128
	OperatorID     uint64 // 操作人, 0 表示系统
	Remark         string // 最长 255 个字符
}

// Change 变动余额并写一条流水, 所有余额变动都必须走这里
//
// tx 必须是调用方开启的事务, 余额变动与业务写入一起提交或回滚;
// 返回错误时调用方应回滚整个事务。
//
// 幂等键命中已有流水时不重复变动, 返回已有流水且 replayed 为 true;
// 同一个幂等键但变动内容不同时返回 IdempotencyConflict。
func Change(tx *gorm.DB, req *ChangeRequest) (*Transaction, bool, error) {
	if committer, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok || committer == nil {
		return nil, false, errors.New("wallet: Change must run inside a transaction")
	}
	if err := validateChange(req); err != nil {
		return nil, false, err
	}

	// 幂等: 同一个 key 的并发请求在咨询锁上排队, 后到的能看到先到的已提交流水
	if req.IdempotencyKey != "" {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "blindcrate:wallet:"+req.IdempotencyKey).Error; err != nil {
			return nil, false, err
		}
		// 查不到是常态, 用 Find 避免每次都打 record not found 日志
		var existing Transaction
		if err := tx.Where("idempotency_key = ?", req.IdempotencyKey).Limit(1).Find(&existing).Error; err != nil {
			return nil, false, err
		}
		if existing.ID != 0 {
			if !existing.sameChange(req) {
				return nil, false, errors.New(IdempotencyConflict)
			}
			return &existing, true, nil
		}
	}

	if err := checkCurrency(tx, req.Currency); err != nil {
		return nil, false, err
	}
	balance, err := applyBalance(tx, req)
	if err != nil {
		return nil, false, err
	}

	txn := &Transaction{
		UserID:         req.UserID,
		Currency:       req.Currency,
		Amount:         req.Amount,
		FrozenAmount:   req.FrozenAmount,
		BalanceAfter:   balance.Balance,
		FrozenAfter:    balance.Frozen,
		BizType:        req.BizType,
		BizID:          req.BizID,
		IdempotencyKey: req.IdempotencyKey,
		OperatorID:     req.OperatorID,
		Remark:         req.Remark,
	}
	if err := tx.Create(txn).Error; err != nil {
		// 咨询锁之外的兜底, 正常不会走到
		if db.IsUniqueViolation(err) {
			return nil, false, errors.New(DuplicateRequest)
		}
		return nil, false, err
	}
	return txn, false, nil
}

// validateChange 校验变动内容, 字段超长属于调用方的编程错误
func validateChange(req *ChangeRequest) error {
	if req.UserID == 0 || req.Currency == "" || req.BizType == "" {
		return errors.New("wallet: user_id, currency and biz_type are required")
	}
	if len(req.BizType) > 32 || len(req.BizID) > 64 || len(req.IdempotencyKey) > 128 || utf8.RuneCountInString(req.Remark) > 255 {
		return errors.New("wallet: biz_type, biz_id, idempotency_key or remark too long")
	}
	if req.Amount == 0 && req.FrozenAmount == 0 {
		return errors.New(AmountInvalid)
	}
	if !amountInRange(req.Amount) || !amountInRange(req.FrozenAmount) {
		return errors.New(AmountInvalid)
	}
	return nil
}

// amountInRange 单笔变动不能超过 MaxAmount
func amountInRange(amount int64) bool {
	return amount >= -MaxAmount && amount <= MaxAmount
}

// currencyOptions 设置中可以选的货币: 全部启用的货币, 按排序值排列
func currencyOptions(tx *gorm.DB) ([]settings.Option, error) {
	var currencies []Currency
	if err := tx.Where("is_enabled").Order("sort_order, code").Find(&currencies).Error; err != nil {
		return nil, err
	}
	options := make([]settings.Option, len(currencies))
	for i, currency := range currencies {
		options[i] = settings.Option{Value: currency.Code, Label: currency.Name}
	}
	return options, nil
}

// checkCurrency 货币必须存在且启用
func checkCurrency(tx *gorm.DB, code string) error {
	var currency Currency
	if err := currency.Exact(tx, code); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(CurrencyNotFound)
		}
		return err
	}
	if !currency.IsEnabled {
		return errors.New(CurrencyDisabled)
	}
	return nil
}

// applyBalance 原子地变动余额并返回变动后的余额行
//
// 不先查后改, 而是把"余额够不够"写进 UPDATE 的条件里,
// 并发请求会在行锁上排队, 余额不足时不命中任何行。
func applyBalance(tx *gorm.DB, req *ChangeRequest) (*Balance, error) {
	var balance Balance
	var result *gorm.DB
	if req.Amount >= 0 && req.FrozenAmount >= 0 {
		// 只增不减: 余额行不存在时直接创建
		result = tx.Raw(
			`INSERT INTO wallet_balances (user_id, currency, balance, frozen) VALUES (?, ?, ?, ?)
			ON CONFLICT (user_id, currency) DO UPDATE SET
				balance = wallet_balances.balance + EXCLUDED.balance,
				frozen = wallet_balances.frozen + EXCLUDED.frozen,
				updated_at = NOW()
			RETURNING *`,
			req.UserID, req.Currency, req.Amount, req.FrozenAmount,
		).Scan(&balance)
	} else {
		result = tx.Raw(
			`UPDATE wallet_balances SET balance = balance + ?, frozen = frozen + ?, updated_at = NOW()
			WHERE user_id = ? AND currency = ? AND balance + ? >= 0 AND frozen + ? >= 0
			RETURNING *`,
			req.Amount, req.FrozenAmount, req.UserID, req.Currency, req.Amount, req.FrozenAmount,
		).Scan(&balance)
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		// 没命中时回查一次, 区分是可用余额还是冻结余额不够; 余额行不存在按 0 处理
		var current Balance
		if err := tx.Where("user_id = ? AND currency = ?", req.UserID, req.Currency).Limit(1).Find(&current).Error; err != nil {
			return nil, err
		}
		if current.Balance+req.Amount < 0 {
			return nil, errors.New(InsufficientBalance)
		}
		return nil, errors.New(InsufficientFrozen)
	}
	return &balance, nil
}

// sameChange 幂等键重放时校验变动内容一致
func (t *Transaction) sameChange(req *ChangeRequest) bool {
	return t.UserID == req.UserID &&
		t.Currency == req.Currency &&
		t.Amount == req.Amount &&
		t.FrozenAmount == req.FrozenAmount &&
		t.BizType == req.BizType &&
		t.BizID == req.BizID &&
		t.Remark == req.Remark
}

// BalanceItem 用户某种货币的余额, 带上货币信息方便展示
type BalanceItem struct {
	Currency  string `json:"currency"`
	Name      string `json:"name"`
	Decimals  int16  `json:"decimals"`
	IsEnabled bool   `json:"is_enabled"`
	Balance   int64  `json:"balance"`
	Frozen    int64  `json:"frozen"`
}

// QueryBalances 查询用户各货币余额, 没有余额行的货币按 0 返回
//
// includeDisabled 为 false 时隐藏已停用且没有余额的货币。
func QueryBalances(ctx context.Context, userID uint64, includeDisabled bool) ([]BalanceItem, error) {
	query := db.DB(ctx).Model(&Currency{}).
		Select(
			"currencies.code AS currency, currencies.name, currencies.decimals, currencies.is_enabled, "+
				"COALESCE(wallet_balances.balance, 0) AS balance, COALESCE(wallet_balances.frozen, 0) AS frozen",
		).
		Joins("LEFT JOIN wallet_balances ON wallet_balances.currency = currencies.code AND wallet_balances.user_id = ?", userID)
	if !includeDisabled {
		query = query.Where("currencies.is_enabled OR wallet_balances.balance > 0 OR wallet_balances.frozen > 0")
	}

	items := []BalanceItem{}
	if err := query.Order("currencies.sort_order, currencies.code").Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// QueryCurrencies 获取全部货币定义
func QueryCurrencies(ctx context.Context) ([]Currency, error) {
	currencies := []Currency{}
	if err := db.DB(ctx).Order("sort_order, code").Find(&currencies).Error; err != nil {
		return nil, err
	}
	return currencies, nil
}

// TransactionItem 用户自己看到的流水, 不含操作人和幂等键
type TransactionItem struct {
	ID           uint64    `json:"id"`
	Currency     string    `json:"currency"`
	Amount       int64     `json:"amount"`
	FrozenAmount int64     `json:"frozen_amount"`
	BalanceAfter int64     `json:"balance_after"`
	FrozenAfter  int64     `json:"frozen_after"`
	BizType      string    `json:"biz_type"`
	BizID        string    `json:"biz_id"`
	Remark       string    `json:"remark"`
	CreatedAt    time.Time `json:"created_at"`
}

// QueryUserTransactions 获取用户自己的流水
func QueryUserTransactions(ctx context.Context, userID uint64, req *ListTransactionsRequest) (int64, []TransactionItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&Transaction{}).Where("user_id = ?", userID)
	if req.Currency != "" {
		query = query.Where("currency = ?", req.Currency)
	}
	if req.BizType != "" {
		query = query.Where("biz_type = ?", req.BizType)
	}
	if req.StartTime != nil {
		query = query.Where("created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("created_at < ?", *req.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	items := []TransactionItem{}
	if err := query.Order("id DESC").Offset(offset).Limit(req.Size).Scan(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}
