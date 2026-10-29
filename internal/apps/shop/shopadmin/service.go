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

package shopadmin

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/shop"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// QueryShopItems 获取商品列表, 按排序值和 ID 排列, 附带全部页签
func QueryShopItems(ctx context.Context, req *ListShopItemsRequest) (*ListShopItemsResponseData, error) {
	offset := (req.Current - 1) * req.Size
	tx := db.DB(ctx)

	query := tx.Model(&shop.Item{})
	if req.Keyword != "" {
		keyword := "%" + db.EscapeLike(req.Keyword) + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", keyword, keyword)
	}
	if req.Tab != "" {
		query = query.Where("tab = ?", req.Tab)
	}
	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	data := &ListShopItemsResponseData{Items: []shop.Item{}, Tabs: []string{}}
	if err := query.Count(&data.Total).Error; err != nil {
		return nil, err
	}
	if err := query.Order("sort_order, id").Offset(offset).Limit(req.Size).Find(&data.Items).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&shop.Item{}).Where("tab <> ''").Distinct("tab").Order("tab").Pluck("tab", &data.Tabs).Error; err != nil {
		return nil, err
	}
	return data, nil
}

// doCreateShopItem 校验并新建商品
func doCreateShopItem(ctx context.Context, req *ShopItemRequest) (*shop.Item, error) {
	target := &shop.Item{}
	ApplyShopItem(target, req)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := shop.Validate(tx, target); err != nil {
			return err
		}
		return tx.Create(target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(shop.CodeTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateShopItem 校验并修改商品, 返回修改后的商品和本次修改的字段, 没有任何字段变化时不写库
func doUpdateShopItem(ctx context.Context, itemID uint64, req *ShopItemRequest) (*shop.Item, map[string]interface{}, error) {
	var (
		target  shop.Item
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockShopItem(tx, &target, itemID); err != nil {
			return err
		}
		if req.Code != target.Code {
			return errors.New(shop.CodeImmutable)
		}
		old := target
		ApplyShopItem(&target, req)
		if err := shop.Validate(tx, &target); err != nil {
			return err
		}
		changes = shopItemChanges(&old, &target)
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

// doDeleteShopItem 软删除商品, 返回被删除的商品
func doDeleteShopItem(ctx context.Context, itemID uint64) (*shop.Item, error) {
	var target shop.Item
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockShopItem(tx, &target, itemID); err != nil {
			return err
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// LockShopItem 锁定商品, 防止并发修改、删除和购买互相覆盖
func LockShopItem(tx *gorm.DB, target *shop.Item, itemID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), itemID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(shop.ItemNotFound)
		}
		return err
	}
	return nil
}

// ApplyShopItem 写入商品字段, 由 shop.Validate 校验; 时间精确到分钟
func ApplyShopItem(target *shop.Item, req *ShopItemRequest) {
	target.Code = req.Code
	target.Name = strings.TrimSpace(req.Name)
	target.Description = strings.TrimSpace(req.Description)
	target.ImageURL = req.ImageURL
	target.Tab = strings.TrimSpace(req.Tab)
	target.SortOrder = req.SortOrder
	target.IsEnabled = req.IsEnabled
	target.Goods = req.Goods
	target.Price = req.Price
	target.OriginalPrice = req.OriginalPrice
	target.Requirements = req.Requirements
	target.Limits = req.Limits
	target.Stock = req.Stock
	target.StartsAt = truncateMinute(req.StartsAt)
	target.EndsAt = truncateMinute(req.EndsAt)
}

// truncateMinute 去掉时间中的秒, 为空时返回空
func truncateMinute(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	truncated := t.Truncate(time.Minute)
	return &truncated
}

// shopItemFields 商品可修改的字段, 用于操作日志; 时间统一为 UTC 的 RFC3339 格式, 便于比较
func shopItemFields(target *shop.Item) map[string]interface{} {
	formatTime := func(t *time.Time) interface{} {
		if t == nil {
			return nil
		}
		return t.UTC().Format(time.RFC3339)
	}
	var stock interface{}
	if target.Stock != nil {
		stock = *target.Stock
	}
	return map[string]interface{}{
		"code":           target.Code,
		"name":           target.Name,
		"description":    target.Description,
		"image_url":      target.ImageURL,
		"tab":            target.Tab,
		"sort_order":     target.SortOrder,
		"is_enabled":     target.IsEnabled,
		"goods":          target.Goods,
		"price":          target.Price,
		"original_price": target.OriginalPrice,
		"requirements":   target.Requirements,
		"limits":         target.Limits,
		"stock":          stock,
		"starts_at":      formatTime(target.StartsAt),
		"ends_at":        formatTime(target.EndsAt),
	}
}

// shopItemChanges 本次修改了哪些字段, 用于操作日志; 价格和条件中有切片, 按内容比较
func shopItemChanges(old, next *shop.Item) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := shopItemFields(next)
	for key, oldValue := range shopItemFields(old) {
		if newValue := nextFields[key]; !reflect.DeepEqual(oldValue, newValue) {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}

// ShopOrderItem 管理后台展示的订单, 带上用户和退款的管理员的用户名
type ShopOrderItem struct {
	shop.Order
	Username         string `json:"username"`          // 用户已不存在时为空
	RefundedUsername string `json:"refunded_username"` // 没有退款时为空
}

// QueryShopOrders 获取订单列表, 从新到旧
func QueryShopOrders(ctx context.Context, req *ListShopOrdersRequest) (int64, []ShopOrderItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&shop.Order{})
	if req.UserID != nil {
		query = query.Where("shop_orders.user_id = ?", *req.UserID)
	}
	if req.ShopItemID != nil {
		query = query.Where("shop_orders.shop_item_id = ?", *req.ShopItemID)
	}
	if req.Status != "" {
		query = query.Where("shop_orders.status = ?", req.Status)
	}
	if req.StartTime != nil {
		query = query.Where("shop_orders.created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("shop_orders.created_at < ?", *req.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	orders := []ShopOrderItem{}
	if err := query.
		Select("shop_orders.*, COALESCE(users.username, '') AS username, COALESCE(refunders.username, '') AS refunded_username").
		Joins("LEFT JOIN users ON users.id = shop_orders.user_id").
		Joins("LEFT JOIN users AS refunders ON refunders.id = shop_orders.refunded_by").
		Order("shop_orders.id DESC").Offset(offset).Limit(req.Size).
		Find(&orders).Error; err != nil {
		return 0, nil, err
	}
	return total, orders, nil
}
