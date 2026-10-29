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

package walletadmin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/admin"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WalletTransactionItem 管理后台展示的余额流水, 带上用户和操作人的用户名
type WalletTransactionItem struct {
	ID               uint64    `json:"id"`
	UserID           uint64    `json:"user_id"`
	Username         string    `json:"username"` // 用户已不存在时为空
	Currency         string    `json:"currency"`
	Amount           int64     `json:"amount"`
	FrozenAmount     int64     `json:"frozen_amount"`
	BalanceAfter     int64     `json:"balance_after"`
	FrozenAfter      int64     `json:"frozen_after"`
	BizType          string    `json:"biz_type"`
	BizID            string    `json:"biz_id"`
	IdempotencyKey   string    `json:"idempotency_key"`
	OperatorID       uint64    `json:"operator_id"`
	OperatorUsername string    `json:"operator_username"` // 系统操作或用户已不存在时为空
	Remark           string    `json:"remark"`
	CreatedAt        time.Time `json:"created_at"`
}

// QueryWalletTransactions 获取全部用户的余额流水
func QueryWalletTransactions(ctx context.Context, req *ListWalletTransactionsRequest) (int64, []WalletTransactionItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&wallet.Transaction{})
	if req.UserID != nil {
		query = query.Where("wallet_transactions.user_id = ?", *req.UserID)
	}
	if req.OperatorID != nil {
		query = query.Where("wallet_transactions.operator_id = ?", *req.OperatorID)
	}
	if req.Currency != "" {
		query = query.Where("wallet_transactions.currency = ?", req.Currency)
	}
	if req.BizType != "" {
		query = query.Where("wallet_transactions.biz_type = ?", req.BizType)
	}
	if req.BizID != "" {
		query = query.Where("wallet_transactions.biz_id = ?", req.BizID)
	}
	if req.StartTime != nil {
		query = query.Where("wallet_transactions.created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("wallet_transactions.created_at < ?", *req.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	items := []WalletTransactionItem{}
	if err := query.
		Select("wallet_transactions.*, COALESCE(users.username, '') AS username, COALESCE(operators.username, '') AS operator_username").
		Joins("LEFT JOIN users ON users.id = wallet_transactions.user_id").
		Joins("LEFT JOIN users AS operators ON operators.id = wallet_transactions.operator_id").
		Order("wallet_transactions.id DESC").Offset(offset).Limit(req.Size).
		Scan(&items).Error; err != nil {
		return 0, nil, err
	}

	return total, items, nil
}

// queryUserBalances 获取指定用户的各货币余额(含已停用货币)
func queryUserBalances(ctx context.Context, userID uint64) ([]wallet.BalanceItem, error) {
	var user auth.User
	if err := user.Exact(db.DB(ctx), userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(auth.UserNotFound)
		}
		return nil, err
	}
	return wallet.QueryBalances(ctx, userID, true)
}

// doAdjustUserBalance 管理员给其他用户调账, 不允许给自己调账
//
// 幂等键由前端在每次打开调账表单时生成, 防止重复点击导致重复入账。
func doAdjustUserBalance(ctx context.Context, operatorID, userID uint64, req *AdjustUserBalanceRequest) (*wallet.Transaction, bool, error) {
	if operatorID == userID {
		return nil, false, errors.New(admin.CannotOperateSelf)
	}
	remark, err := admin.NormalizeRemark(req.Remark)
	if err != nil {
		return nil, false, err
	}

	var (
		txn      *wallet.Transaction
		replayed bool
	)
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var user auth.User
		if err := user.Exact(tx, userID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(auth.UserNotFound)
			}
			return err
		}

		var err error
		txn, replayed, err = wallet.Change(tx, &wallet.ChangeRequest{
			UserID:         userID,
			Currency:       req.Currency,
			Amount:         req.Amount,
			BizType:        wallet.BizAdminAdjust,
			IdempotencyKey: fmt.Sprintf("%s:%d:%s", wallet.BizAdminAdjust, operatorID, req.IdempotencyKey),
			OperatorID:     operatorID,
			Remark:         remark,
		})
		return err
	})
	if err != nil {
		return nil, false, err
	}
	return txn, replayed, nil
}

// doCreateCurrency 新建货币
func doCreateCurrency(ctx context.Context, req *CreateCurrencyRequest) (*wallet.Currency, error) {
	target := &wallet.Currency{Code: req.Code}
	ApplyCurrency(target, &req.CurrencyRequest)
	if err := wallet.ValidateCurrency(target); err != nil {
		return nil, err
	}
	if err := db.DB(ctx).Create(target).Error; err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(wallet.CurrencyTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateCurrency 修改货币, 返回修改后的货币和本次修改的字段, 没有任何字段变化时不写库
func doUpdateCurrency(ctx context.Context, code string, req *CurrencyRequest) (*wallet.Currency, map[string]interface{}, error) {
	var (
		target  wallet.Currency
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), code); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(wallet.CurrencyNotFound)
			}
			return err
		}
		old := target
		ApplyCurrency(&target, req)
		if err := CheckCurrencyChange(tx, &old, &target); err != nil {
			return err
		}
		changes = currencyChanges(&old, &target)
		if len(changes) == 0 {
			return nil
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return &target, changes, nil
}

// CheckCurrencyChange 校验修改后的货币
//
// 金额按最小单位存储, 小数位只影响展示; 货币已被使用时修改小数位会让现有金额整体变大或变小, 因此只能修改未使用的货币。
func CheckCurrencyChange(tx *gorm.DB, old, next *wallet.Currency) error {
	if err := wallet.ValidateCurrency(next); err != nil {
		return err
	}
	if next.Decimals != old.Decimals {
		used, err := currencyUsed(tx, old.Code)
		if err != nil {
			return err
		}
		if used {
			return errors.New(wallet.CurrencyInUse)
		}
	}
	if old.IsEnabled && !next.IsEnabled {
		if err := wallet.CheckDisable(tx, old.Code); err != nil {
			return err
		}
	}
	return nil
}

// currencyUsed 货币是否已被余额、物品、地图或掉落池使用, 已删除的物品和地图也算
//
// 对局、出售单和流水都会先写余额, 有余额行就说明已经产生过金额。
func currencyUsed(tx *gorm.DB, code string) (bool, error) {
	var used bool
	err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM wallet_balances WHERE currency = @code)
		OR EXISTS (SELECT 1 FROM items WHERE sell_currency = @code)
		OR EXISTS (SELECT 1 FROM venues WHERE bid_currency = @code)
		OR EXISTS (SELECT 1 FROM loot_pool_entries WHERE currency = @code)`, map[string]interface{}{"code": code}).
		Scan(&used).Error
	return used, err
}

// ApplyCurrency 写入货币字段, 名称去掉首尾空白
func ApplyCurrency(target *wallet.Currency, req *CurrencyRequest) {
	target.Name = strings.TrimSpace(req.Name)
	target.Decimals = req.Decimals
	target.SortOrder = req.SortOrder
	target.IsEnabled = req.IsEnabled
}

// currencyFields 货币的字段, 用于操作日志
func currencyFields(target *wallet.Currency) map[string]interface{} {
	return map[string]interface{}{
		"code":       target.Code,
		"name":       target.Name,
		"decimals":   target.Decimals,
		"sort_order": target.SortOrder,
		"is_enabled": target.IsEnabled,
	}
}

// currencyChanges 本次修改了哪些字段, 用于操作日志
func currencyChanges(old, next *wallet.Currency) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := currencyFields(next)
	for key, oldValue := range currencyFields(old) {
		if newValue := nextFields[key]; oldValue != newValue {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}
