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

type ListItemsRequest struct {
	Current int    `json:"current" form:"current" binding:"min=1"`
	Size    int    `json:"size" form:"size" binding:"min=1,max=100"`
	Quality *int16 `json:"quality" form:"quality" binding:"omitempty,min=1,max=7"`
}

type ListItemsResponseData struct {
	Total int64            `json:"total"`
	Items []UserItemDetail `json:"items"`
}

type ListItemsResponse struct {
	ErrorMsg string                 `json:"error_msg"`
	Data     *ListItemsResponseData `json:"data"`
}

// ListItems 获取当前用户背包中的物品
// @Tags inventory
// @Param request query ListItemsRequest true "request query"
// @Produce json
// @Success 200 {object} ListItemsResponse
// @Failure 400 {object} ListItemsResponse
// @Failure 401 {object} ListItemsResponse
// @Failure 403 {object} ListItemsResponse
// @Failure 500 {object} ListItemsResponse
// @Router /api/v1/inventory/items [get]
func ListItems(c *gin.Context) {
	// init req
	req := &ListItemsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, ListItemsResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ListItemsResponse{ErrorMsg: auth.UnAuthorized})
		return
	}
	// query
	total, items, err := QueryUserItems(c.Request.Context(), user.ID, req)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[ListItems] query failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, ListItemsResponse{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, ListItemsResponse{Data: &ListItemsResponseData{Total: total, Items: items}})
}

type ListTransactionsRequest struct {
	Current   int        `json:"current" form:"current" binding:"min=1"`
	Size      int        `json:"size" form:"size" binding:"min=1,max=100"`
	ItemID    *uint64    `json:"item_id" form:"item_id"`
	BizType   string     `json:"biz_type" form:"biz_type" binding:"max=32"`
	StartTime *time.Time `json:"start_time" form:"start_time" time_format:"2006-01-02T15:04:05Z07:00"` // RFC3339, 包含
	EndTime   *time.Time `json:"end_time" form:"end_time" time_format:"2006-01-02T15:04:05Z07:00"`     // RFC3339, 不包含
}

type ListTransactionsResponseData struct {
	Total        int64             `json:"total"`
	Transactions []TransactionItem `json:"transactions"`
}

type ListTransactionsResponse struct {
	ErrorMsg string                        `json:"error_msg"`
	Data     *ListTransactionsResponseData `json:"data"`
}

// ListTransactions 获取当前用户的背包流水
// @Tags inventory
// @Param request query ListTransactionsRequest true "request query"
// @Produce json
// @Success 200 {object} ListTransactionsResponse
// @Failure 400 {object} ListTransactionsResponse
// @Failure 401 {object} ListTransactionsResponse
// @Failure 403 {object} ListTransactionsResponse
// @Failure 500 {object} ListTransactionsResponse
// @Router /api/v1/inventory/transactions [get]
func ListTransactions(c *gin.Context) {
	// init req
	req := &ListTransactionsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, ListTransactionsResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ListTransactionsResponse{ErrorMsg: auth.UnAuthorized})
		return
	}
	// query
	total, transactions, err := QueryUserTransactions(c.Request.Context(), user.ID, req)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[ListTransactions] query failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, ListTransactionsResponse{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, ListTransactionsResponse{Data: &ListTransactionsResponseData{Total: total, Transactions: transactions}})
}

type SellItemsRequestItem struct {
	ItemID       uint64 `json:"item_id" binding:"required"`
	Quantity     int64  `json:"quantity" binding:"required,min=1,max=1000000"`
	SellCurrency string `json:"sell_currency" binding:"required,max=32"` // 背包中看到的卖出货币, 与当前不一致时拒绝出售
	SellPrice    int64  `json:"sell_price" binding:"required,min=1"`     // 背包中看到的单价, 与当前不一致时拒绝出售
}

type SellItemsRequest struct {
	Items          []SellItemsRequestItem `json:"items" binding:"required,min=1,max=100,dive"`          // 同一物品只能出现一次
	IdempotencyKey string                 `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次打开出售确认框生成一个, 重复提交只生效一次
}

type SellItemsResponse struct {
	ErrorMsg string           `json:"error_msg"`
	Data     *SellOrderDetail `json:"data"`
}

// SellItems 按物品当前的卖出价出售背包中的物品, 可以同时卖出多种物品, 全部成功或全部失败
// @Tags inventory
// @Param request body SellItemsRequest true "request body"
// @Produce json
// @Success 200 {object} SellItemsResponse
// @Failure 400 {object} SellItemsResponse
// @Failure 401 {object} SellItemsResponse
// @Failure 403 {object} SellItemsResponse
// @Failure 409 {object} SellItemsResponse
// @Failure 500 {object} SellItemsResponse
// @Router /api/v1/inventory/sell [post]
func SellItems(c *gin.Context) {
	// init req
	var req SellItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, SellItemsResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, SellItemsResponse{ErrorMsg: auth.UnAuthorized})
		return
	}
	// do sell
	order, replayed, err := doSellItems(c.Request.Context(), user.ID, &req)
	if err != nil {
		switch err.Error() {
		case SellItemDuplicated, ItemNotSellable, SellAmountExceeded, QuantityInvalid, InsufficientQuantity, wallet.CurrencyNotFound, wallet.CurrencyDisabled:
			c.JSON(http.StatusBadRequest, SellItemsResponse{ErrorMsg: err.Error()})
		case SellPriceChanged, IdempotencyConflict, DuplicateRequest:
			c.JSON(http.StatusConflict, SellItemsResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[SellItems] sell failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, SellItemsResponse{ErrorMsg: auth.InternalError})
		}
		return
	}
	// 重复提交只返回原出售单, 不重复记录
	if !replayed {
		logger.InfoF(c.Request.Context(), "[SellItems] %d sold %d kinds of items in order %d", user.ID, len(order.Items), order.ID)
	}
	c.JSON(http.StatusOK, SellItemsResponse{Data: order})
}

type ListSellOrdersRequest struct {
	Current   int        `json:"current" form:"current" binding:"min=1"`
	Size      int        `json:"size" form:"size" binding:"min=1,max=100"`
	StartTime *time.Time `json:"start_time" form:"start_time" time_format:"2006-01-02T15:04:05Z07:00"` // RFC3339, 包含
	EndTime   *time.Time `json:"end_time" form:"end_time" time_format:"2006-01-02T15:04:05Z07:00"`     // RFC3339, 不包含
}

type ListSellOrdersResponseData struct {
	Total  int64             `json:"total"`
	Orders []SellOrderDetail `json:"orders"`
}

type ListSellOrdersResponse struct {
	ErrorMsg string                      `json:"error_msg"`
	Data     *ListSellOrdersResponseData `json:"data"`
}

// ListSellOrders 获取当前用户的出售记录
// @Tags inventory
// @Param request query ListSellOrdersRequest true "request query"
// @Produce json
// @Success 200 {object} ListSellOrdersResponse
// @Failure 400 {object} ListSellOrdersResponse
// @Failure 401 {object} ListSellOrdersResponse
// @Failure 403 {object} ListSellOrdersResponse
// @Failure 500 {object} ListSellOrdersResponse
// @Router /api/v1/inventory/sell-orders [get]
func ListSellOrders(c *gin.Context) {
	// init req
	req := &ListSellOrdersRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, ListSellOrdersResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ListSellOrdersResponse{ErrorMsg: auth.UnAuthorized})
		return
	}
	// query
	total, orders, err := QueryUserSellOrders(c.Request.Context(), user.ID, req)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[ListSellOrders] query failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, ListSellOrdersResponse{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, ListSellOrdersResponse{Data: &ListSellOrdersResponseData{Total: total, Orders: orders}})
}
