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
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/shop"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// ListShopItemsRequest 商品列表查询参数
type ListShopItemsRequest struct {
	Current   int    `json:"current" form:"current" binding:"min=1"`
	Size      int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"` // 按代码或名称搜索, 不区分大小写
	Tab       string `json:"tab" form:"tab" binding:"max=16"`         // 按页签筛选, 为空时不筛选
	IsEnabled *bool  `json:"is_enabled" form:"is_enabled"`
}

// ListShopItemsResponseData 商品列表分页结果
type ListShopItemsResponseData struct {
	Total int64       `json:"total"`
	Items []shop.Item `json:"items"`
	Tabs  []string    `json:"tabs"` // 未删除的商品用到的页签, 按名称排列, 不含空页签
}

// ListShopItems
// @Tags admin
// @Summary 获取商品列表
// @Description 按排序值和 ID 排列, 不含已删除的商品
// @Produce json
// @Param request query ListShopItemsRequest true "request query"
// @Success 200 {object} reply.Response{data=ListShopItemsResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/shop/items [get]
func ListShopItems(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListShopItemsRequest{}
	if !reply.Query(c, req) {
		return
	}
	data, err := QueryShopItems(ctx, req)
	if err != nil {
		reply.Fail(c, err, "[ListShopItems] query failed")
		return
	}
	reply.OK(c, data)
}

// ShopItemRequest 新建或修改商品请求体
//
// 价格、商品、购买条件和限购不在这里校验, 由 shop.Validate 校验, 提示更具体。
type ShopItemRequest struct {
	Code          string                `json:"code" binding:"required,max=32"` // 小写字母、数字和下划线, 创建后不能修改
	Name          string                `json:"name" binding:"required,max=64"`
	Description   string                `json:"description" binding:"max=255"`
	ImageURL      string                `json:"image_url" binding:"omitempty,max=1024,image_url"`
	Tab           string                `json:"tab" binding:"max=16"`
	SortOrder     int32                 `json:"sort_order" binding:"min=-1000000,max=1000000"` // 商城中从小到大排列
	IsEnabled     bool                  `json:"is_enabled"`
	Goods         reward.Bundle         `json:"goods" binding:"-"`          // 一份商品, 数量为正数, 不能为空
	Price         reward.Bundle         `json:"price" binding:"-"`          // 一份的价格, 数量为正数, 为空时免费
	OriginalPrice reward.Bundle         `json:"original_price" binding:"-"` // 原价, 只用于展示折扣
	Requirements  []condition.Condition `json:"requirements" binding:"-"`
	Limits        shop.Limits           `json:"limits" binding:"-"`
	Stock         *int64                `json:"stock"`     // 剩余库存, 为空时不限; 修改时直接覆盖
	StartsAt      *time.Time            `json:"starts_at"` // RFC3339, 包含, 为空时不限
	EndsAt        *time.Time            `json:"ends_at"`   // RFC3339, 不包含, 为空时不限
}

// CreateShopItem
// @Tags admin
// @Summary 新建商品
// @Accept json
// @Produce json
// @Param request body ShopItemRequest true "request body"
// @Success 200 {object} reply.Response{data=shop.Item}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/shop/items [post]
func CreateShopItem(c *gin.Context) {
	ctx := c.Request.Context()
	var req ShopItemRequest
	if !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	created, err := doCreateShopItem(ctx, &req)
	if err != nil {
		reply.Fail(c, err, "[CreateShopItem] create failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, shop.ActionShopItemCreate, shop.TargetShopItem, created.ID, shopItemFields(created))
	logger.InfoF(ctx, "[CreateShopItem] %d created shop item %d %s", operator.ID, created.ID, created.Code)
	reply.OK(c, created)
}

// UpdateShopItem
// @Tags admin
// @Summary 修改商品
// @Description 需提交全部字段, 代码不能修改; 已有订单不受影响
// @Accept json
// @Produce json
// @Param id path int true "shop item id"
// @Param request body ShopItemRequest true "request body"
// @Success 200 {object} reply.Response{data=shop.Item}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/shop/items/{id} [put]
func UpdateShopItem(c *gin.Context) {
	ctx := c.Request.Context()
	var req ShopItemRequest
	itemID, ok := reply.ID(c, "id")
	if !ok || !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	updated, changes, err := doUpdateShopItem(ctx, itemID, &req)
	if err != nil {
		reply.Fail(c, err, "[UpdateShopItem] update failed")
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, shop.ActionShopItemUpdate, shop.TargetShopItem, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateShopItem] %d updated shop item %d %s", operator.ID, updated.ID, updated.Code)
	}
	reply.OK(c, updated)
}

// DeleteShopItem
// @Tags admin
// @Summary 删除商品
// @Description 软删除, 代码不能再被新商品使用; 已有订单仍可退款
// @Produce json
// @Param id path int true "shop item id"
// @Success 200 {object} reply.Response
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/shop/items/{id} [delete]
func DeleteShopItem(c *gin.Context) {
	ctx := c.Request.Context()
	itemID, ok := reply.ID(c, "id")
	if !ok {
		return
	}
	operator := auth.CurrentUser(c)
	deleted, err := doDeleteShopItem(ctx, itemID)
	if err != nil {
		reply.Fail(c, err, "[DeleteShopItem] delete failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, shop.ActionShopItemDelete, shop.TargetShopItem, deleted.ID, map[string]interface{}{
		"code": deleted.Code,
		"name": deleted.Name,
	})
	logger.InfoF(ctx, "[DeleteShopItem] %d deleted shop item %d %s", operator.ID, deleted.ID, deleted.Code)
	reply.OK(c, nil)
}

// ListShopOrdersRequest 订单列表查询参数
type ListShopOrdersRequest struct {
	Current    int        `json:"current" form:"current" binding:"min=1"`
	Size       int        `json:"size" form:"size" binding:"min=1,max=100"`
	UserID     *uint64    `json:"user_id" form:"user_id"`
	ShopItemID *uint64    `json:"shop_item_id" form:"shop_item_id"`
	Status     string     `json:"status" form:"status" binding:"omitempty,oneof=paid refunded"`
	StartTime  *time.Time `json:"start_time" form:"start_time" time_format:"2006-01-02T15:04:05Z07:00"` // RFC3339, 包含
	EndTime    *time.Time `json:"end_time" form:"end_time" time_format:"2006-01-02T15:04:05Z07:00"`     // RFC3339, 不包含
}

// ListShopOrdersResponseData 订单列表分页结果
type ListShopOrdersResponseData struct {
	Total  int64           `json:"total"`
	Orders []ShopOrderItem `json:"orders"`
}

// ListShopOrders
// @Tags admin
// @Summary 获取订单列表
// @Description 从新到旧, 带上用户和退款的管理员的用户名
// @Produce json
// @Param request query ListShopOrdersRequest true "request query"
// @Success 200 {object} reply.Response{data=ListShopOrdersResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/shop/orders [get]
func ListShopOrders(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListShopOrdersRequest{}
	if !reply.Query(c, req) {
		return
	}
	total, orders, err := QueryShopOrders(ctx, req)
	if err != nil {
		reply.Fail(c, err, "[ListShopOrders] query failed")
		return
	}
	reply.OK(c, ListShopOrdersResponseData{Total: total, Orders: orders})
}

// RefundShopOrder
// @Tags admin
// @Summary 订单退款
// @Description 退还价格并收回商品, 恢复库存和购买时所在周期的限购数量; 价格中已下架的物品按回收价折成货币退还, 用户的货币或物品不足以收回商品时返回 409
// @Produce json
// @Param id path int true "order id"
// @Success 200 {object} reply.Response{data=shop.Order}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/shop/orders/{id}/refund [post]
func RefundShopOrder(c *gin.Context) {
	ctx := c.Request.Context()
	orderID, ok := reply.ID(c, "id")
	if !ok {
		return
	}
	operator := auth.CurrentUser(c)
	order, err := shop.Refund(ctx, operator.ID, orderID)
	if err != nil {
		reply.Fail(c, err, "[RefundShopOrder] refund failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, shop.ActionShopOrderRefund, shop.TargetShopOrder, order.ID, map[string]interface{}{
		"user_id":   order.UserID,
		"item_code": order.ItemCode,
		"quantity":  order.Quantity,
		"price":     order.Price,
		"goods":     order.Goods,
	})
	logger.InfoF(ctx, "[RefundShopOrder] %d refunded shop order %d of user %d", operator.ID, order.ID, order.UserID)
	reply.OK(c, order)
}
