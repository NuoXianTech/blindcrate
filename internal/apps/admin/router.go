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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/mail"
	"github.com/NuoXianTech/blindcrate/internal/ws"
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
	if err := ws.DisconnectUser(ctx, userID); err != nil {
		logger.ErrorF(ctx, "[BanUser] disconnect websocket failed: %v", err)
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
		if err := ws.DisconnectUser(ctx, userID); err != nil {
			logger.ErrorF(ctx, "[SetUserDisabled] disconnect websocket failed: %v", err)
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
	// 重复提交只返回原流水, 不重复记录和推送
	if !replayed {
		wallet.PushChanges(ctx, txn)
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

// ListLootPoolsRequest 掉落池列表查询参数
type ListLootPoolsRequest struct {
	Current   int    `json:"current" form:"current" binding:"min=1"`
	Size      int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"` // 按代码或名称搜索, 不区分大小写
	IsEnabled *bool  `json:"is_enabled" form:"is_enabled"`
}

// ListLootPoolsResponseData 掉落池列表分页结果
type ListLootPoolsResponseData struct {
	Total int64              `json:"total"`
	Pools []LootPoolListItem `json:"pools"`
}

// ListLootPools
// @Tags admin
// @Summary 获取掉落池列表
// @Produce json
// @Param request query ListLootPoolsRequest true "request query"
// @Success 200 {object} Response{data=ListLootPoolsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/loot-pools [get]
func ListLootPools(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListLootPoolsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, pools, err := QueryLootPools(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListLootPools] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListLootPoolsResponseData{Total: total, Pools: pools}})
}

// LootPoolRequest 新建或修改掉落池请求体
type LootPoolRequest struct {
	Code        string `json:"code" binding:"required,max=64"` // 玩法中引用的代码, 小写字母、数字和下划线
	Name        string `json:"name" binding:"required,max=64"`
	Description string `json:"description" binding:"max=255"`
	IsEnabled   bool   `json:"is_enabled"` // 停用后不能再抽取
}

// CreateLootPool
// @Tags admin
// @Summary 新建掉落池
// @Description 新建后没有条目, 需要再替换条目
// @Accept json
// @Produce json
// @Param request body LootPoolRequest true "request body"
// @Success 200 {object} Response{data=loot.Pool}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/loot-pools [post]
func CreateLootPool(c *gin.Context) {
	ctx := c.Request.Context()
	var req LootPoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateLootPool(ctx, &req)
	if err != nil {
		switch err.Error() {
		case loot.PoolCodeInvalid, loot.PoolNameInvalid:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case loot.PoolCodeTaken:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[CreateLootPool] create failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLootPoolCreate, audit.TargetLootPool, created.ID, lootPoolFields(created))
	logger.InfoF(ctx, "[CreateLootPool] %d created loot pool %d %s", operator.ID, created.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateLootPool
// @Tags admin
// @Summary 修改掉落池
// @Description 需提交全部字段; 修改代码后, 仍使用旧代码的玩法会找不到掉落池
// @Accept json
// @Produce json
// @Param id path int true "loot pool id"
// @Param request body LootPoolRequest true "request body"
// @Success 200 {object} Response{data=loot.Pool}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/loot-pools/{id} [put]
func UpdateLootPool(c *gin.Context) {
	ctx := c.Request.Context()
	var req LootPoolRequest
	poolID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateLootPool(ctx, poolID, &req)
	if err != nil {
		switch err.Error() {
		case loot.PoolCodeInvalid, loot.PoolNameInvalid:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case loot.PoolNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case loot.PoolCodeTaken:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[UpdateLootPool] update failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLootPoolUpdate, audit.TargetLootPool, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateLootPool] %d updated loot pool %d %s", operator.ID, updated.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteLootPoolResponseData 删除掉落池的结果
type DeleteLootPoolResponseData struct {
	EntryCount int64 `json:"entry_count"` // 一同删除的条目数
}

// DeleteLootPool
// @Tags admin
// @Summary 删除掉落池
// @Description 连同条目一起删除, 仍使用该代码的玩法会找不到掉落池
// @Produce json
// @Param id path int true "loot pool id"
// @Success 200 {object} Response{data=DeleteLootPoolResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/loot-pools/{id} [delete]
func DeleteLootPool(c *gin.Context) {
	ctx := c.Request.Context()
	poolID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, entries, err := doDeleteLootPool(ctx, poolID)
	if err != nil {
		if err.Error() == loot.PoolNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteLootPool] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLootPoolDelete, audit.TargetLootPool, deleted.ID, map[string]interface{}{
		"code":        deleted.Code,
		"name":        deleted.Name,
		"entry_count": entries,
	})
	logger.InfoF(ctx, "[DeleteLootPool] %d deleted loot pool %d %s, %d entries", operator.ID, deleted.ID, deleted.Code, entries)
	c.JSON(http.StatusOK, Response{Data: DeleteLootPoolResponseData{EntryCount: entries}})
}

// LootPoolEntriesResponseData 掉落池及其全部条目
type LootPoolEntriesResponseData struct {
	Pool    loot.Pool          `json:"pool"`
	Entries []loot.EntryDetail `json:"entries"` // 按条目 ID 排序, 包含不参与抽取的条目
}

// ListLootPoolEntries
// @Tags admin
// @Summary 获取掉落池的全部条目
// @Description 包含已下架、已删除的物品和已停用的货币, 这些条目不参与抽取, 概率按其余条目计算
// @Produce json
// @Param id path int true "loot pool id"
// @Success 200 {object} Response{data=LootPoolEntriesResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/loot-pools/{id}/entries [get]
func ListLootPoolEntries(c *gin.Context) {
	ctx := c.Request.Context()
	poolID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	pool, entries, err := queryLootPoolEntries(ctx, poolID)
	if err != nil {
		if err.Error() == loot.PoolNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[ListLootPoolEntries] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: LootPoolEntriesResponseData{Pool: *pool, Entries: entries}})
}

// LootPoolEntryRequest 掉落池的一个条目
type LootPoolEntryRequest struct {
	ItemID      uint64 `json:"item_id"`                                                 // 掉落的物品, 与 currency 二选一
	Currency    string `json:"currency" binding:"max=32"`                               // 掉落的货币
	Weight      int32  `json:"weight" binding:"required,min=1,max=1000000"`             // 权重
	MinQuantity int64  `json:"min_quantity" binding:"required,min=1,max=1000000000000"` // 每次掉落的数量范围, 货币为金额, 物品最多 1000000
	MaxQuantity int64  `json:"max_quantity" binding:"required,min=1,max=1000000000000"`
}

// ReplaceLootPoolEntriesRequest 替换掉落池条目请求体
type ReplaceLootPoolEntriesRequest struct {
	Entries []LootPoolEntryRequest `json:"entries" binding:"required,max=200,dive"` // 提交空数组清空条目
}

// ReplaceLootPoolEntries
// @Tags admin
// @Summary 替换掉落池的全部条目
// @Description 用提交的条目整体替换; 物品需未删除, 货币需存在, 已下架的物品和已停用的货币也可以加入, 恢复后才参与抽取
// @Accept json
// @Produce json
// @Param id path int true "loot pool id"
// @Param request body ReplaceLootPoolEntriesRequest true "request body"
// @Success 200 {object} Response{data=LootPoolEntriesResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/loot-pools/{id}/entries [put]
func ReplaceLootPoolEntries(c *gin.Context) {
	ctx := c.Request.Context()
	var req ReplaceLootPoolEntriesRequest
	poolID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	pool, entries, changes, err := doReplaceLootPoolEntries(ctx, poolID, &req)
	if err != nil {
		switch err.Error() {
		case loot.EntryTargetInvalid, loot.EntryRangeInvalid, loot.EntryTooMany, inventory.QuantityInvalid, item.ItemNotFound, wallet.CurrencyNotFound:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case loot.PoolNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[ReplaceLootPoolEntries] replace failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLootPoolEntriesUpdate, audit.TargetLootPool, pool.ID, changes)
	logger.InfoF(ctx, "[ReplaceLootPoolEntries] %d replaced loot pool %d %s with %d entries", operator.ID, pool.ID, pool.Code, len(entries))
	c.JSON(http.StatusOK, Response{Data: LootPoolEntriesResponseData{Pool: *pool, Entries: entries}})
}

// ListVenuesRequest 地图列表查询参数
type ListVenuesRequest struct {
	Current   int    `json:"current" form:"current" binding:"min=1"`
	Size      int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"` // 按代码或名称搜索, 不区分大小写
	IsEnabled *bool  `json:"is_enabled" form:"is_enabled"`
}

// ListVenuesResponseData 地图列表分页结果
type ListVenuesResponseData struct {
	Total  int64         `json:"total"`
	Venues []venue.Venue `json:"venues"`
}

// ListVenues
// @Tags admin
// @Summary 获取地图列表
// @Description 按排序值和 ID 排列, 不含已删除的地图
// @Produce json
// @Param request query ListVenuesRequest true "request query"
// @Success 200 {object} Response{data=ListVenuesResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/venues [get]
func ListVenues(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListVenuesRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, venues, err := QueryVenues(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListVenues] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListVenuesResponseData{Total: total, Venues: venues}})
}

// VenueRequest 新建、修改或试算地图请求体
//
// 入场代价、入场条件和各项规则不在这里校验, 由 venue.Validate 校验, 提示更具体。
type VenueRequest struct {
	Code          string                `json:"code" binding:"required,max=24"` // 小写字母、数字和下划线, 创建后不能修改
	Name          string                `json:"name" binding:"required,max=64"`
	Description   string                `json:"description" binding:"max=255"`
	ImageURL      string                `json:"image_url" binding:"omitempty,max=1024,http_url"`
	SortOrder     int32                 `json:"sort_order" binding:"min=-1000000,max=1000000"` // 大厅中从小到大排列
	IsEnabled     bool                  `json:"is_enabled"`
	PveEnabled    bool                  `json:"pve_enabled"`
	RankedEnabled bool                  `json:"ranked_enabled"`
	BidCurrency   string                `json:"bid_currency" binding:"required,max=32"`
	EntryCost     reward.Bundle         `json:"entry_cost" binding:"-"` // 数量为正数, 开局时扣除
	Requirements  []condition.Condition `json:"requirements" binding:"-"`
	Warehouse     venue.WarehouseRule   `json:"warehouse" binding:"-"`
	Auction       venue.AuctionRule     `json:"auction" binding:"-"`
	Bots          venue.BotRule         `json:"bots" binding:"-"`
	Pve           venue.PveRule         `json:"pve" binding:"-"`
}

// CreateVenue
// @Tags admin
// @Summary 新建地图
// @Description 保存前模拟 10000 个仓库, 摆放失败率超过 1% 时拒绝; 模拟结果写入 simulation
// @Accept json
// @Produce json
// @Param request body VenueRequest true "request body"
// @Success 200 {object} Response{data=venue.Venue}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/venues [post]
func CreateVenue(c *gin.Context) {
	ctx := c.Request.Context()
	var req VenueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateVenue(ctx, &req)
	if err != nil {
		var invalid *venue.InvalidError
		switch {
		case errors.As(err, &invalid):
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case err.Error() == venue.CodeTaken:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[CreateVenue] create failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionVenueCreate, audit.TargetVenue, created.ID, venueFields(created))
	logger.InfoF(ctx, "[CreateVenue] %d created venue %d %s", operator.ID, created.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateVenue
// @Tags admin
// @Summary 修改地图
// @Description 需提交全部字段, 代码不能修改; 每次保存都会重新模拟仓库; 进行中的对局使用开局时的快照, 不受影响
// @Accept json
// @Produce json
// @Param id path int true "venue id"
// @Param request body VenueRequest true "request body"
// @Success 200 {object} Response{data=venue.Venue}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/venues/{id} [put]
func UpdateVenue(c *gin.Context) {
	ctx := c.Request.Context()
	var req VenueRequest
	venueID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateVenue(ctx, venueID, &req)
	if err != nil {
		var invalid *venue.InvalidError
		switch {
		case errors.As(err, &invalid), err.Error() == venue.CodeImmutable:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case err.Error() == venue.VenueNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[UpdateVenue] update failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionVenueUpdate, audit.TargetVenue, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateVenue] %d updated venue %d %s", operator.ID, updated.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteVenue
// @Tags admin
// @Summary 删除地图
// @Description 软删除, 代码不能再被新地图使用; 排队中的玩家开局时被拒绝, 进行中的对局照常打完
// @Produce json
// @Param id path int true "venue id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/venues/{id} [delete]
func DeleteVenue(c *gin.Context) {
	ctx := c.Request.Context()
	venueID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeleteVenue(ctx, venueID)
	if err != nil {
		if err.Error() == venue.VenueNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteVenue] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionVenueDelete, audit.TargetVenue, deleted.ID, map[string]interface{}{
		"code": deleted.Code,
		"name": deleted.Name,
	})
	logger.InfoF(ctx, "[DeleteVenue] %d deleted venue %d %s", operator.ID, deleted.ID, deleted.Code)
	c.JSON(http.StatusOK, Response{})
}

// SimulateVenue
// @Tags admin
// @Summary 试算地图
// @Description 按保存时的规则校验并模拟 10000 个仓库, 返回价值分布, 不保存
// @Accept json
// @Produce json
// @Param request body VenueRequest true "request body"
// @Success 200 {object} Response{data=venue.Simulation}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/venues/simulate [post]
func SimulateVenue(c *gin.Context) {
	ctx := c.Request.Context()
	var req VenueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	simulation, err := doSimulateVenue(ctx, &req)
	if err != nil {
		var invalid *venue.InvalidError
		if errors.As(err, &invalid) {
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[SimulateVenue] simulate failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: simulation})
}

// ListConditionTypesResponseData 全部条件类型
type ListConditionTypesResponseData struct {
	Types []condition.TypeView `json:"types"`
}

// ListConditionTypes
// @Tags admin
// @Summary 获取全部条件类型
// @Description 包含每种条件的参数表单描述, 用于条件编辑器; input 为 currency、item、scope 的字段需要用选择器填写
// @Produce json
// @Success 200 {object} Response{data=ListConditionTypesResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Router /api/v1/admin/conditions/types [get]
func ListConditionTypes(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Data: ListConditionTypesResponseData{Types: condition.Types()}})
}

// ListMatchesRequest 对局列表查询参数
type ListMatchesRequest struct {
	Current int     `json:"current" form:"current" binding:"min=1"`
	Size    int     `json:"size" form:"size" binding:"min=1,max=100"`
	Mode    string  `json:"mode" form:"mode" binding:"max=32"`
	Status  string  `json:"status" form:"status" binding:"omitempty,oneof=playing finished cancelled"`
	UserID  *uint64 `json:"user_id" form:"user_id"` // 只看该用户参与的对局
}

// ListMatchesResponseData 对局列表分页结果
type ListMatchesResponseData struct {
	Total   int64               `json:"total"`
	Matches []match.MatchDetail `json:"matches"`
}

// ListMatches
// @Tags admin
// @Summary 获取对局列表
// @Description 包含每个座位是真人还是人机, 按开始时间从新到旧
// @Produce json
// @Param request query ListMatchesRequest true "request query"
// @Success 200 {object} Response{data=ListMatchesResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/matches [get]
func ListMatches(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListMatchesRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, matches, err := QueryMatches(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListMatches] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListMatchesResponseData{Total: total, Matches: matches}})
}

// CancelMatch
// @Tags admin
// @Summary 取消进行中的对局
// @Description 用于对局卡住等异常情况; 玩法会退还入场费等, 玩法已下线时需要手动处理
// @Produce json
// @Param id path int true "match id"
// @Success 200 {object} Response{data=match.Match}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/matches/{id}/cancel [post]
func CancelMatch(c *gin.Context) {
	ctx := c.Request.Context()
	matchID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	cancelled, err := match.Cancel(ctx, matchID)
	if err != nil {
		switch err.Error() {
		case match.MatchNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case match.MatchNotPlaying:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[CancelMatch] cancel failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionMatchCancel, audit.TargetMatch, cancelled.ID, map[string]interface{}{
		"mode":  cancelled.Mode,
		"phase": cancelled.Phase,
		"round": cancelled.Round,
	})
	logger.InfoF(ctx, "[CancelMatch] %d cancelled match %d %s", operator.ID, cancelled.ID, cancelled.Mode)
	c.JSON(http.StatusOK, Response{Data: cancelled})
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
	// 重复提交只返回原流水, 不重复记录和推送
	if !replayed {
		inventory.PushChanges(ctx, txn)
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

// ListSettingsRequest 获取设置组请求参数
type ListSettingsRequest struct {
	Category string `json:"category" form:"category" binding:"omitempty,oneof=system game"` // 为空时返回全部
}

// ListSettingsResponseData 设置组和分类
type ListSettingsResponseData struct {
	Categories []settings.Category  `json:"categories"`
	Groups     []settings.GroupView `json:"groups"`
}

// ListSettings
// @Tags admin
// @Summary 获取设置组
// @Description 包含表单描述和当前值, 密钥字段不返回明文; 可以按分类筛选
// @Produce json
// @Param request query ListSettingsRequest true "request query"
// @Success 200 {object} Response{data=ListSettingsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings [get]
func ListSettings(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListSettingsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	groups, err := settings.QueryGroups(ctx, req.Category)
	if err != nil {
		logger.ErrorF(ctx, "[ListSettings] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListSettingsResponseData{Categories: settings.Categories, Groups: groups}})
}

// GetSetting
// @Tags admin
// @Summary 获取一个设置组
// @Produce json
// @Param key path string true "setting group key"
// @Success 200 {object} Response{data=settings.GroupView}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings/{key} [get]
func GetSetting(c *gin.Context) {
	ctx := c.Request.Context()
	group, err := settings.QueryGroup(ctx, c.Param("key"))
	if err != nil {
		if err.Error() == settings.GroupNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[GetSetting] query %s failed: %v", c.Param("key"), err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: group})
}

// UpdateSettingRequest 保存设置组请求体
type UpdateSettingRequest struct {
	Version      *int64          `json:"version" binding:"required,min=0"`              // 读取时的版本, 已被别人修改时返回 409
	Value        json.RawMessage `json:"value" binding:"required" swaggertype:"object"` // 设置组的字段, 只需提交改动的字段, 没有提交的保持不变, 密钥字段留空表示不修改
	ClearSecrets []string        `json:"clear_secrets" binding:"max=32"`                // 需要清空的密钥字段
}

// UpdateSetting
// @Tags admin
// @Summary 保存一个设置组
// @Description 保存后立即生效, 并生成一个新版本
// @Accept json
// @Produce json
// @Param key path string true "setting group key"
// @Param request body UpdateSettingRequest true "request body"
// @Success 200 {object} Response{data=settings.GroupView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
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
	group, changes, err := settings.Update(ctx, operator.ID, c.Param("key"), *req.Version, req.Value, req.ClearSecrets)
	if err != nil {
		c.JSON(settingErrorStatus(err), Response{ErrorMsg: err.Error()})
		return
	}
	recordSettingChange(c, operator.ID, audit.ActionSettingsUpdate, group, changes, nil)
	c.JSON(http.StatusOK, Response{Data: group})
}

// ListSettingRevisionsRequest 获取设置组版本请求参数
type ListSettingRevisionsRequest struct {
	Current int `json:"current" form:"current" binding:"min=1"`
	Size    int `json:"size" form:"size" binding:"min=1,max=100"`
}

// ListSettingRevisionsResponseData 设置组版本分页结果
type ListSettingRevisionsResponseData struct {
	Total     int64                 `json:"total"`
	Revisions []SettingRevisionItem `json:"revisions"`
}

// ListSettingRevisions
// @Tags admin
// @Summary 获取设置组的历史版本
// @Description 按版本从新到旧排列, 不含快照内容
// @Produce json
// @Param key path string true "setting group key"
// @Param request query ListSettingRevisionsRequest true "request query"
// @Success 200 {object} Response{data=ListSettingRevisionsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings/{key}/revisions [get]
func ListSettingRevisions(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListSettingRevisionsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	key := c.Param("key")
	if !settings.HasGroup(key) {
		c.JSON(http.StatusNotFound, Response{ErrorMsg: settings.GroupNotFound})
		return
	}
	total, revisions, err := QuerySettingRevisions(ctx, key, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListSettingRevisions] query %s failed: %v", key, err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListSettingRevisionsResponseData{Total: total, Revisions: revisions}})
}

// GetSettingRevision
// @Tags admin
// @Summary 获取设置组某个版本的快照
// @Description 密钥字段不返回明文, 只返回是否已设置
// @Produce json
// @Param key path string true "setting group key"
// @Param version path int true "version"
// @Success 200 {object} Response{data=settings.RevisionView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings/{key}/revisions/{version} [get]
func GetSettingRevision(c *gin.Context) {
	ctx := c.Request.Context()
	version, err := strconv.ParseInt(c.Param("version"), 10, 64)
	if err != nil || version < 0 {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	revision, err := settings.QueryRevision(ctx, c.Param("key"), version)
	if err != nil {
		switch err.Error() {
		case settings.GroupNotFound, settings.RevisionNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[GetSettingRevision] query %s@%d failed: %v", c.Param("key"), version, err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	c.JSON(http.StatusOK, Response{Data: revision})
}

// RollbackSettingRequest 回滚设置组请求体
type RollbackSettingRequest struct {
	Version *int64 `json:"version" binding:"required,min=0"` // 当前版本, 已被别人修改时返回 409
	Target  *int64 `json:"target" binding:"required,min=0"`  // 回滚到的版本
}

// RollbackSetting
// @Tags admin
// @Summary 回滚设置组
// @Description 用旧版本的快照生成一个新版本, 立即生效; 快照中的密钥无法解密时不能回滚
// @Accept json
// @Produce json
// @Param key path string true "setting group key"
// @Param request body RollbackSettingRequest true "request body"
// @Success 200 {object} Response{data=settings.GroupView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings/{key}/rollback [post]
func RollbackSetting(c *gin.Context) {
	ctx := c.Request.Context()
	var req RollbackSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	group, changes, err := settings.Rollback(ctx, operator.ID, c.Param("key"), *req.Version, *req.Target)
	if err != nil {
		c.JSON(settingErrorStatus(err), Response{ErrorMsg: err.Error()})
		return
	}
	recordSettingChange(c, operator.ID, audit.ActionSettingsRollback, group, changes, map[string]interface{}{"target": *req.Target})
	c.JSON(http.StatusOK, Response{Data: group})
}

// ResetSettingRequest 恢复默认值请求体
type ResetSettingRequest struct {
	Version *int64 `json:"version" binding:"required,min=0"` // 当前版本, 已被别人修改时返回 409
}

// ResetSetting
// @Tags admin
// @Summary 恢复设置组的默认值
// @Description 恢复为代码中的默认值并生成一个新版本, 密钥字段会被清空
// @Accept json
// @Produce json
// @Param key path string true "setting group key"
// @Param request body ResetSettingRequest true "request body"
// @Success 200 {object} Response{data=settings.GroupView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings/{key}/reset [post]
func ResetSetting(c *gin.Context) {
	ctx := c.Request.Context()
	var req ResetSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	group, changes, err := settings.Reset(ctx, operator.ID, c.Param("key"), *req.Version)
	if err != nil {
		c.JSON(settingErrorStatus(err), Response{ErrorMsg: err.Error()})
		return
	}
	recordSettingChange(c, operator.ID, audit.ActionSettingsReset, group, changes, nil)
	c.JSON(http.StatusOK, Response{Data: group})
}

// settingErrorStatus 保存设置组的错误对应的状态码, 错误提示都可以直接展示
func settingErrorStatus(err error) int {
	switch err.Error() {
	case settings.GroupNotFound, settings.RevisionNotFound:
		return http.StatusNotFound
	case settings.VersionConflict:
		return http.StatusConflict
	case settings.SaveFailed:
		return http.StatusInternalServerError
	default:
		// 其余均为设置内容校验失败
		return http.StatusBadRequest
	}
}

// recordSettingChange 记录设置组的修改, 没有任何修改时不记录
func recordSettingChange(c *gin.Context, operatorID uint64, action string, group *settings.GroupView, changes, extra map[string]interface{}) {
	if len(changes) == 0 {
		return
	}
	detail := map[string]interface{}{
		"key":     group.Key,
		"version": group.Version,
		"changes": changes,
	}
	for k, v := range extra {
		detail[k] = v
	}
	audit.Record(c, operatorID, audit.ScopeAdmin, action, audit.TargetSetting, 0, detail)
	logger.InfoF(c.Request.Context(), "[%s] %d changed %s to version %d", action, operatorID, group.Key, group.Version)
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
// @Failure 429 {object} Response
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
