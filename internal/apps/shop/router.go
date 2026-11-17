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
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// ListItemsResponseData 上架中的商品
type ListItemsResponseData struct {
	Items []ItemView `json:"items"`
}

// ListItems
// @Tags shop
// @Summary 获取上架中的商品
// @Description 按排序值和 ID 排列, 带价格和商品的名称、当前用户对购买条件的达成情况和已购买的份数; 页签由前端按 tab 分组
// @Produce json
// @Success 200 {object} reply.Response{data=ListItemsResponseData}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/shop/items [get]
func ListItems(c *gin.Context) {
	ctx := c.Request.Context()
	user := auth.CurrentUser(c)
	items, err := QueryItems(ctx, user.ID)
	if err != nil {
		reply.Fail(c, err, "[ListShopItems] query failed")
		return
	}
	reply.OK(c, ListItemsResponseData{Items: items})
}

// CreateOrderRequest 购买请求体
type CreateOrderRequest struct {
	Code           string         `json:"code" binding:"required,max=32"`
	Quantity       int64          `json:"quantity" binding:"required,min=1,max=99"`
	Price          *reward.Bundle `json:"price" binding:"-"`                                    // 商城中看到的一份的价格, 传入时与当前不一致则拒绝购买
	IdempotencyKey string         `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次打开购买确认框生成一个, 重复提交只生效一次
}

// CreateOrder
// @Tags shop
// @Summary 购买商品
// @Description 扣除价格并发放商品, 全部成功或全部失败; 重复提交返回原订单。不在上架时间、库存不足、不满足购买条件或达到限购时返回 403
// @Accept json
// @Produce json
// @Param request body CreateOrderRequest true "request body"
// @Success 200 {object} reply.Response{data=OrderView}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 429 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Param X-Captcha-Token header string false "人机验证令牌, 该场景开启人机验证时必填"
// @Router /api/v1/shop/orders [post]
func CreateOrder(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreateOrderRequest
	if !reply.JSON(c, &req) {
		return
	}
	user := auth.CurrentUser(c)
	order, replayed, err := Purchase(ctx, user.ID, &PurchaseRequest{
		Code:           req.Code,
		Quantity:       req.Quantity,
		Price:          req.Price,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		reply.Fail(c, err, "[CreateShopOrder] purchase failed")
		return
	}
	// 重复提交只返回原订单, 不重复记录
	if !replayed {
		logger.InfoF(ctx, "[CreateShopOrder] %d bought %d of %s in order %d", user.ID, order.Quantity, order.ItemCode, order.ID)
	}
	reply.OK(c, order)
}

// ListOrdersRequest 订单查询参数
type ListOrdersRequest struct {
	Current int `json:"current" form:"current" binding:"min=1"`
	Size    int `json:"size" form:"size" binding:"min=1,max=100"`
}

// ListOrdersResponseData 订单分页结果
type ListOrdersResponseData struct {
	Total  int64       `json:"total"`
	Orders []OrderView `json:"orders"`
}

// ListOrders
// @Tags shop
// @Summary 获取当前用户的购买记录
// @Description 从新到旧
// @Produce json
// @Param request query ListOrdersRequest true "request query"
// @Success 200 {object} reply.Response{data=ListOrdersResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/shop/orders [get]
func ListOrders(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListOrdersRequest{}
	if !reply.Query(c, req) {
		return
	}
	user := auth.CurrentUser(c)
	total, orders, err := QueryUserOrders(ctx, user.ID, (req.Current-1)*req.Size, req.Size)
	if err != nil {
		reply.Fail(c, err, "[ListShopOrders] query failed")
		return
	}
	reply.OK(c, ListOrdersResponseData{Total: total, Orders: orders})
}
