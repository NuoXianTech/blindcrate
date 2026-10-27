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
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

type ListBalancesResponseData struct {
	Balances []BalanceItem `json:"balances"`
}

type ListBalancesResponse struct {
	ErrorMsg string                    `json:"error_msg"`
	Data     *ListBalancesResponseData `json:"data"`
}

// ListBalances 获取当前用户的各货币余额
// @Tags wallet
// @Produce json
// @Success 200 {object} ListBalancesResponse
// @Failure 401 {object} ListBalancesResponse
// @Failure 403 {object} ListBalancesResponse
// @Failure 500 {object} ListBalancesResponse
// @Router /api/v1/wallet/balances [get]
func ListBalances(c *gin.Context) {
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ListBalancesResponse{ErrorMsg: auth.UnAuthorized})
		return
	}
	// query
	balances, err := QueryBalances(c.Request.Context(), user.ID, false)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[ListBalances] query failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, ListBalancesResponse{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, ListBalancesResponse{Data: &ListBalancesResponseData{Balances: balances}})
}

type ListTransactionsRequest struct {
	Current   int        `json:"current" form:"current" binding:"min=1"`
	Size      int        `json:"size" form:"size" binding:"min=1,max=100"`
	Currency  string     `json:"currency" form:"currency" binding:"max=32"`
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

// ListTransactions 获取当前用户的余额流水
// @Tags wallet
// @Param request query ListTransactionsRequest true "request query"
// @Produce json
// @Success 200 {object} ListTransactionsResponse
// @Failure 400 {object} ListTransactionsResponse
// @Failure 401 {object} ListTransactionsResponse
// @Failure 403 {object} ListTransactionsResponse
// @Failure 500 {object} ListTransactionsResponse
// @Router /api/v1/wallet/transactions [get]
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
