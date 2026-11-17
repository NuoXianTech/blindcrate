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

package inventoryadmin

import (
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// ListUserInventoryResponseData 指定用户背包物品分页结果
type ListUserInventoryResponseData struct {
	Total int64                      `json:"total"`
	Items []inventory.UserItemDetail `json:"items"`
}

// ListUserInventory
// @Tags admin
// @Summary 获取指定用户背包中的物品
// @Produce json
// @Param id path int true "user id"
// @Param request query inventory.ListItemsRequest true "request query"
// @Success 200 {object} reply.Response{data=ListUserInventoryResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/users/{id}/inventory/items [get]
func ListUserInventory(c *gin.Context) {
	ctx := c.Request.Context()
	req := &inventory.ListItemsRequest{}
	userID, ok := reply.ID(c, "id")
	if !ok || !reply.Query(c, req) {
		return
	}
	total, items, err := queryUserInventory(ctx, userID, req)
	if err != nil {
		reply.Fail(c, err, "[ListUserInventory] query failed")
		return
	}
	reply.OK(c, ListUserInventoryResponseData{Total: total, Items: items})
}

// AdjustUserInventoryRequest 发放或扣除物品请求体
type AdjustUserInventoryRequest struct {
	ItemID         uint64 `json:"item_id" binding:"required"`
	Quantity       int64  `json:"quantity" binding:"required"`                          // 正数发放, 负数扣除, 扣除后数量不能为负
	Remark         string `json:"remark"`                                               // 调整原因, 可以为空, 用户可在流水中看到
	IdempotencyKey string `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次打开表单生成一个, 重复提交只生效一次
}

// AdjustUserInventory
// @Tags admin
// @Summary 给用户发放或扣除物品
// @Description 写入一条 admin_adjust 流水, 也可以给自己调整; 重复提交返回原流水
// @Accept json
// @Produce json
// @Param id path int true "user id"
// @Param request body AdjustUserInventoryRequest true "request body"
// @Success 200 {object} reply.Response{data=inventory.Transaction}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/users/{id}/inventory/adjust [post]
func AdjustUserInventory(c *gin.Context) {
	ctx := c.Request.Context()
	var req AdjustUserInventoryRequest
	userID, ok := reply.ID(c, "id")
	if !ok || !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	txn, replayed, err := doAdjustUserInventory(ctx, operator.ID, userID, &req)
	if err != nil {
		reply.Fail(c, err, "[AdjustUserInventory] adjust failed")
		return
	}
	// 重复提交只返回原流水, 不重复记录和推送
	if !replayed {
		inventory.PushChanges(ctx, txn)
		audit.Record(c, operator.ID, audit.ScopeAdmin, inventory.ActionInventoryAdjust, auth.TargetUser, userID, map[string]interface{}{
			"transaction_id": txn.ID,
			"item_id":        txn.ItemID,
			"item_name":      txn.ItemName,
			"quantity":       txn.Quantity,
			"quantity_after": txn.QuantityAfter,
			"remark":         txn.Remark,
		})
		logger.InfoF(ctx, "[AdjustUserInventory] %d adjusted %d item %d by %d, quantity %d", operator.ID, userID, txn.ItemID, txn.Quantity, txn.QuantityAfter)
	}
	reply.OK(c, txn)
}

// ListInventoryTransactionsRequest 背包流水查询参数
type ListInventoryTransactionsRequest struct {
	Current    int        `json:"current" form:"current" binding:"min=1"`
	Size       int        `json:"size" form:"size" binding:"min=1,max=100"`
	UserID     *uint64    `json:"user_id" form:"user_id"`
	OperatorID *uint64    `json:"operator_id" form:"operator_id"`
	ItemID     *uint64    `json:"item_id" form:"item_id"`
	BizType    string     `json:"biz_type" form:"biz_type" binding:"max=32"`
	BizID      string     `json:"biz_id" form:"biz_id" binding:"max=64"`
	StartTime  *time.Time `json:"start_time" form:"start_time" time_format:"2006-01-02T15:04:05Z07:00"` // RFC3339, 包含
	EndTime    *time.Time `json:"end_time" form:"end_time" time_format:"2006-01-02T15:04:05Z07:00"`     // RFC3339, 不包含
}

// ListInventoryTransactionsResponseData 背包流水分页结果
type ListInventoryTransactionsResponseData struct {
	Total        int64                      `json:"total"`
	Transactions []InventoryTransactionItem `json:"transactions"`
}

// ListInventoryTransactions
// @Tags admin
// @Summary 获取全部用户的背包流水
// @Produce json
// @Param request query ListInventoryTransactionsRequest true "request query"
// @Success 200 {object} reply.Response{data=ListInventoryTransactionsResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/inventory/transactions [get]
func ListInventoryTransactions(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListInventoryTransactionsRequest{}
	if !reply.Query(c, req) {
		return
	}
	total, transactions, err := QueryInventoryTransactions(ctx, req)
	if err != nil {
		reply.Fail(c, err, "[ListInventoryTransactions] query failed")
		return
	}
	reply.OK(c, ListInventoryTransactionsResponseData{Total: total, Transactions: transactions})
}
