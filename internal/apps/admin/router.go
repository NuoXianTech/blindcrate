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

package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/mail"
	"github.com/gin-gonic/gin"
)

// Response 统一的 admin 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListUsersRequest 用户列表查询参数
type ListUsersRequest struct {
	Current    int    `json:"current" form:"current" binding:"min=1"`
	Size       int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword    string `json:"keyword" form:"keyword" binding:"max=255"` // 按用户名或邮箱前缀搜索, 不区分大小写
	IsAdmin    *bool  `json:"is_admin" form:"is_admin"`
	IsDisabled *bool  `json:"is_disabled" form:"is_disabled"`
	IsBanned   *bool  `json:"is_banned" form:"is_banned"`
}

// ListUsersResponseData 用户列表分页结果
type ListUsersResponseData struct {
	Total int64       `json:"total"`
	Users []auth.User `json:"users"`
}

// ListUsers
// @Tags admin
// @Summary 获取用户列表
// @Produce json
// @Param request query ListUsersRequest true "request query"
// @Success 200 {object} Response{data=ListUsersResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/users [get]
func ListUsers(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListUsersRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, users, err := QueryUsersList(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListUsers] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListUsersResponseData{Total: total, Users: users}})
}

// UpdateUserRequest 修改用户资料请求体
type UpdateUserRequest struct {
	Username  *string `json:"username"`   // 不传表示不修改
	Nickname  *string `json:"nickname"`   // 不传表示不修改
	AvatarURL *string `json:"avatar_url"` // 不传表示不修改, 传空字符串表示清除头像
}

// UpdateUser
// @Tags admin
// @Summary 修改用户资料
// @Accept json
// @Produce json
// @Param id path int true "user id"
// @Param request body UpdateUserRequest true "request body"
// @Success 200 {object} Response{data=auth.BasicUserInfo}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/users/{id} [patch]
func UpdateUser(c *gin.Context) {
	ctx := c.Request.Context()
	var req UpdateUserRequest
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, err := auth.UpdateUserProfile(ctx, userID, req.Username, req.Nickname, req.AvatarURL)
	if err != nil {
		switch err.Error() {
		case auth.UsernameInvalid, auth.NicknameInvalid, auth.AvatarURLInvalid:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case auth.UserNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case auth.UsernameTaken:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[UpdateUser] update failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionUserUpdate, audit.TargetUser, updated.ID, auth.ProfileChanges(req.Username, req.Nickname, req.AvatarURL))
	logger.InfoF(ctx, "[UpdateUser] %d updated %d %s", operator.ID, updated.ID, updated.Username)
	c.JSON(
		http.StatusOK,
		Response{
			Data: auth.BasicUserInfo{
				ID:        updated.ID,
				Username:  updated.Username,
				Nickname:  updated.Nickname,
				Email:     updated.Email,
				AvatarURL: updated.AvatarURL,
				IsAdmin:   updated.IsAdmin,
			},
		},
	)
}

// BanUserRequest 封禁用户请求体
type BanUserRequest struct {
	BannedUntil time.Time `json:"banned_until" binding:"required"` // RFC3339, 必须晚于当前时间; 永久封禁请使用停用
	Reason      string    `json:"reason" binding:"max=255"`
}

// BanUser
// @Tags admin
// @Summary 封禁用户
// @Description 封禁到指定时间, 并让其所有设备下线
// @Accept json
// @Produce json
// @Param id path int true "user id"
// @Param request body BanUserRequest true "request body"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/users/{id}/ban [put]
func BanUser(c *gin.Context) {
	ctx := c.Request.Context()
	var req BanUserRequest
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	if err := doBanUser(ctx, operator.ID, userID, req.BannedUntil, req.Reason); err != nil {
		switch err.Error() {
		case CannotOperateSelf, BanUntilInvalid:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case auth.UserNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[BanUser] ban failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	// 所有设备下线
	if err := auth.RevokeUserSessions(ctx, userID, ""); err != nil {
		logger.ErrorF(ctx, "[BanUser] revoke sessions failed: %v", err)
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionUserBan, audit.TargetUser, userID, map[string]interface{}{"banned_until": req.BannedUntil, "reason": req.Reason})
	logger.InfoF(ctx, "[BanUser] %d banned %d until %s", operator.ID, userID, req.BannedUntil.Format(time.RFC3339))
	c.JSON(http.StatusOK, Response{})
}

// UnbanUser
// @Tags admin
// @Summary 解除封禁
// @Produce json
// @Param id path int true "user id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/users/{id}/ban [delete]
func UnbanUser(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	if err := doUnbanUser(ctx, operator.ID, userID); err != nil {
		switch err.Error() {
		case CannotOperateSelf:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case auth.UserNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[UnbanUser] unban failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionUserUnban, audit.TargetUser, userID, nil)
	logger.InfoF(ctx, "[UnbanUser] %d unbanned %d", operator.ID, userID)
	c.JSON(http.StatusOK, Response{})
}

// SetUserDisabledRequest 停用或启用用户请求体
type SetUserDisabledRequest struct {
	Disabled *bool `json:"disabled" binding:"required"`
}

// SetUserDisabled
// @Tags admin
// @Summary 停用或启用用户
// @Description 停用时让其所有设备下线
// @Accept json
// @Produce json
// @Param id path int true "user id"
// @Param request body SetUserDisabledRequest true "request body"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/users/{id}/disabled [put]
func SetUserDisabled(c *gin.Context) {
	ctx := c.Request.Context()
	var req SetUserDisabledRequest
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	if err := doSetUserDisabled(ctx, operator.ID, userID, *req.Disabled); err != nil {
		switch err.Error() {
		case CannotOperateSelf:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case auth.UserNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[SetUserDisabled] set failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	// 停用时所有设备下线
	if *req.Disabled {
		if err := auth.RevokeUserSessions(ctx, userID, ""); err != nil {
			logger.ErrorF(ctx, "[SetUserDisabled] revoke sessions failed: %v", err)
		}
	}
	action := audit.ActionUserEnable
	if *req.Disabled {
		action = audit.ActionUserDisable
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, action, audit.TargetUser, userID, nil)
	logger.InfoF(ctx, "[SetUserDisabled] %d set %d disabled=%t", operator.ID, userID, *req.Disabled)
	c.JSON(http.StatusOK, Response{})
}

// ListAuditLogsRequest 操作日志查询参数
type ListAuditLogsRequest struct {
	Current    int        `json:"current" form:"current" binding:"min=1"`
	Size       int        `json:"size" form:"size" binding:"min=1,max=100"`
	ActorID    *uint64    `json:"actor_id" form:"actor_id"`
	Scope      string     `json:"scope" form:"scope" binding:"omitempty,oneof=user admin"`
	Action     string     `json:"action" form:"action" binding:"max=64"`
	TargetType string     `json:"target_type" form:"target_type" binding:"max=32"`
	TargetID   *uint64    `json:"target_id" form:"target_id"`
	StartTime  *time.Time `json:"start_time" form:"start_time" time_format:"2006-01-02T15:04:05Z07:00"` // RFC3339, 包含
	EndTime    *time.Time `json:"end_time" form:"end_time" time_format:"2006-01-02T15:04:05Z07:00"`     // RFC3339, 不包含
}

// ListAuditLogsResponseData 操作日志分页结果
type ListAuditLogsResponseData struct {
	Total int64          `json:"total"`
	Logs  []AuditLogItem `json:"logs"`
}

// ListAuditLogs
// @Tags admin
// @Summary 获取操作日志列表
// @Description 包含用户和管理员的操作
// @Produce json
// @Param request query ListAuditLogsRequest true "request query"
// @Success 200 {object} Response{data=ListAuditLogsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/audit-logs [get]
func ListAuditLogs(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListAuditLogsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, logs, err := QueryAuditLogs(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListAuditLogs] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListAuditLogsResponseData{Total: total, Logs: logs}})
}

// ListCurrenciesResponseData 全部货币定义
type ListCurrenciesResponseData struct {
	Currencies []wallet.Currency `json:"currencies"`
}

// ListCurrencies
// @Tags admin
// @Summary 获取全部货币定义
// @Description 包含已停用的货币
// @Produce json
// @Success 200 {object} Response{data=ListCurrenciesResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/wallet/currencies [get]
func ListCurrencies(c *gin.Context) {
	ctx := c.Request.Context()
	currencies, err := wallet.QueryCurrencies(ctx)
	if err != nil {
		logger.ErrorF(ctx, "[ListCurrencies] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListCurrenciesResponseData{Currencies: currencies}})
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
// @Success 200 {object} Response{data=ListWalletTransactionsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/wallet/transactions [get]
func ListWalletTransactions(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListWalletTransactionsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, transactions, err := QueryWalletTransactions(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListWalletTransactions] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListWalletTransactionsResponseData{Total: total, Transactions: transactions}})
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
// @Success 200 {object} Response{data=ListUserBalancesResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/users/{id}/wallet/balances [get]
func ListUserBalances(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	balances, err := queryUserBalances(ctx, userID)
	if err != nil {
		if err.Error() == auth.UserNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[ListUserBalances] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListUserBalancesResponseData{Balances: balances}})
}

// AdjustUserBalanceRequest 调账请求体
type AdjustUserBalanceRequest struct {
	Currency       string `json:"currency" binding:"required,max=32"`
	Amount         int64  `json:"amount" binding:"required"`                            // 正数加款, 负数扣款, 扣款后余额不能为负
	Remark         string `json:"remark" binding:"required"`                            // 调账原因, 用户可在流水中看到
	IdempotencyKey string `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次打开调账表单生成一个, 重复提交只生效一次
}

// AdjustUserBalance
// @Tags admin
// @Summary 给用户调账
// @Description 写入一条 admin_adjust 流水, 不能给自己调账; 重复提交返回原流水
// @Accept json
// @Produce json
// @Param id path int true "user id"
// @Param request body AdjustUserBalanceRequest true "request body"
// @Success 200 {object} Response{data=wallet.Transaction}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/users/{id}/wallet/adjust [post]
func AdjustUserBalance(c *gin.Context) {
	ctx := c.Request.Context()
	var req AdjustUserBalanceRequest
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	txn, replayed, err := doAdjustUserBalance(ctx, operator.ID, userID, &req)
	if err != nil {
		switch err.Error() {
		case CannotOperateSelf, RemarkInvalid, wallet.AmountInvalid, wallet.CurrencyNotFound, wallet.CurrencyDisabled, wallet.InsufficientBalance:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case auth.UserNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case wallet.IdempotencyConflict, wallet.DuplicateRequest:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[AdjustUserBalance] adjust failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	// 重复提交只返回原流水, 不重复记录
	if !replayed {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionWalletAdjust, audit.TargetUser, userID, map[string]interface{}{
			"transaction_id": txn.ID,
			"currency":       txn.Currency,
			"amount":         txn.Amount,
			"balance_after":  txn.BalanceAfter,
			"remark":         txn.Remark,
		})
		logger.InfoF(ctx, "[AdjustUserBalance] %d adjusted %d %s by %d, balance %d", operator.ID, userID, txn.Currency, txn.Amount, txn.BalanceAfter)
	}
	c.JSON(http.StatusOK, Response{Data: txn})
}

// ListItemsRequest 物品列表查询参数
type ListItemsRequest struct {
	Current   int    `json:"current" form:"current" binding:"min=1"`
	Size      int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"` // 按名称搜索, 不区分大小写
	Quality   *int16 `json:"quality" form:"quality" binding:"omitempty,min=1,max=7"`
	IsEnabled *bool  `json:"is_enabled" form:"is_enabled"`
}

// ListItemsResponseData 物品列表分页结果
type ListItemsResponseData struct {
	Total int64          `json:"total"`
	Items []ItemListItem `json:"items"`
}

// ListItems
// @Tags admin
// @Summary 获取物品列表
// @Description 包含已下架的物品, 不包含已删除的物品
// @Produce json
// @Param request query ListItemsRequest true "request query"
// @Success 200 {object} Response{data=ListItemsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/items [get]
func ListItems(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListItemsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, items, err := QueryItems(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListItems] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListItemsResponseData{Total: total, Items: items}})
}

// ItemRequest 新建或修改物品请求体
type ItemRequest struct {
	Name         string `json:"name" binding:"required,max=64"`
	Description  string `json:"description" binding:"max=255"`
	Quality      int16  `json:"quality" binding:"required,min=1,max=7"` // 1-7: 灰/绿/蓝/紫/金/红/炫彩
	ImageURL     string `json:"image_url" binding:"omitempty,max=1024,http_url"`
	SellCurrency string `json:"sell_currency" binding:"required,max=32"`
	SellPrice    int64  `json:"sell_price" binding:"min=0,max=1000000000000"` // 0 表示不可卖出
	Weight       int32  `json:"weight" binding:"min=0,max=1000000"`           // 掉落权重, 0 表示不会掉落
	Width        int16  `json:"width" binding:"required,min=1,max=10"`        // 在保险柜中占的格数
	Height       int16  `json:"height" binding:"required,min=1,max=10"`
	IsEnabled    bool   `json:"is_enabled"` // 下架后不会掉落, 也不能再发放, 已有的保留
}

// CreateItem
// @Tags admin
// @Summary 新建物品
// @Accept json
// @Produce json
// @Param request body ItemRequest true "request body"
// @Success 200 {object} Response{data=item.Item}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/items [post]
func CreateItem(c *gin.Context) {
	ctx := c.Request.Context()
	var req ItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateItem(ctx, &req)
	if err != nil {
		switch err.Error() {
		case item.ItemNameInvalid, wallet.CurrencyNotFound:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case item.ItemNameTaken:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[CreateItem] create failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionItemCreate, audit.TargetItem, created.ID, itemFields(created))
	logger.InfoF(ctx, "[CreateItem] %d created item %d %s", operator.ID, created.ID, created.Name)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateItem
// @Tags admin
// @Summary 修改物品
// @Description 需提交全部字段; 修改名称不影响已有流水中的名称
// @Accept json
// @Produce json
// @Param id path int true "item id"
// @Param request body ItemRequest true "request body"
// @Success 200 {object} Response{data=item.Item}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/items/{id} [put]
func UpdateItem(c *gin.Context) {
	ctx := c.Request.Context()
	var req ItemRequest
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateItem(ctx, itemID, &req)
	if err != nil {
		switch err.Error() {
		case item.ItemNameInvalid, wallet.CurrencyNotFound:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case item.ItemNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case item.ItemNameTaken:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[UpdateItem] update failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionItemUpdate, audit.TargetItem, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateItem] %d updated item %d %s", operator.ID, updated.ID, updated.Name)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteItemResponseData 删除物品的结果
type DeleteItemResponseData struct {
	HolderCount int64 `json:"holder_count"` // 删除时持有该物品的用户数
}

// DeleteItem
// @Tags admin
// @Summary 删除物品
// @Description 已持有的玩家背包保留该物品并变为占位物品, 物品 ID 不会被复用
// @Produce json
// @Param id path int true "item id"
// @Success 200 {object} Response{data=DeleteItemResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/items/{id} [delete]
func DeleteItem(c *gin.Context) {
	ctx := c.Request.Context()
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, holders, err := doDeleteItem(ctx, itemID)
	if err != nil {
		if err.Error() == item.ItemNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteItem] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionItemDelete, audit.TargetItem, deleted.ID, map[string]interface{}{
		"name":         deleted.Name,
		"holder_count": holders,
	})
	logger.InfoF(ctx, "[DeleteItem] %d deleted item %d %s, %d holders", operator.ID, deleted.ID, deleted.Name, holders)
	c.JSON(http.StatusOK, Response{Data: DeleteItemResponseData{HolderCount: holders}})
}

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
// @Success 200 {object} Response{data=ListUserInventoryResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/users/{id}/inventory [get]
func ListUserInventory(c *gin.Context) {
	ctx := c.Request.Context()
	req := &inventory.ListItemsRequest{}
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindQuery(req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, items, err := queryUserInventory(ctx, userID, req)
	if err != nil {
		if err.Error() == auth.UserNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[ListUserInventory] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListUserInventoryResponseData{Total: total, Items: items}})
}

// AdjustUserInventoryRequest 发放或扣除物品请求体
type AdjustUserInventoryRequest struct {
	ItemID         uint64 `json:"item_id" binding:"required"`
	Quantity       int64  `json:"quantity" binding:"required"`                          // 正数发放, 负数扣除, 扣除后数量不能为负
	Remark         string `json:"remark" binding:"required"`                            // 调整原因, 用户可在流水中看到
	IdempotencyKey string `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次打开表单生成一个, 重复提交只生效一次
}

// AdjustUserInventory
// @Tags admin
// @Summary 给用户发放或扣除物品
// @Description 写入一条 admin_adjust 流水, 不能给自己调整; 重复提交返回原流水
// @Accept json
// @Produce json
// @Param id path int true "user id"
// @Param request body AdjustUserInventoryRequest true "request body"
// @Success 200 {object} Response{data=inventory.Transaction}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/users/{id}/inventory/adjust [post]
func AdjustUserInventory(c *gin.Context) {
	ctx := c.Request.Context()
	var req AdjustUserInventoryRequest
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	txn, replayed, err := doAdjustUserInventory(ctx, operator.ID, userID, &req)
	if err != nil {
		switch err.Error() {
		case CannotOperateSelf, RemarkInvalid, inventory.QuantityInvalid, item.ItemNotFound, item.ItemDisabled, inventory.InsufficientQuantity:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case auth.UserNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case inventory.IdempotencyConflict, inventory.DuplicateRequest:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[AdjustUserInventory] adjust failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	// 重复提交只返回原流水, 不重复记录
	if !replayed {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionInventoryAdjust, audit.TargetUser, userID, map[string]interface{}{
			"transaction_id": txn.ID,
			"item_id":        txn.ItemID,
			"item_name":      txn.ItemName,
			"quantity":       txn.Quantity,
			"quantity_after": txn.QuantityAfter,
			"remark":         txn.Remark,
		})
		logger.InfoF(ctx, "[AdjustUserInventory] %d adjusted %d item %d by %d, quantity %d", operator.ID, userID, txn.ItemID, txn.Quantity, txn.QuantityAfter)
	}
	c.JSON(http.StatusOK, Response{Data: txn})
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
// @Success 200 {object} Response{data=ListInventoryTransactionsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/inventory/transactions [get]
func ListInventoryTransactions(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListInventoryTransactionsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, transactions, err := QueryInventoryTransactions(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListInventoryTransactions] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListInventoryTransactionsResponseData{Total: total, Transactions: transactions}})
}

// ListSettingsResponseData 全部系统设置
type ListSettingsResponseData struct {
	Groups []settings.GroupView `json:"groups"`
}

// ListSettings
// @Tags admin
// @Summary 获取全部系统设置
// @Description 包含表单描述和当前值, 密钥字段不返回明文
// @Produce json
// @Success 200 {object} Response{data=ListSettingsResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings [get]
func ListSettings(c *gin.Context) {
	ctx := c.Request.Context()
	groups, err := settings.QueryGroups(ctx)
	if err != nil {
		logger.ErrorF(ctx, "[ListSettings] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListSettingsResponseData{Groups: groups}})
}

// UpdateSettingRequest 保存设置组请求体
type UpdateSettingRequest struct {
	Value        json.RawMessage `json:"value" binding:"required" swaggertype:"object"` // 设置组的字段, 只需提交改动的字段, 没有提交的保持不变, 密钥字段留空表示不修改
	ClearSecrets []string        `json:"clear_secrets" binding:"max=32"`                // 需要清空的密钥字段
}

// UpdateSetting
// @Tags admin
// @Summary 保存一个设置组
// @Description 保存后立即生效
// @Accept json
// @Produce json
// @Param key path string true "setting group key"
// @Param request body UpdateSettingRequest true "request body"
// @Success 200 {object} Response{data=settings.GroupView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings/{key} [put]
func UpdateSetting(c *gin.Context) {
	ctx := c.Request.Context()
	var req UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	group, changes, err := settings.Update(ctx, operator.ID, c.Param("key"), req.Value, req.ClearSecrets)
	if err != nil {
		switch err.Error() {
		case settings.GroupNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case settings.SaveFailed:
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: err.Error()})
		default:
			// 其余均为设置内容校验失败, 提示可直接展示
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		}
		return
	}
	// 没有任何修改时不记录
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionSettingsUpdate, audit.TargetSetting, 0, map[string]interface{}{
			"key":     group.Key,
			"changes": changes,
		})
		logger.InfoF(ctx, "[UpdateSetting] %d updated %s", operator.ID, group.Key)
	}
	c.JSON(http.StatusOK, Response{Data: group})
}

// TestMailRequest 测试邮件请求体
type TestMailRequest struct {
	To string `json:"to" binding:"required"` // 收件人
}

// TestMail
// @Tags admin
// @Summary 发送测试邮件
// @Description 使用已保存的邮件设置发送, 未启用邮件发送时也可以测试; 发送失败时返回 SMTP 的错误信息
// @Accept json
// @Produce json
// @Param request body TestMailRequest true "request body"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Failure 502 {object} Response
// @Router /api/v1/admin/settings/mail/test [post]
func TestMail(c *gin.Context) {
	ctx := c.Request.Context()
	var req TestMailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	to, mailConfig, err := prepareTestMail(ctx, &req)
	if err != nil {
		switch err.Error() {
		case auth.EmailInvalid, MailTestIncomplete:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[TestMail] query settings failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	// 发送失败时把 SMTP 返回的错误交给管理员排查
	sendErr := mail.Deliver(ctx, mailConfig, to, "测试邮件", "这是一封测试邮件, 收到说明邮件设置可用。")
	detail := map[string]interface{}{"to": to, "host": mailConfig.Host, "port": mailConfig.Port, "success": sendErr == nil}
	if sendErr != nil {
		detail["error"] = sendErr.Error()
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionMailTest, audit.TargetSetting, 0, detail)
	if sendErr != nil {
		logger.WarnF(ctx, "[TestMail] %d send to %s failed: %v", operator.ID, to, sendErr)
		c.JSON(http.StatusBadGateway, Response{ErrorMsg: fmt.Sprintf("%s: %s", MailTestFailed, sendErr.Error())})
		return
	}
	c.JSON(http.StatusOK, Response{})
}
