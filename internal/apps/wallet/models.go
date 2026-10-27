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
	"time"

	"gorm.io/gorm"
)

// Currency 货币定义
type Currency struct {
	Code      string    `gorm:"primaryKey;size:32" json:"code"`
	Name      string    `gorm:"size:32;not null" json:"name"`
	Decimals  int16     `gorm:"not null;default:0" json:"decimals"` // 展示用小数位, 金额统一按最小单位的整数存储
	SortOrder int32     `gorm:"not null;default:0" json:"sort_order"`
	IsEnabled bool      `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Currency) TableName() string { return "currencies" }

// Exact 按货币代码查询
func (c *Currency) Exact(tx *gorm.DB, code string) error {
	if err := tx.Where("code = ?", code).First(c).Error; err != nil {
		return err
	}
	return nil
}

// Balance 用户某种货币的余额, 首次入账时创建
type Balance struct {
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
	Currency  string    `gorm:"primaryKey;size:32" json:"currency"`
	Balance   int64     `gorm:"not null;default:0;check:chk_wallet_balances_balance,balance >= 0" json:"balance"`
	Frozen    int64     `gorm:"not null;default:0;check:chk_wallet_balances_frozen,frozen >= 0" json:"frozen"`
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Balance) TableName() string { return "wallet_balances" }

// Transaction 余额流水, 只追加不修改
//
// 联合索引：
//   - idx_wallet_transactions_user_currency (user_id, currency, id)：按用户和货币倒序分页查询流水
//   - idx_wallet_transactions_biz           (biz_type, biz_id)：按业务单号回查流水
type Transaction struct {
	ID             uint64    `gorm:"primaryKey;index:idx_wallet_transactions_user_currency,priority:3" json:"id"`
	UserID         uint64    `gorm:"not null;index:idx_wallet_transactions_user_currency,priority:1" json:"user_id"`
	Currency       string    `gorm:"size:32;not null;index:idx_wallet_transactions_user_currency,priority:2" json:"currency"`
	Amount         int64     `gorm:"not null;default:0;check:chk_wallet_transactions_nonzero,amount <> 0 OR frozen_amount <> 0" json:"amount"`
	FrozenAmount   int64     `gorm:"not null;default:0" json:"frozen_amount"`
	BalanceAfter   int64     `gorm:"not null" json:"balance_after"`
	FrozenAfter    int64     `gorm:"not null" json:"frozen_after"`
	BizType        string    `gorm:"size:32;not null;index:idx_wallet_transactions_biz,priority:1" json:"biz_type"`
	BizID          string    `gorm:"size:64;not null;default:'';index:idx_wallet_transactions_biz,priority:2" json:"biz_id"`
	IdempotencyKey string    `gorm:"size:128;not null;default:'';index:uniq_wallet_transactions_idempotency_key,unique,where:idempotency_key <> ''" json:"idempotency_key"`
	OperatorID     uint64    `gorm:"not null;default:0;index:idx_wallet_transactions_operator_id,where:operator_id <> 0" json:"operator_id"`
	Remark         string    `gorm:"size:255;not null;default:''" json:"remark"`
	CreatedAt      time.Time `gorm:"not null;default:now();autoCreateTime;index" json:"created_at"`
}

// TableName 自定义表名
func (Transaction) TableName() string { return "wallet_transactions" }
