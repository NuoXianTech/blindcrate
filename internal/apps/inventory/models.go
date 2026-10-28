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
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
	ItemID    uint64    `gorm:"primaryKey;autoIncrement:false;index:idx_inventory_items_item_id,where:quantity > 0" json:"item_id"`
	Quantity  int64     `gorm:"not null;default:0;check:chk_inventory_items_quantity,quantity >= 0" json:"quantity"`
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"` // 首次获得时间
	UpdatedAt time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (UserItem) TableName() string { return "inventory_items" }

// Transaction 背包流水, 只追加不修改
//
// 联合索引：
//   - idx_inventory_transactions_user_item (user_id, item_id, id)：按用户和物品倒序分页查询流水
//   - idx_inventory_transactions_biz       (biz_type, biz_id)：按业务单号回查流水
type Transaction struct {
	ID             uint64    `gorm:"primaryKey;index:idx_inventory_transactions_user_item,priority:3" json:"id"`
	UserID         uint64    `gorm:"not null;index:idx_inventory_transactions_user_item,priority:1" json:"user_id"`
	ItemID         uint64    `gorm:"not null;index:idx_inventory_transactions_user_item,priority:2" json:"item_id"`
	ItemName       string    `gorm:"size:64;not null" json:"item_name"` // 变动时的物品名称
	Quantity       int64     `gorm:"not null;check:chk_inventory_transactions_nonzero,quantity <> 0" json:"quantity"`
	QuantityAfter  int64     `gorm:"not null" json:"quantity_after"`
	BizType        string    `gorm:"size:32;not null;index:idx_inventory_transactions_biz,priority:1" json:"biz_type"`
	BizID          string    `gorm:"size:64;not null;default:'';index:idx_inventory_transactions_biz,priority:2" json:"biz_id"`
	IdempotencyKey string    `gorm:"size:128;not null;default:'';index:uniq_inventory_transactions_idempotency_key,unique,where:idempotency_key <> ''" json:"idempotency_key"`
	OperatorID     uint64    `gorm:"not null;default:0;index:idx_inventory_transactions_operator_id,where:operator_id <> 0" json:"operator_id"`
	Remark         string    `gorm:"size:255;not null;default:''" json:"remark"`
	CreatedAt      time.Time `gorm:"not null;default:now();autoCreateTime;index" json:"created_at"`
}

// TableName 自定义表名
func (Transaction) TableName() string { return "inventory_transactions" }

// SellOrder 出售单, 一次出售一行, 可以同时卖出多种物品
//
// 出售产生的背包流水和余额流水, biz_id 都是出售单 ID。
//
// 联合索引：
//   - idx_inventory_sell_orders_user_id          (user_id, id)：按用户倒序分页查询出售单
//   - uniq_inventory_sell_orders_idempotency_key (user_id, idempotency_key)：幂等键在同一用户内唯一
type SellOrder struct {
	ID             uint64    `gorm:"primaryKey;index:idx_inventory_sell_orders_user_id,priority:2" json:"id"`
	UserID         uint64    `gorm:"not null;index:idx_inventory_sell_orders_user_id,priority:1;index:uniq_inventory_sell_orders_idempotency_key,unique,priority:1" json:"user_id"`
	IdempotencyKey string    `gorm:"size:64;not null;index:uniq_inventory_sell_orders_idempotency_key,unique,priority:2" json:"idempotency_key"` // 同一用户内唯一
	FeeRate        int32     `gorm:"not null;default:0" json:"fee_rate"`                                                                         // 出售时的手续费率, 单位为基点
	CreatedAt      time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (SellOrder) TableName() string { return "inventory_sell_orders" }

// SellOrderItem 出售单明细, 每种物品一行, 记录出售时的物品名称和价格
type SellOrderItem struct {
	ID           uint64 `gorm:"primaryKey" json:"id"`
	OrderID      uint64 `gorm:"not null;index" json:"order_id"`
	ItemID       uint64 `gorm:"not null;index" json:"item_id"`
	ItemName     string `gorm:"size:64;not null" json:"item_name"` // 出售时的物品名称
	Quantity     int64  `gorm:"not null;check:chk_inventory_sell_order_items_quantity,quantity > 0" json:"quantity"`
	SellCurrency string `gorm:"size:32;not null" json:"sell_currency"`
	SellPrice    int64  `gorm:"not null;check:chk_inventory_sell_order_items_sell_price,sell_price > 0" json:"sell_price"` // 出售时的单价
	Amount       int64  `gorm:"not null" json:"amount"`                                                                    // 数量 × 单价
}

// TableName 自定义表名
func (SellOrderItem) TableName() string { return "inventory_sell_order_items" }
