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

package walletadmin

import (
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// ListCurrenciesResponseData 全部货币定义
type ListCurrenciesResponseData struct {
	Currencies []wallet.Currency `json:"currencies"`
}

// ListCurrencies
// @Tags admin
// @Summary 获取全部货币定义
// @Description 包含已停用的货币
// @Produce json
// @Success 200 {object} reply.Response{data=ListCurrenciesResponseData}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/currencies [get]
func ListCurrencies(c *gin.Context) {
	ctx := c.Request.Context()
	currencies, err := wallet.QueryCurrencies(ctx)
	if err != nil {
		reply.Fail(c, err, "[ListCurrencies] query failed")
		return
	}
	reply.OK(c, ListCurrenciesResponseData{Currencies: currencies})
}

// CurrencyRequest 修改货币请求体, 代码创建后不能修改
type CurrencyRequest struct {
	Name      string `json:"name" binding:"required,max=32"`
	Decimals  int16  `json:"decimals" binding:"min=0,max=6"`                // 展示用小数位, 已被使用的货币不能修改
	SortOrder int32  `json:"sort_order" binding:"min=-1000000,max=1000000"` // 余额和下拉框中从小到大排列
	IsEnabled bool   `json:"is_enabled"`                                    // 停用后不能再入账或扣款, 基础货币不能停用
}

// CreateCurrencyRequest 新建货币请求体
type CreateCurrencyRequest struct {
	Code string `json:"code" binding:"required,max=32"` // 自然键, 小写字母、数字和下划线
	CurrencyRequest
}

// CreateCurrency
// @Tags admin
// @Summary 新建货币
// @Accept json
// @Produce json
// @Param request body CreateCurrencyRequest true "request body"
// @Success 200 {object} reply.Response{data=wallet.Currency}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/currencies [post]
func CreateCurrency(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreateCurrencyRequest
	if !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	created, err := doCreateCurrency(ctx, &req)
	if err != nil {
		reply.Fail(c, err, "[CreateCurrency] create failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, wallet.ActionCurrencyCreate, wallet.TargetCurrency, 0, currencyFields(created))
	logger.InfoF(ctx, "[CreateCurrency] %d created currency %s", operator.ID, created.Code)
	reply.OK(c, created)
}

// UpdateCurrency
// @Tags admin
// @Summary 修改货币
// @Description 需提交全部字段, 代码不能修改; 已被余额、物品、地图或掉落池使用的货币不能修改小数位, 基础货币不能停用
// @Accept json
// @Produce json
// @Param code path string true "currency code"
// @Param request body CurrencyRequest true "request body"
// @Success 200 {object} reply.Response{data=wallet.Currency}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/currencies/{code} [put]
func UpdateCurrency(c *gin.Context) {
	ctx := c.Request.Context()
	var req CurrencyRequest
	if !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	updated, changes, err := doUpdateCurrency(ctx, c.Param("code"), &req)
	if err != nil {
		reply.Fail(c, err, "[UpdateCurrency] update failed")
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, wallet.ActionCurrencyUpdate, wallet.TargetCurrency, 0, map[string]interface{}{
			"code":    updated.Code,
			"changes": changes,
		})
		logger.InfoF(ctx, "[UpdateCurrency] %d updated currency %s", operator.ID, updated.Code)
	}
	reply.OK(c, updated)
}

// ListWalletTransactionsRequest 余额流水查询参数
type ListWalletTransactionsRequest struct {
	Current    int        `json:"current" form:"current" binding:"min=1"`
	Size       int        `json:"size" form:"size" binding:"min=1,max=100"`
	UserID     *uint64    `json:"user_id" form:"user_id"`
	OperatorID *uint64    `json:"operator_id" form:"operator_id"`
	Currency   string     `json:"currency" form:"currency" binding:"max=32"`
	BizType    string     `json:"biz_type" form:"biz_type" binding:"max=32"`
	BizID      string     `json:"biz_id" form:"biz_id" binding:"max=64"`
	StartTime  *time.Time `json:"start_time" form:"start_time" time_format:"2006-01-02T15:04:05Z07:00"` // RFC3339, 包含
	EndTime    *time.Time `json:"end_time" form:"end_time" time_format:"2006-01-02T15:04:05Z07:00"`     // RFC3339, 不包含
}

// ListWalletTransactionsResponseData 余额流水分页结果
type ListWalletTransactionsResponseData struct {
	Total        int64                   `json:"total"`
	Transactions []WalletTransactionItem `json:"transactions"`
}

// ListWalletTransactions
// @Tags admin
// @Summary 获取全部用户的余额流水
// @Produce json
// @Param request query ListWalletTransactionsRequest true "request query"
// @Success 200 {object} reply.Response{data=ListWalletTransactionsResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/wallet/transactions [get]
func ListWalletTransactions(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListWalletTransactionsRequest{}
	if !reply.Query(c, req) {
		return
	}
	total, transactions, err := QueryWalletTransactions(ctx, req)
	if err != nil {
		reply.Fail(c, err, "[ListWalletTransactions] query failed")
		return
	}
	reply.OK(c, ListWalletTransactionsResponseData{Total: total, Transactions: transactions})
}

// ListUserBalancesResponseData 指定用户的各货币余额
type ListUserBalancesResponseData struct {
	Balances []wallet.BalanceItem `json:"balances"`
}

// ListUserBalances
// @Tags admin
// @Summary 获取指定用户的各货币余额
// @Description 包含已停用的货币
// @Produce json
// @Param id path int true "user id"
// @Success 200 {object} reply.Response{data=ListUserBalancesResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/users/{id}/wallet/balances [get]
func ListUserBalances(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := reply.ID(c, "id")
	if !ok {
		return
	}
	balances, err := queryUserBalances(ctx, userID)
	if err != nil {
		reply.Fail(c, err, "[ListUserBalances] query failed")
		return
	}
	reply.OK(c, ListUserBalancesResponseData{Balances: balances})
}

// AdjustUserBalanceRequest 调账请求体
type AdjustUserBalanceRequest struct {
	Currency       string `json:"currency" binding:"required,max=32"`
	Amount         int64  `json:"amount" binding:"required"`                            // 正数加款, 负数扣款, 扣款后余额不能为负
	Remark         string `json:"remark"`                                               // 调账原因, 可以为空, 用户可在流水中看到
	IdempotencyKey string `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次打开调账表单生成一个, 重复提交只生效一次
}

// AdjustUserBalance
// @Tags admin
// @Summary 给用户调账
// @Description 写入一条 admin_adjust 流水, 也可以给自己调账; 重复提交返回原流水
// @Accept json
// @Produce json
// @Param id path int true "user id"
// @Param request body AdjustUserBalanceRequest true "request body"
// @Success 200 {object} reply.Response{data=wallet.Transaction}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/users/{id}/wallet/adjust [post]
func AdjustUserBalance(c *gin.Context) {
	ctx := c.Request.Context()
	var req AdjustUserBalanceRequest
	userID, ok := reply.ID(c, "id")
	if !ok || !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	txn, replayed, err := doAdjustUserBalance(ctx, operator.ID, userID, &req)
	if err != nil {
		reply.Fail(c, err, "[AdjustUserBalance] adjust failed")
		return
	}
	// 重复提交只返回原流水, 不重复记录和推送
	if !replayed {
		wallet.PushChanges(ctx, txn)
		audit.Record(c, operator.ID, audit.ScopeAdmin, wallet.ActionWalletAdjust, auth.TargetUser, userID, map[string]interface{}{
			"transaction_id": txn.ID,
			"currency":       txn.Currency,
			"amount":         txn.Amount,
			"balance_after":  txn.BalanceAfter,
			"remark":         txn.Remark,
		})
		logger.InfoF(ctx, "[AdjustUserBalance] %d adjusted %d %s by %d, balance %d", operator.ID, userID, txn.Currency, txn.Amount, txn.BalanceAfter)
	}
	reply.OK(c, txn)
}
