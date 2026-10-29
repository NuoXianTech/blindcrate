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
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"gorm.io/gorm"
)

// Item 商品, 一份商品为一组货币和物品, 按一份的价格购买
//
// 删除为软删除; 代码在已删除的商品中也唯一, 避免新商品沿用旧商品的限购计数。
type Item struct {
	ID            uint64                `gorm:"primaryKey" json:"id"`
	Code          string                `gorm:"size:32;not null;uniqueIndex:uniq_shop_items_code" json:"code"` // 自然键, 拼进限购计数的指标, 创建后不能修改
	Name          string                `gorm:"size:64;not null" json:"name"`
	Description   string                `gorm:"size:255;not null;default:''" json:"description"`
	ImageURL      string                `gorm:"size:1024;not null;default:''" json:"image_url"`
	Tab           string                `gorm:"size:16;not null;default:''" json:"tab"`                                 // 页签, 如 道具、礼包; 为空时归入默认页签
	SortOrder     int32                 `gorm:"not null;default:0" json:"sort_order"`                                   // 商城中从小到大排列
	Goods         reward.Bundle         `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"goods"`          // 一份商品, 数量为正数, 不能为空
	Price         reward.Bundle         `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"price"`          // 一份的价格, 数量为正数, 为空时免费
	OriginalPrice reward.Bundle         `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"original_price"` // 原价, 只用于展示折扣, 为空时不展示
	Requirements  []condition.Condition `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"requirements"`   // 购买条件, 需要全部满足
	Limits        Limits                `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"limits"`         // 每人限购的份数
	Stock         *int64                `gorm:"check:chk_shop_items_stock,stock >= 0" json:"stock"`                     // 全服剩余库存, 为空时不限
	Sold          int64                 `gorm:"not null;default:0" json:"sold"`                                         // 已售出的份数, 退款时扣回
	StartsAt      *time.Time            `json:"starts_at"`                                                              // 上架时间, 包含, 为空时不限
	EndsAt        *time.Time            `json:"ends_at"`                                                                // 下架时间, 不包含, 为空时不限
	IsEnabled     bool                  `gorm:"not null;default:false" json:"is_enabled"`
	CreatedAt     time.Time             `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time             `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt        `gorm:"index" json:"-"`
}

// Limits 每人限购的份数, 0 表示不限
type Limits struct {
	Total  int64 `json:"total"`  // 累计
	Daily  int64 `json:"daily"`  // 每天, 日期按 schedule.time_zone 计算
	Season int64 `json:"season"` // 每赛季, 设置后只能在赛季中购买
}

// TableName 自定义表名
func (Item) TableName() string { return "shop_items" }

// Exact 按 ID 获取未删除的商品
func (i *Item) Exact(tx *gorm.DB, id uint64) error {
	if err := tx.Where("id = ?", id).First(i).Error; err != nil {
		return err
	}
	return nil
}

// OnShelf 商品在 now 时是否可以购买: 已启用且在上架时间内, 不检查库存
func (i *Item) OnShelf(now time.Time) bool {
	return i.IsEnabled && (i.StartsAt == nil || !now.Before(*i.StartsAt)) && (i.EndsAt == nil || now.Before(*i.EndsAt))
}

// Order 订单, 记录下单时的商品名称、价格和商品, 之后修改商品不影响已有订单
type Order struct {
	ID             uint64        `gorm:"primaryKey;index:idx_shop_orders_user_id,priority:2" json:"id"`
	UserID         uint64        `gorm:"not null;index:idx_shop_orders_user_id,priority:1;index:uniq_shop_orders_idempotency_key,unique,priority:1" json:"user_id"`
	IdempotencyKey string        `gorm:"size:64;not null;index:uniq_shop_orders_idempotency_key,unique,priority:2" json:"-"` // 同一用户内唯一
	ShopItemID     uint64        `gorm:"not null;index" json:"shop_item_id"`
	ItemCode       string        `gorm:"size:32;not null" json:"item_code"`
	Name           string        `gorm:"size:64;not null" json:"name"`
	Quantity       int64         `gorm:"not null;check:chk_shop_orders_quantity,quantity > 0" json:"quantity"` // 购买的份数
	Price          reward.Bundle `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"price"`        // 实际支付的总价, 数量为正数
	Goods          reward.Bundle `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"goods"`        // 实际得到的全部商品, 数量为正数
	Status         string        `gorm:"size:16;not null;default:'paid'" json:"status"`                        // paid 或 refunded
	RefundedAt     *time.Time    `json:"refunded_at"`
	RefundedBy     uint64        `gorm:"not null;default:0" json:"refunded_by"` // 退款的管理员, 没有退款时为 0
	CreatedAt      time.Time     `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (Order) TableName() string { return "shop_orders" }
