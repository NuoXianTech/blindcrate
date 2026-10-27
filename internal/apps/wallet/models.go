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
	Code      string    `json:"code" gorm:"primaryKey;size:32"`
	Name      string    `json:"name" gorm:"size:32;not null"`
	Decimals  int16     `json:"decimals" gorm:"not null;default:0"` // 展示用小数位, 金额统一按最小单位的整数存储
	SortOrder int32     `json:"sort_order" gorm:"not null;default:0"`
	IsEnabled bool      `json:"is_enabled" gorm:"not null;default:true"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:now();autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null;default:now();autoUpdateTime"`
}

func (c *Currency) TableName() string {
	return "currencies"
}

func (c *Currency) Exact(tx *gorm.DB, code string) error {
	if err := tx.Where("code = ?", code).First(c).Error; err != nil {
		return err
	}
	return nil
}

// Balance 用户某种货币的余额, 首次入账时创建
type Balance struct {
	UserID    uint64    `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	Currency  string    `json:"currency" gorm:"primaryKey;size:32"`
	Balance   int64     `json:"balance" gorm:"not null;default:0;check:chk_wallet_balances_balance,balance >= 0"`
	Frozen    int64     `json:"frozen" gorm:"not null;default:0;check:chk_wallet_balances_frozen,frozen >= 0"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:now();autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null;default:now();autoUpdateTime"`
}

func (b *Balance) TableName() string {
	return "wallet_balances"
}

// Transaction 余额流水, 只追加不修改
type Transaction struct {
	ID             uint64    `json:"id" gorm:"primaryKey;index:idx_wallet_transactions_user_currency,priority:3"`
	UserID         uint64    `json:"user_id" gorm:"not null;index:idx_wallet_transactions_user_currency,priority:1"`
	Currency       string    `json:"currency" gorm:"size:32;not null;index:idx_wallet_transactions_user_currency,priority:2"`
	Amount         int64     `json:"amount" gorm:"not null;default:0;check:chk_wallet_transactions_nonzero,amount <> 0 OR frozen_amount <> 0"`
	FrozenAmount   int64     `json:"frozen_amount" gorm:"not null;default:0"`
	BalanceAfter   int64     `json:"balance_after" gorm:"not null"`
	FrozenAfter    int64     `json:"frozen_after" gorm:"not null"`
	BizType        string    `json:"biz_type" gorm:"size:32;not null;index:idx_wallet_transactions_biz,priority:1"`
	BizID          string    `json:"biz_id" gorm:"size:64;not null;default:'';index:idx_wallet_transactions_biz,priority:2"`
	IdempotencyKey string    `json:"idempotency_key" gorm:"size:128;not null;default:'';index:uniq_wallet_transactions_idempotency_key,unique,where:idempotency_key <> ''"`
	OperatorID     uint64    `json:"operator_id" gorm:"not null;default:0;index:idx_wallet_transactions_operator_id,where:operator_id <> 0"`
	Remark         string    `json:"remark" gorm:"size:255;not null;default:''"`
	CreatedAt      time.Time `json:"created_at" gorm:"not null;default:now();autoCreateTime;index"`
}

func (t *Transaction) TableName() string {
	return "wallet_transactions"
}
