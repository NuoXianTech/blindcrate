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
	"net/http"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/ledger"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 wallet 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListBalancesResponseData 当前用户的各货币余额
type ListBalancesResponseData struct {
	Balances []BalanceItem `json:"balances"`
}

// ListBalances
// @Tags wallet
// @Summary 获取当前用户的各货币余额
// @Description 没有余额行的货币按 0 返回, 已停用且没有余额的货币不返回
// @Produce json
// @Success 200 {object} Response{data=ListBalancesResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/wallet/balances [get]
func ListBalances(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	balances, err := QueryBalances(ctx, user.ID, false)
	if err != nil {
		logger.ErrorF(ctx, "[ListBalances] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListBalancesResponseData{Balances: balances}})
}

// ListTransactionsRequest 余额流水查询参数
type ListTransactionsRequest struct {
	Current   int        `json:"current" form:"current" binding:"min=1"`
	Size      int        `json:"size" form:"size" binding:"min=1,max=100"`
	Currency  string     `json:"currency" form:"currency" binding:"max=32"`
	BizType   string     `json:"biz_type" form:"biz_type" binding:"max=32"`
	StartTime *time.Time `json:"start_time" form:"start_time" time_format:"2006-01-02T15:04:05Z07:00"` // RFC3339, 包含
	EndTime   *time.Time `json:"end_time" form:"end_time" time_format:"2006-01-02T15:04:05Z07:00"`     // RFC3339, 不包含
}

// ListTransactionsResponseData 余额流水分页结果
type ListTransactionsResponseData struct {
	Total        int64             `json:"total"`
	Transactions []TransactionItem `json:"transactions"`
}

// ListTransactions
// @Tags wallet
// @Summary 获取当前用户的余额流水
// @Produce json
// @Param request query ListTransactionsRequest true "request query"
// @Success 200 {object} Response{data=ListTransactionsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/wallet/transactions [get]
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

// ListBizTypesResponseData 业务类型和名称
type ListBizTypesResponseData struct {
	BizTypes []ledger.BizType `json:"biz_types"`
}

// ListBizTypes
// @Tags wallet
// @Summary 获取余额流水的业务类型和名称
// @Description 用于展示流水和筛选, 按登记顺序
// @Produce json
// @Success 200 {object} Response{data=ListBizTypesResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Router /api/v1/wallet/biz-types [get]
func ListBizTypes(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Data: ListBizTypesResponseData{BizTypes: ledger.BizTypes(ledger.KindWallet)}})
}
