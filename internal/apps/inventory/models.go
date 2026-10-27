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
	"time"
)

// UserItem 用户持有的某种物品, 首次获得时创建, 数量扣到 0 也保留
//
// 物品被删除后这一行保留, 按占位物品展示, 见 item.Resolve。
type UserItem struct {
	UserID    uint64    `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	ItemID    uint64    `json:"item_id" gorm:"primaryKey;autoIncrement:false;index:idx_inventory_items_item_id,where:quantity > 0"`
	Quantity  int64     `json:"quantity" gorm:"not null;default:0;check:chk_inventory_items_quantity,quantity >= 0"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:now();autoCreateTime"` // 首次获得时间
	UpdatedAt time.Time `json:"updated_at" gorm:"not null;default:now();autoUpdateTime"`
}

func (u *UserItem) TableName() string {
	return "inventory_items"
}

// Transaction 背包流水, 只追加不修改
type Transaction struct {
	ID             uint64    `json:"id" gorm:"primaryKey;index:idx_inventory_transactions_user_item,priority:3"`
	UserID         uint64    `json:"user_id" gorm:"not null;index:idx_inventory_transactions_user_item,priority:1"`
	ItemID         uint64    `json:"item_id" gorm:"not null;index:idx_inventory_transactions_user_item,priority:2"`
	ItemName       string    `json:"item_name" gorm:"size:64;not null"` // 变动时的物品名称
	Quantity       int64     `json:"quantity" gorm:"not null;check:chk_inventory_transactions_nonzero,quantity <> 0"`
	QuantityAfter  int64     `json:"quantity_after" gorm:"not null"`
	BizType        string    `json:"biz_type" gorm:"size:32;not null;index:idx_inventory_transactions_biz,priority:1"`
	BizID          string    `json:"biz_id" gorm:"size:64;not null;default:'';index:idx_inventory_transactions_biz,priority:2"`
	IdempotencyKey string    `json:"idempotency_key" gorm:"size:128;not null;default:'';index:uniq_inventory_transactions_idempotency_key,unique,where:idempotency_key <> ''"`
	OperatorID     uint64    `json:"operator_id" gorm:"not null;default:0;index:idx_inventory_transactions_operator_id,where:operator_id <> 0"`
	Remark         string    `json:"remark" gorm:"size:255;not null;default:''"`
	CreatedAt      time.Time `json:"created_at" gorm:"not null;default:now();autoCreateTime;index"`
}

func (t *Transaction) TableName() string {
	return "inventory_transactions"
}

// SellOrder 出售单, 一次出售一行, 可以同时卖出多种物品
//
// 出售产生的背包流水和余额流水, biz_id 都是出售单 ID。
type SellOrder struct {
	ID             uint64    `json:"id" gorm:"primaryKey;index:idx_inventory_sell_orders_user_id,priority:2"`
	UserID         uint64    `json:"user_id" gorm:"not null;index:idx_inventory_sell_orders_user_id,priority:1;index:uniq_inventory_sell_orders_idempotency_key,unique,priority:1"`
	IdempotencyKey string    `json:"idempotency_key" gorm:"size:64;not null;index:uniq_inventory_sell_orders_idempotency_key,unique,priority:2"` // 同一用户内唯一
	CreatedAt      time.Time `json:"created_at" gorm:"not null;default:now();autoCreateTime"`
}

func (o *SellOrder) TableName() string {
	return "inventory_sell_orders"
}

// SellOrderItem 出售单明细, 每种物品一行, 记录出售时的物品名称和价格
type SellOrderItem struct {
	ID           uint64 `json:"id" gorm:"primaryKey"`
	OrderID      uint64 `json:"order_id" gorm:"not null;index"`
	ItemID       uint64 `json:"item_id" gorm:"not null;index"`
	ItemName     string `json:"item_name" gorm:"size:64;not null"` // 出售时的物品名称
	Quantity     int64  `json:"quantity" gorm:"not null;check:chk_inventory_sell_order_items_quantity,quantity > 0"`
	SellCurrency string `json:"sell_currency" gorm:"size:32;not null"`
	SellPrice    int64  `json:"sell_price" gorm:"not null;check:chk_inventory_sell_order_items_sell_price,sell_price > 0"` // 出售时的单价
	Amount       int64  `json:"amount" gorm:"not null"`                                                                    // 数量 × 单价
}

func (i *SellOrderItem) TableName() string {
	return "inventory_sell_order_items"
}
