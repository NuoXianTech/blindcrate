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
	"net/http"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 inventory 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListItemsRequest 背包物品查询参数
type ListItemsRequest struct {
	Current int    `json:"current" form:"current" binding:"min=1"`
	Size    int    `json:"size" form:"size" binding:"min=1,max=100"`
	Quality *int16 `json:"quality" form:"quality" binding:"omitempty,min=1,max=7"`
}

// ListItemsResponseData 背包物品分页结果
type ListItemsResponseData struct {
	Total int64            `json:"total"`
	Items []UserItemDetail `json:"items"`
}

// ListItems
// @Tags inventory
// @Summary 获取当前用户背包中的物品
// @Description 只返回数量大于 0 的物品, 已删除的物品按占位物品返回
// @Produce json
// @Param request query ListItemsRequest true "request query"
// @Success 200 {object} Response{data=ListItemsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/inventory/items [get]
func ListItems(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListItemsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	total, items, err := QueryUserItems(ctx, user.ID, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListItems] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListItemsResponseData{Total: total, Items: items}})
}

// ListTransactionsRequest 背包流水查询参数
type ListTransactionsRequest struct {
	Current   int        `json:"current" form:"current" binding:"min=1"`
	Size      int        `json:"size" form:"size" binding:"min=1,max=100"`
	ItemID    *uint64    `json:"item_id" form:"item_id"`
	BizType   string     `json:"biz_type" form:"biz_type" binding:"max=32"`
	StartTime *time.Time `json:"start_time" form:"start_time" time_format:"2006-01-02T15:04:05Z07:00"` // RFC3339, 包含
	EndTime   *time.Time `json:"end_time" form:"end_time" time_format:"2006-01-02T15:04:05Z07:00"`     // RFC3339, 不包含
}

// ListTransactionsResponseData 背包流水分页结果
type ListTransactionsResponseData struct {
	Total        int64             `json:"total"`
	Transactions []TransactionItem `json:"transactions"`
}

// ListTransactions
// @Tags inventory
// @Summary 获取当前用户的背包流水
// @Produce json
// @Param request query ListTransactionsRequest true "request query"
// @Success 200 {object} Response{data=ListTransactionsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/inventory/transactions [get]
func ListTransactions(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListTransactionsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	total, transactions, err := QueryUserTransactions(ctx, user.ID, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListTransactions] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListTransactionsResponseData{Total: total, Transactions: transactions}})
}

// SellItemsRequestItem 出售的一种物品
type SellItemsRequestItem struct {
	ItemID       uint64 `json:"item_id" binding:"required"`
	Quantity     int64  `json:"quantity" binding:"required,min=1,max=1000000"`
	SellCurrency string `json:"sell_currency" binding:"required,max=32"` // 背包中看到的卖出货币, 与当前不一致时拒绝出售
	SellPrice    int64  `json:"sell_price" binding:"required,min=1"`     // 背包中看到的单价, 与当前不一致时拒绝出售
}

// SellItemsRequest 出售请求体
type SellItemsRequest struct {
	Items          []SellItemsRequestItem `json:"items" binding:"required,min=1,max=100,dive"`          // 同一物品只能出现一次
	IdempotencyKey string                 `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次打开出售确认框生成一个, 重复提交只生效一次
}

// SellItems
// @Tags inventory
// @Summary 出售背包中的物品
// @Description 按物品当前的卖出价出售, 可以同时卖出多种物品, 全部成功或全部失败; 重复提交返回原出售单
// @Accept json
// @Produce json
// @Param request body SellItemsRequest true "request body"
// @Success 200 {object} Response{data=SellOrderDetail}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/inventory/sell [post]
func SellItems(c *gin.Context) {
	ctx := c.Request.Context()
	var req SellItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	order, replayed, err := doSellItems(ctx, user.ID, &req)
	if err != nil {
		switch err.Error() {
		case SellItemDuplicated, ItemNotSellable, SellAmountExceeded, QuantityInvalid, InsufficientQuantity, wallet.CurrencyNotFound, wallet.CurrencyDisabled:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case SellPriceChanged, IdempotencyConflict, DuplicateRequest:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[SellItems] sell failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	// 重复提交只返回原出售单, 不重复记录
	if !replayed {
		logger.InfoF(ctx, "[SellItems] %d sold %d kinds of items in order %d", user.ID, len(order.Items), order.ID)
	}
	c.JSON(http.StatusOK, Response{Data: order})
}

// ListSellOrdersRequest 出售记录查询参数
type ListSellOrdersRequest struct {
	Current   int        `json:"current" form:"current" binding:"min=1"`
	Size      int        `json:"size" form:"size" binding:"min=1,max=100"`
	StartTime *time.Time `json:"start_time" form:"start_time" time_format:"2006-01-02T15:04:05Z07:00"` // RFC3339, 包含
	EndTime   *time.Time `json:"end_time" form:"end_time" time_format:"2006-01-02T15:04:05Z07:00"`     // RFC3339, 不包含
}

// ListSellOrdersResponseData 出售记录分页结果
type ListSellOrdersResponseData struct {
	Total  int64             `json:"total"`
	Orders []SellOrderDetail `json:"orders"`
}

// ListSellOrders
// @Tags inventory
// @Summary 获取当前用户的出售记录
// @Produce json
// @Param request query ListSellOrdersRequest true "request query"
// @Success 200 {object} Response{data=ListSellOrdersResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/inventory/sell-orders [get]
func ListSellOrders(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListSellOrdersRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	total, orders, err := QueryUserSellOrders(ctx, user.ID, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListSellOrders] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListSellOrdersResponseData{Total: total, Orders: orders}})
}
