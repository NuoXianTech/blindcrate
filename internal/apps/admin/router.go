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
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auction"
	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/effect"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/apps/ranked"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/shop"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
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

// currencyErrorStatus 货币配置错误对应的状态码, 不是配置错误时返回 0
func currencyErrorStatus(err error) int {
	switch err.Error() {
	case wallet.CurrencyCodeInvalid, wallet.CurrencyNameInvalid:
		return http.StatusBadRequest
	case wallet.CurrencyNotFound:
		return http.StatusNotFound
	case wallet.CurrencyTaken, wallet.CurrencyInUse, wallet.CurrencyIsBase:
		return http.StatusConflict
	}
	return 0
}

// CreateCurrency
// @Tags admin
// @Summary 新建货币
// @Accept json
// @Produce json
// @Param request body CreateCurrencyRequest true "request body"
// @Success 200 {object} Response{data=wallet.Currency}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/wallet/currencies [post]
func CreateCurrency(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreateCurrencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateCurrency(ctx, &req)
	if err != nil {
		if status := currencyErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateCurrency] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionCurrencyCreate, audit.TargetCurrency, 0, currencyFields(created))
	logger.InfoF(ctx, "[CreateCurrency] %d created currency %s", operator.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateCurrency
// @Tags admin
// @Summary 修改货币
// @Description 需提交全部字段, 代码不能修改; 已被余额、物品、地图或掉落池使用的货币不能修改小数位, 基础货币不能停用
// @Accept json
// @Produce json
// @Param code path string true "currency code"
// @Param request body CurrencyRequest true "request body"
// @Success 200 {object} Response{data=wallet.Currency}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/wallet/currencies/{code} [put]
func UpdateCurrency(c *gin.Context) {
	ctx := c.Request.Context()
	var req CurrencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateCurrency(ctx, c.Param("code"), &req)
	if err != nil {
		if status := currencyErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateCurrency] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionCurrencyUpdate, audit.TargetCurrency, 0, map[string]interface{}{
			"code":    updated.Code,
			"changes": changes,
		})
		logger.InfoF(ctx, "[UpdateCurrency] %d updated currency %s", operator.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
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
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"`          // 按代码或名称搜索, 不区分大小写
	Quality   *int16 `json:"quality" form:"quality" binding:"omitempty,min=1"` // 品质等级
	Category  string `json:"category" form:"category" binding:"max=32"`        // 类别代码
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
	Code         string         `json:"code" binding:"required,max=64"` // 英文别名, 小写字母、数字和下划线, 可以重复
	Name         string         `json:"name" binding:"required,max=64"`
	Description  string         `json:"description" binding:"max=255"`
	Quality      int16          `json:"quality" binding:"required,min=1"`   // 品质等级, 必须存在
	Category     string         `json:"category" binding:"required,max=32"` // 类别代码, 必须已注册
	ImageURL     string         `json:"image_url" binding:"omitempty,max=1024,image_url"`
	SellCurrency string         `json:"sell_currency" binding:"required,max=32"`
	SellPrice    int64          `json:"sell_price" binding:"min=0,max=1000000000000"` // 0 表示不可卖出
	Width        int16          `json:"width" binding:"required,min=1,max=10"`        // 在仓库中占的格数
	Height       int16          `json:"height" binding:"required,min=1,max=10"`
	Effect       string         `json:"effect,omitempty" binding:"max=32"`            // 使用效果代码, 空表示不能使用; 藏品不能配置
	EffectParams map[string]any `json:"effect_params,omitempty" swaggertype:"object"` // 效果参数, 结构见 GET /admin/items/effects
	IsEnabled    bool           `json:"is_enabled"`                                   // 下架后不会掉落, 也不能再发放, 已有的保留
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
		var invalid *effect.InvalidError
		if errors.As(err, &invalid) {
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
			return
		}
		switch err.Error() {
		case item.ItemCodeInvalid, item.ItemNameInvalid, item.QualityNotFound, item.CategoryNotFound, wallet.CurrencyNotFound:
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
		var invalid *effect.InvalidError
		if errors.As(err, &invalid) {
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
			return
		}
		switch err.Error() {
		case item.ItemCodeInvalid, item.ItemNameInvalid, item.QualityNotFound, item.CategoryNotFound, wallet.CurrencyNotFound:
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

// ListItemCategoriesResponseData 物品类别列表
type ListItemCategoriesResponseData struct {
	Categories []item.Category `json:"categories"`
}

// ListItemCategories
// @Tags admin
// @Summary 获取物品类别
// @Description 由代码注册的全部物品类别, 按注册顺序
// @Produce json
// @Success 200 {object} Response{data=ListItemCategoriesResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Router /api/v1/admin/items/categories [get]
func ListItemCategories(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Data: ListItemCategoriesResponseData{Categories: item.Categories()}})
}

// ListItemEffectsResponseData 物品的使用效果列表
type ListItemEffectsResponseData struct {
	Effects []effect.TypeView `json:"effects"`
}

// ListItemEffects
// @Tags admin
// @Summary 获取物品的使用效果
// @Description 由代码注册的全部效果, 按注册顺序; fields 为参数的表单描述, 与条件类型相同
// @Produce json
// @Success 200 {object} Response{data=ListItemEffectsResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Router /api/v1/admin/items/effects [get]
func ListItemEffects(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Data: ListItemEffectsResponseData{Effects: effect.Types()}})
}

// ListQualitiesResponseData 品质列表
type ListQualitiesResponseData struct {
	Qualities []QualityListItem `json:"qualities"`
}

// ListQualities
// @Tags admin
// @Summary 获取品质列表
// @Description 按排序值排列, 带上使用每个品质的物品数
// @Produce json
// @Success 200 {object} Response{data=ListQualitiesResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/qualities [get]
func ListQualities(c *gin.Context) {
	ctx := c.Request.Context()
	qualities, err := queryQualities(ctx)
	if err != nil {
		logger.ErrorF(ctx, "[ListQualities] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListQualitiesResponseData{Qualities: qualities}})
}

// QualityRequest 修改品质请求体, 等级和代码创建后不能修改
type QualityRequest struct {
	Name      string `json:"name" binding:"required,max=32"`
	Color     string `json:"color" binding:"required,max=7"`                // #rrggbb
	Effect    string `json:"effect" binding:"max=16"`                       // 特效, 空表示纯色, rainbow 为流动渐变
	ScanMs    int32  `json:"scan_ms" binding:"min=100,max=10000"`           // 开箱时扫描这一品质藏品的时长, 毫秒; 品质越高可以扫得越久
	SortOrder int32  `json:"sort_order" binding:"min=-1000000,max=1000000"` // 筛选和图例中从小到大排列
}

// CreateQualityRequest 新建品质请求体
type CreateQualityRequest struct {
	Level int16  `json:"level" binding:"required,min=1,max=999"` // 等级, 越大越稀有
	Code  string `json:"code" binding:"required,max=32"`         // 自然键, 小写字母、数字和下划线
	QualityRequest
}

// qualityErrorStatus 品质配置错误对应的状态码, 不是配置错误时返回 0
func qualityErrorStatus(err error) int {
	switch err.Error() {
	case item.QualityCodeInvalid, item.QualityNameInvalid, item.QualityColorInvalid, item.QualityEffectInvalid:
		return http.StatusBadRequest
	case item.QualityNotFound:
		return http.StatusNotFound
	case item.QualityTaken, item.QualityInUse:
		return http.StatusConflict
	}
	return 0
}

// CreateQuality
// @Tags admin
// @Summary 新建品质
// @Accept json
// @Produce json
// @Param request body CreateQualityRequest true "request body"
// @Success 200 {object} Response{data=item.Quality}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/qualities [post]
func CreateQuality(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreateQualityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateQuality(ctx, &req)
	if err != nil {
		if status := qualityErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateQuality] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionQualityCreate, audit.TargetQuality, uint64(created.Level), qualityFields(created))
	logger.InfoF(ctx, "[CreateQuality] %d created quality %d %s", operator.ID, created.Level, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateQuality
// @Tags admin
// @Summary 修改品质
// @Description 需提交全部字段, 等级和代码不能修改; 新的扫描时长只影响之后开局的对局
// @Accept json
// @Produce json
// @Param level path int true "quality level"
// @Param request body QualityRequest true "request body"
// @Success 200 {object} Response{data=item.Quality}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/qualities/{level} [put]
func UpdateQuality(c *gin.Context) {
	ctx := c.Request.Context()
	var req QualityRequest
	level, err := strconv.ParseInt(c.Param("level"), 10, 16)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateQuality(ctx, int16(level), &req)
	if err != nil {
		if status := qualityErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateQuality] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionQualityUpdate, audit.TargetQuality, uint64(updated.Level), changes)
		logger.InfoF(ctx, "[UpdateQuality] %d updated quality %d %s", operator.ID, updated.Level, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteQuality
// @Tags admin
// @Summary 删除品质
// @Description 还有未删除的物品使用该品质时不能删除; 已开局对局中的藏品保留原等级
// @Produce json
// @Param level path int true "quality level"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/qualities/{level} [delete]
func DeleteQuality(c *gin.Context) {
	ctx := c.Request.Context()
	level, err := strconv.ParseInt(c.Param("level"), 10, 16)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeleteQuality(ctx, int16(level))
	if err != nil {
		if status := qualityErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteQuality] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionQualityDelete, audit.TargetQuality, uint64(deleted.Level), qualityFields(deleted))
	logger.InfoF(ctx, "[DeleteQuality] %d deleted quality %d %s", operator.ID, deleted.Level, deleted.Code)
	c.JSON(http.StatusOK, Response{})
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
	ImageURL      string                `json:"image_url" binding:"omitempty,max=1024,image_url"`
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
	Consumables   venue.ConsumableRule  `json:"consumables" binding:"-"` // 对局中使用道具的规则
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
	Prefix  string  `json:"prefix" form:"prefix" binding:"max=32"` // 玩法代码冒号前的部分, 如 pve
	Suffix  string  `json:"suffix" form:"suffix" binding:"max=32"` // 玩法代码冒号后的部分, 如地图代码
	Status  string  `json:"status" form:"status" binding:"omitempty,oneof=playing finished cancelled"`
	UserID  *uint64 `json:"user_id" form:"user_id"`   // 只看该用户参与的对局
	OtherID *uint64 `json:"other_id" form:"other_id"` // 只看该用户也参与的对局, 与 user_id 一起用于查看两个人一起打的对局
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

// GetMatchAuction
// @Tags admin
// @Summary 获取一局暗拍的详情
// @Description 与玩家看到的公开部分相同, 附带每个座位的带入资金、出价、估值和盈亏
// @Produce json
// @Param id path int true "match id"
// @Success 200 {object} Response{data=auction.AdminView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/matches/{id}/auction [get]
func GetMatchAuction(c *gin.Context) {
	ctx := c.Request.Context()
	matchID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	view, err := auction.QueryAdminAuction(ctx, matchID)
	if err != nil {
		if err.Error() == match.MatchNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[GetMatchAuction] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: view})
}

// CancelMatchRequest 取消对局请求体
type CancelMatchRequest struct {
	Reason string `json:"reason" binding:"required,max=255"` // 记录在操作日志中
}

// CancelMatch
// @Tags admin
// @Summary 取消进行中的对局
// @Description 用于对局卡住等异常情况; 玩法会退还入场费等, 玩法已下线时需要手动处理
// @Accept json
// @Produce json
// @Param id path int true "match id"
// @Param request body CancelMatchRequest true "request body"
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
	var req CancelMatchRequest
	matchID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.Reason) == "" {
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
		"mode":   cancelled.Mode,
		"phase":  cancelled.Phase,
		"round":  cancelled.Round,
		"reason": strings.TrimSpace(req.Reason),
	})
	logger.InfoF(ctx, "[CancelMatch] %d cancelled match %d %s", operator.ID, cancelled.ID, cancelled.Mode)
	c.JSON(http.StatusOK, Response{Data: cancelled})
}

// ListLeaderboardsRequest 榜单列表查询参数
type ListLeaderboardsRequest struct {
	Current   int    `json:"current" form:"current" binding:"min=1"`
	Size      int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"` // 按代码或名称搜索, 不区分大小写
	IsEnabled *bool  `json:"is_enabled" form:"is_enabled"`
}

// ListLeaderboardsResponseData 榜单列表分页结果
type ListLeaderboardsResponseData struct {
	Total  int64               `json:"total"`
	Boards []leaderboard.Board `json:"boards"`
}

// ListLeaderboards
// @Tags admin
// @Summary 获取榜单列表
// @Description 按排序值排列, 包含停用的榜单
// @Produce json
// @Param request query ListLeaderboardsRequest true "request query"
// @Success 200 {object} Response{data=ListLeaderboardsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/leaderboards [get]
func ListLeaderboards(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListLeaderboardsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, boards, err := QueryLeaderboards(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListLeaderboards] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListLeaderboardsResponseData{Total: total, Boards: boards}})
}

// ListLeaderboardMetricsResponseData 可以排名的指标
type ListLeaderboardMetricsResponseData struct {
	Metrics []stats.Metric `json:"metrics"`
}

// ListLeaderboardMetrics
// @Tags admin
// @Summary 获取可以排名的指标
// @Description 已注册的计数指标和每种货币的净收益
// @Produce json
// @Success 200 {object} Response{data=ListLeaderboardMetricsResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/leaderboards/metrics [get]
func ListLeaderboardMetrics(c *gin.Context) {
	ctx := c.Request.Context()
	metrics, err := leaderboard.QueryMetrics(ctx)
	if err != nil {
		logger.ErrorF(ctx, "[ListLeaderboardMetrics] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListLeaderboardMetricsResponseData{Metrics: metrics}})
}

// LeaderboardRequest 新建或修改榜单请求体
type LeaderboardRequest struct {
	Code        string `json:"code" binding:"required,max=64"` // 自然键, 小写字母、数字和下划线
	Name        string `json:"name" binding:"required,max=64"`
	Description string `json:"description" binding:"max=255"`
	Metric      string `json:"metric" binding:"required,max=64"`              // 排名的指标, 见可以排名的指标
	Scope       string `json:"scope" binding:"required,max=64"`               // 统计范围, 如 all、mode:ranked、venue:dock
	Period      string `json:"period" binding:"required"`                     // total 累计、day 当天或 season 当前赛季
	Size        int32  `json:"size"`                                          // 展示前多少名, 10-500
	MinMatches  int64  `json:"min_matches"`                                   // 同一范围和周期内至少完成多少局才上榜
	SortOrder   int32  `json:"sort_order" binding:"min=-1000000,max=1000000"` // 玩家页面的页签从小到大排列
	IsEnabled   bool   `json:"is_enabled"`                                    // 停用后玩家看不到
}

// leaderboardErrorStatus 榜单配置错误对应的状态码, 不是配置错误时返回 0
func leaderboardErrorStatus(err error) int {
	switch err.Error() {
	case leaderboard.CodeInvalid, leaderboard.NameInvalid, leaderboard.MetricInvalid, leaderboard.ScopeInvalid,
		leaderboard.PeriodInvalid, leaderboard.SizeInvalid, leaderboard.MinMatchesInvalid, leaderboard.RatingBoardInvalid:
		return http.StatusBadRequest
	case leaderboard.BoardNotFound:
		return http.StatusNotFound
	case leaderboard.CodeTaken:
		return http.StatusConflict
	}
	return 0
}

// CreateLeaderboard
// @Tags admin
// @Summary 新建榜单
// @Accept json
// @Produce json
// @Param request body LeaderboardRequest true "request body"
// @Success 200 {object} Response{data=leaderboard.Board}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/leaderboards [post]
func CreateLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()
	var req LeaderboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateLeaderboard(ctx, &req)
	if err != nil {
		if status := leaderboardErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateLeaderboard] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLeaderboardCreate, audit.TargetLeaderboard, created.ID, leaderboardFields(created))
	logger.InfoF(ctx, "[CreateLeaderboard] %d created leaderboard %d %s", operator.ID, created.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateLeaderboard
// @Tags admin
// @Summary 修改榜单
// @Description 需提交全部字段; 修改后缓存的前 N 名随即失效
// @Accept json
// @Produce json
// @Param id path int true "leaderboard id"
// @Param request body LeaderboardRequest true "request body"
// @Success 200 {object} Response{data=leaderboard.Board}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/leaderboards/{id} [put]
func UpdateLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()
	var req LeaderboardRequest
	boardID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateLeaderboard(ctx, boardID, &req)
	if err != nil {
		if status := leaderboardErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateLeaderboard] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLeaderboardUpdate, audit.TargetLeaderboard, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateLeaderboard] %d updated leaderboard %d %s", operator.ID, updated.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteLeaderboard
// @Tags admin
// @Summary 删除榜单
// @Description 只删除榜单配置, 统计数据不受影响
// @Produce json
// @Param id path int true "leaderboard id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/leaderboards/{id} [delete]
func DeleteLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()
	boardID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeleteLeaderboard(ctx, boardID)
	if err != nil {
		if err.Error() == leaderboard.BoardNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteLeaderboard] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLeaderboardDelete, audit.TargetLeaderboard, deleted.ID, leaderboardFields(deleted))
	logger.InfoF(ctx, "[DeleteLeaderboard] %d deleted leaderboard %d %s", operator.ID, deleted.ID, deleted.Code)
	c.JSON(http.StatusOK, Response{})
}

// ListSeasonsRequest 赛季列表查询参数
type ListSeasonsRequest struct {
	Current int    `json:"current" form:"current" binding:"min=1"`
	Size    int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword string `json:"keyword" form:"keyword" binding:"max=64"`                                      // 按代码或名称搜索, 不区分大小写
	Status  string `json:"status" form:"status" binding:"omitempty,oneof=upcoming active ended settled"` // 按状态筛选, 为空时不筛选
}

// ListSeasonsResponseData 赛季列表分页结果
type ListSeasonsResponseData struct {
	Total   int64         `json:"total"`
	Seasons []season.View `json:"seasons"`
}

// ListSeasons
// @Tags admin
// @Summary 获取赛季列表
// @Description 按开始时间从晚到早排列
// @Produce json
// @Param request query ListSeasonsRequest true "request query"
// @Success 200 {object} Response{data=ListSeasonsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/seasons [get]
func ListSeasons(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListSeasonsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, seasons, err := QuerySeasons(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListSeasons] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListSeasonsResponseData{Total: total, Seasons: seasons}})
}

// SeasonRequest 修改赛季请求体, 代码创建后不能修改; 时间精确到分钟, 秒会被舍去
type SeasonRequest struct {
	Name        string          `json:"name" binding:"required,max=64"`
	StartsAt    time.Time       `json:"starts_at" binding:"required"`         // RFC3339, 包含; 赛季开始后不能修改
	EndsAt      time.Time       `json:"ends_at" binding:"required"`           // RFC3339, 不包含; 需晚于开始时间和当前时间, 赛季结束后不能修改
	RatingCarry int16           `json:"rating_carry" binding:"min=0,max=100"` // 段位分继承上一个赛季的百分比; 赛季开始后不能修改
	Rewards     []season.Reward `json:"rewards" binding:"-"`                  // 赛季奖励, 分数线不能重复, 每档至少一项; 赛季结束后不能修改
}

// CreateSeasonRequest 新建赛季请求体
type CreateSeasonRequest struct {
	Code string `json:"code" binding:"required,max=24"` // 自然键, 小写字母、数字和下划线
	SeasonRequest
}

// seasonErrorStatus 赛季配置错误对应的状态码, 不是配置错误时返回 0
func seasonErrorStatus(err error) int {
	switch err.Error() {
	case season.CodeInvalid, season.NameInvalid, season.TimeInvalid, season.CarryInvalid,
		season.RewardsTooMany, season.RewardRatingInvalid, season.RewardRatingTaken, season.RewardEmpty,
		season.RewardLinesTooMany, season.RewardAmountInvalid, season.RewardNotFound:
		return http.StatusBadRequest
	case season.SeasonNotFound:
		return http.StatusNotFound
	case season.CodeTaken, season.TimeOverlapped, season.StartLocked, season.SeasonEnded, season.DeleteStarted, season.EndNotActive:
		return http.StatusConflict
	}
	return 0
}

// CreateSeason
// @Tags admin
// @Summary 新建赛季
// @Description 时间不能与其他赛季重叠
// @Accept json
// @Produce json
// @Param request body CreateSeasonRequest true "request body"
// @Success 200 {object} Response{data=season.View}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/seasons [post]
func CreateSeason(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreateSeasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateSeason(ctx, &req)
	if err != nil {
		if status := seasonErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateSeason] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionSeasonCreate, audit.TargetSeason, created.ID, seasonFields(&created.Season))
	logger.InfoF(ctx, "[CreateSeason] %d created season %d %s", operator.ID, created.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateSeason
// @Tags admin
// @Summary 修改赛季
// @Description 需提交全部字段; 赛季开始后不能修改开始时间, 结束后只能修改名称, 时间不能与其他赛季重叠
// @Accept json
// @Produce json
// @Param id path int true "season id"
// @Param request body SeasonRequest true "request body"
// @Success 200 {object} Response{data=season.View}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/seasons/{id} [put]
func UpdateSeason(c *gin.Context) {
	ctx := c.Request.Context()
	var req SeasonRequest
	seasonID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateSeason(ctx, seasonID, &req)
	if err != nil {
		if status := seasonErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateSeason] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionSeasonUpdate, audit.TargetSeason, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateSeason] %d updated season %d %s", operator.ID, updated.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteSeason
// @Tags admin
// @Summary 删除赛季
// @Description 只能删除未开始的赛季
// @Produce json
// @Param id path int true "season id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/seasons/{id} [delete]
func DeleteSeason(c *gin.Context) {
	ctx := c.Request.Context()
	seasonID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeleteSeason(ctx, seasonID)
	if err != nil {
		if status := seasonErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteSeason] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionSeasonDelete, audit.TargetSeason, deleted.ID, seasonFields(deleted))
	logger.InfoF(ctx, "[DeleteSeason] %d deleted season %d %s", operator.ID, deleted.ID, deleted.Code)
	c.JSON(http.StatusOK, Response{})
}

// EndSeason
// @Tags admin
// @Summary 提前结束赛季
// @Description 把进行中的赛季的结束时间改为当前时间, 之后的对局不再计入该赛季
// @Produce json
// @Param id path int true "season id"
// @Success 200 {object} Response{data=season.View}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/seasons/{id}/end [post]
func EndSeason(c *gin.Context) {
	ctx := c.Request.Context()
	seasonID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	ended, changes, err := doEndSeason(ctx, seasonID)
	if err != nil {
		if status := seasonErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[EndSeason] end failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionSeasonEnd, audit.TargetSeason, ended.ID, changes)
	logger.InfoF(ctx, "[EndSeason] %d ended season %d %s", operator.ID, ended.ID, ended.Code)
	c.JSON(http.StatusOK, Response{Data: ended})
}

// ListRankedTiersResponseData 段位列表
type ListRankedTiersResponseData struct {
	Tiers []ranked.Tier `json:"tiers"` // 按分数线从低到高
}

// ListRankedTiers
// @Tags admin
// @Summary 获取段位列表
// @Description 全部段位, 按分数线从低到高
// @Produce json
// @Success 200 {object} Response{data=ListRankedTiersResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/ranked/tiers [get]
func ListRankedTiers(c *gin.Context) {
	ctx := c.Request.Context()
	tiers, err := queryRankedTiers(ctx)
	if err != nil {
		logger.ErrorF(ctx, "[ListRankedTiers] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListRankedTiersResponseData{Tiers: tiers}})
}

// RankedTierRequest 新建或修改段位请求体
type RankedTierRequest struct {
	Name      string `json:"name" binding:"required,max=32"`
	MinRating int32  `json:"min_rating"`                     // 分数线, 段位分不低于它时属于这个段位, 0-1000000
	Color     string `json:"color" binding:"required,max=7"` // #rrggbb
}

// rankedTierErrorStatus 段位配置错误对应的状态码, 不是配置错误时返回 0
func rankedTierErrorStatus(err error) int {
	switch err.Error() {
	case ranked.TierNameInvalid, ranked.TierRatingInvalid, ranked.TierColorInvalid:
		return http.StatusBadRequest
	case ranked.TierNotFound:
		return http.StatusNotFound
	case ranked.TierRatingTaken:
		return http.StatusConflict
	}
	return 0
}

// CreateRankedTier
// @Tags admin
// @Summary 新建段位
// @Accept json
// @Produce json
// @Param request body RankedTierRequest true "request body"
// @Success 200 {object} Response{data=ranked.Tier}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/ranked/tiers [post]
func CreateRankedTier(c *gin.Context) {
	ctx := c.Request.Context()
	var req RankedTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateRankedTier(ctx, &req)
	if err != nil {
		if status := rankedTierErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateRankedTier] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionRankedTierCreate, audit.TargetRankedTier, created.ID, rankedTierFields(created))
	logger.InfoF(ctx, "[CreateRankedTier] %d created ranked tier %d %s", operator.ID, created.ID, created.Name)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateRankedTier
// @Tags admin
// @Summary 修改段位
// @Description 需提交全部字段; 段位按段位分实时计算, 修改分数线后所有人的段位随即变化
// @Accept json
// @Produce json
// @Param id path int true "ranked tier id"
// @Param request body RankedTierRequest true "request body"
// @Success 200 {object} Response{data=ranked.Tier}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/ranked/tiers/{id} [put]
func UpdateRankedTier(c *gin.Context) {
	ctx := c.Request.Context()
	var req RankedTierRequest
	tierID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateRankedTier(ctx, tierID, &req)
	if err != nil {
		if status := rankedTierErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateRankedTier] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionRankedTierUpdate, audit.TargetRankedTier, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateRankedTier] %d updated ranked tier %d %s", operator.ID, updated.ID, updated.Name)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteRankedTier
// @Tags admin
// @Summary 删除段位
// @Description 删除后原来在这个段位的人按剩下的分数线重新计算段位, 段位分不受影响
// @Produce json
// @Param id path int true "ranked tier id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/ranked/tiers/{id} [delete]
func DeleteRankedTier(c *gin.Context) {
	ctx := c.Request.Context()
	tierID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeleteRankedTier(ctx, tierID)
	if err != nil {
		if err.Error() == ranked.TierNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteRankedTier] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionRankedTierDelete, audit.TargetRankedTier, deleted.ID, rankedTierFields(deleted))
	logger.InfoF(ctx, "[DeleteRankedTier] %d deleted ranked tier %d %s", operator.ID, deleted.ID, deleted.Name)
	c.JSON(http.StatusOK, Response{})
}

// ListRankedSuspectsRequest 可疑名单查询参数
type ListRankedSuspectsRequest struct {
	Current  int   `json:"current" form:"current" binding:"min=1"`
	Size     int   `json:"size" form:"size" binding:"min=1,max=100"`
	Reviewed *bool `json:"reviewed" form:"reviewed"` // 为空返回全部; false 只返回待处理的, 即没处理过或处理后又一起打过排位; true 只返回其余的
}

// ListRankedSuspectsResponseData 可疑名单分页结果
type ListRankedSuspectsResponseData struct {
	Total    int64             `json:"total"`
	Suspects []auction.Suspect `json:"suspects"`
}

// ListRankedSuspects
// @Tags admin
// @Summary 获取排位的可疑名单
// @Description 最近若干天已结束的排位中, 一起打的局数较多, 或同局时其中一人多次低价成交的两个账号; 天数和次数见排位设置。其中一人低价成交多的排在前面
// @Produce json
// @Param request query ListRankedSuspectsRequest true "request query"
// @Success 200 {object} Response{data=ListRankedSuspectsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/ranked/suspects [get]
func ListRankedSuspects(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListRankedSuspectsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, suspects, err := auction.QuerySuspects(ctx, req.Reviewed, (req.Current-1)*req.Size, req.Size)
	if err != nil {
		logger.ErrorF(ctx, "[ListRankedSuspects] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListRankedSuspectsResponseData{Total: total, Suspects: suspects}})
}

// ReviewRankedSuspectRequest 处理可疑账号请求体
type ReviewRankedSuspectRequest struct {
	UserA       uint64 `json:"user_a" binding:"required"`
	UserB       uint64 `json:"user_b" binding:"required,gtfield=UserA"` // 大于 user_a
	LastMatchID uint64 `json:"last_match_id" binding:"required"`        // 处理到的对局, 即名单中的 last_match_id
	Note        string `json:"note" binding:"max=255"`                  // 处理结果, 如已封禁、正常组队
}

// ReviewRankedSuspect
// @Tags admin
// @Summary 把一对可疑账号标记为已处理
// @Description 封禁等处理在用户管理中进行; 已处理过时覆盖上一次的记录, 之后两人又一起打了排位时重新进入待处理
// @Accept json
// @Produce json
// @Param request body ReviewRankedSuspectRequest true "request body"
// @Success 200 {object} Response{data=auction.SuspectReview}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/ranked/suspects/review [post]
func ReviewRankedSuspect(c *gin.Context) {
	ctx := c.Request.Context()
	var req ReviewRankedSuspectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	note := strings.TrimSpace(req.Note)
	review, err := auction.ReviewSuspect(ctx, operator.ID, req.UserA, req.UserB, req.LastMatchID, note)
	if err != nil {
		logger.ErrorF(ctx, "[ReviewRankedSuspect] review failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionSuspectReview, audit.TargetSuspect, req.UserA, map[string]interface{}{
		"user_b":        req.UserB,
		"last_match_id": req.LastMatchID,
		"note":          note,
	})
	logger.InfoF(ctx, "[ReviewRankedSuspect] %d reviewed suspects %d and %d up to match %d", operator.ID, req.UserA, req.UserB, req.LastMatchID)
	c.JSON(http.StatusOK, Response{Data: review})
}

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
// @Success 200 {object} Response{data=ListShopItemsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/shop/items [get]
func ListShopItems(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListShopItemsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	data, err := QueryShopItems(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListShopItems] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: data})
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

// shopItemErrorStatus 商品配置错误对应的状态码, 不是配置错误时返回 0
func shopItemErrorStatus(err error) int {
	var invalid *shop.InvalidError
	switch {
	case errors.As(err, &invalid), err.Error() == shop.CodeImmutable:
		return http.StatusBadRequest
	case err.Error() == shop.ItemNotFound:
		return http.StatusNotFound
	case err.Error() == shop.CodeTaken:
		return http.StatusConflict
	}
	return 0
}

// CreateShopItem
// @Tags admin
// @Summary 新建商品
// @Accept json
// @Produce json
// @Param request body ShopItemRequest true "request body"
// @Success 200 {object} Response{data=shop.Item}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/shop/items [post]
func CreateShopItem(c *gin.Context) {
	ctx := c.Request.Context()
	var req ShopItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateShopItem(ctx, &req)
	if err != nil {
		if status := shopItemErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateShopItem] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionShopItemCreate, audit.TargetShopItem, created.ID, shopItemFields(created))
	logger.InfoF(ctx, "[CreateShopItem] %d created shop item %d %s", operator.ID, created.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateShopItem
// @Tags admin
// @Summary 修改商品
// @Description 需提交全部字段, 代码不能修改; 已有订单不受影响
// @Accept json
// @Produce json
// @Param id path int true "shop item id"
// @Param request body ShopItemRequest true "request body"
// @Success 200 {object} Response{data=shop.Item}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/shop/items/{id} [put]
func UpdateShopItem(c *gin.Context) {
	ctx := c.Request.Context()
	var req ShopItemRequest
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
	updated, changes, err := doUpdateShopItem(ctx, itemID, &req)
	if err != nil {
		if status := shopItemErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateShopItem] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionShopItemUpdate, audit.TargetShopItem, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateShopItem] %d updated shop item %d %s", operator.ID, updated.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteShopItem
// @Tags admin
// @Summary 删除商品
// @Description 软删除, 代码不能再被新商品使用; 已有订单仍可退款
// @Produce json
// @Param id path int true "shop item id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/shop/items/{id} [delete]
func DeleteShopItem(c *gin.Context) {
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
	deleted, err := doDeleteShopItem(ctx, itemID)
	if err != nil {
		if err.Error() == shop.ItemNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteShopItem] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionShopItemDelete, audit.TargetShopItem, deleted.ID, map[string]interface{}{
		"code": deleted.Code,
		"name": deleted.Name,
	})
	logger.InfoF(ctx, "[DeleteShopItem] %d deleted shop item %d %s", operator.ID, deleted.ID, deleted.Code)
	c.JSON(http.StatusOK, Response{})
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
// @Success 200 {object} Response{data=ListShopOrdersResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/shop/orders [get]
func ListShopOrders(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListShopOrdersRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, orders, err := QueryShopOrders(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListShopOrders] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListShopOrdersResponseData{Total: total, Orders: orders}})
}

// RefundShopOrder
// @Tags admin
// @Summary 订单退款
// @Description 退还价格并收回商品, 恢复库存, 不恢复限购数量; 价格中已下架的物品按回收价折成货币退还, 用户的货币或物品不足以收回商品时返回 409
// @Produce json
// @Param id path int true "order id"
// @Success 200 {object} Response{data=shop.Order}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/shop/orders/{id}/refund [post]
func RefundShopOrder(c *gin.Context) {
	ctx := c.Request.Context()
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	order, err := shop.Refund(ctx, operator.ID, orderID)
	if err != nil {
		switch err.Error() {
		case shop.OrderNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case shop.OrderRefunded, shop.RefundInsufficient, shop.RefundUnavailable:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[RefundShopOrder] refund failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionShopOrderRefund, audit.TargetShopOrder, order.ID, map[string]interface{}{
		"user_id":   order.UserID,
		"item_code": order.ItemCode,
		"quantity":  order.Quantity,
		"price":     order.Price,
		"goods":     order.Goods,
	})
	logger.InfoF(ctx, "[RefundShopOrder] %d refunded shop order %d of user %d", operator.ID, order.ID, order.UserID)
	c.JSON(http.StatusOK, Response{Data: order})
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

// ExportPackRequest 导出配置包
type ExportPackRequest struct {
	Name     string   `json:"name" binding:"required,max=64"` // 配置包的名称
	Author   string   `json:"author" binding:"max=64"`
	Sections []string `json:"sections" binding:"required,min=1,max=8"` // 导出的配置种类: currencies、qualities、items、loot_pools、venues、shop_items、leaderboards、settings
}

// ExportPack
// @Tags admin
// @Summary 导出配置包
// @Description zip 文件, 包含 manifest.json、每种配置一个 JSON 文件, 以及 images/ 下引用到的本站图片;
// @Description 已删除的配置不导出, 掉落池中已删除物品的条目也不导出; 物品按 ID 导出, 设置只导出游戏分类且不含密钥字段
// @Accept json
// @Produce application/zip
// @Param request body ExportPackRequest true "request body"
// @Success 200 {file} file
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/packs/export [post]
func ExportPack(c *gin.Context) {
	ctx := c.Request.Context()
	var req ExportPackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	data, counts, err := doExportPack(ctx, &req)
	if err != nil {
		var packErr *PackError
		if errors.As(err, &packErr) {
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[ExportPack] export failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionPackExport, audit.TargetPack, 0, map[string]interface{}{
		"name":   strings.TrimSpace(req.Name),
		"counts": counts,
		"size":   len(data),
	})
	logger.InfoF(ctx, "[ExportPack] %d exported pack %v", operator.ID, counts)
	filename := fmt.Sprintf("blindcrate-pack-%s.zip", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/zip", data)
}

// readPackForm 读取上传的配置包和处理方式, 不符合要求时写入响应并返回 false
func readPackForm(c *gin.Context, action string) ([]byte, []PackAction, bool) {
	ctx := c.Request.Context()
	tooLarge := fmt.Sprintf(PackTooLarge, maxPackSize>>20)
	// 请求头中的大小已经超过上限时直接拒绝, 不读取请求体
	if c.Request.ContentLength > maxPackSize+packFormOverhead {
		c.JSON(http.StatusRequestEntityTooLarge, Response{ErrorMsg: tooLarge})
		return nil, nil, false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPackSize+packFormOverhead)
	if err := c.Request.ParseMultipartForm(packFormOverhead); err != nil {
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			c.JSON(http.StatusRequestEntityTooLarge, Response{ErrorMsg: tooLarge})
			return nil, nil, false
		}
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: PackRequired})
		return nil, nil, false
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: PackRequired})
		return nil, nil, false
	}
	if file.Size > maxPackSize {
		c.JSON(http.StatusRequestEntityTooLarge, Response{ErrorMsg: tooLarge})
		return nil, nil, false
	}
	var actions []PackAction
	if raw := c.PostForm("actions"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &actions); err != nil {
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: PackActionInvalid})
			return nil, nil, false
		}
	}
	f, err := file.Open()
	if err != nil {
		logger.ErrorF(ctx, "[%s] open file failed: %v", action, err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return nil, nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxPackSize+1))
	if err != nil {
		logger.ErrorF(ctx, "[%s] read file failed: %v", action, err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return nil, nil, false
	}
	return data, actions, true
}

// PreviewPack
// @Tags admin
// @Summary 预览配置包
// @Description multipart 表单, 按依赖顺序列出每一项与已有配置的比较结果: 新增、覆盖、相同或冲突(带原因), 不写入任何内容;
// @Description 在回滚的事务中按处理方式试着导入, 校验规则与导入相同; 货币、掉落池、地图、商品、榜单按代码对应, 品质按等级, 物品按 ID, 设置按设置组
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "配置包"
// @Param actions formData string false "处理方式, PackAction 的 JSON 数组, 没有提交的项按覆盖处理"
// @Success 200 {object} Response{data=PackPreview}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 413 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/packs/preview [post]
func PreviewPack(c *gin.Context) {
	ctx := c.Request.Context()
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	data, actions, ok := readPackForm(c, "PreviewPack")
	if !ok {
		return
	}
	preview, err := doPreviewPack(ctx, operator.ID, data, actions)
	if err != nil {
		var packErr *PackError
		if errors.As(err, &packErr) {
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[PreviewPack] preview failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: preview})
}

// ImportPack
// @Tags admin
// @Summary 导入配置包
// @Description multipart 表单, 在一个事务中按依赖顺序写入, 全部成功或全部回滚; 只新增或覆盖配置, 不会停用或删除已有配置, 也不改动玩家数据, 已有商品的剩余库存保持不变;
// @Description 有未跳过的冲突项时返回 409, data 为每一项的结果; 设置按版本保存, 来源为 import, 之后可以回滚
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "配置包"
// @Param actions formData string false "处理方式, PackAction 的 JSON 数组, 没有提交的项按覆盖处理"
// @Success 200 {object} Response{data=Import}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response{data=[]PackEntryResult}
// @Failure 413 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/packs/import [post]
func ImportPack(c *gin.Context) {
	ctx := c.Request.Context()
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	data, actions, ok := readPackForm(c, "ImportPack")
	if !ok {
		return
	}
	record, results, err := doImportPack(ctx, operator.ID, data, actions)
	if err != nil {
		var packErr *PackError
		switch {
		case errors.As(err, &packErr) && results != nil:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error(), Data: results})
		case errors.As(err, &packErr):
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[ImportPack] import failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionPackImport, audit.TargetPack, record.ID, map[string]interface{}{
		"name":    record.Name,
		"author":  record.Author,
		"version": record.Version,
		"digest":  record.Digest,
		"size":    record.Size,
		"created": record.Created,
		"updated": record.Updated,
		"skipped": record.Skipped,
	})
	logger.InfoF(ctx, "[ImportPack] %d imported pack %d %s: %d created, %d updated, %d skipped",
		operator.ID, record.ID, record.Name, record.Created, record.Updated, record.Skipped)
	c.JSON(http.StatusOK, Response{Data: record})
}

// ListPackImportsRequest 导入记录查询参数
type ListPackImportsRequest struct {
	Current int `json:"current" form:"current" binding:"min=1"`
	Size    int `json:"size" form:"size" binding:"min=1,max=100"`
}

// ListPackImportsResponseData 导入记录分页结果
type ListPackImportsResponseData struct {
	Total   int64    `json:"total"`
	Imports []Import `json:"imports"` // 不含每一项的结果
}

// ListPackImports
// @Tags admin
// @Summary 获取导入记录列表
// @Description 按导入时间从晚到早排列, 每一项的结果见导入记录详情
// @Produce json
// @Param request query ListPackImportsRequest true "request query"
// @Success 200 {object} Response{data=ListPackImportsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/packs/imports [get]
func ListPackImports(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListPackImportsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, imports, err := QueryPackImports(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListPackImports] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListPackImportsResponseData{Total: total, Imports: imports}})
}

// GetPackImport
// @Tags admin
// @Summary 获取导入记录详情
// @Description 包括每一项的处理结果
// @Produce json
// @Param id path int true "import id"
// @Success 200 {object} Response{data=Import}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/packs/imports/{id} [get]
func GetPackImport(c *gin.Context) {
	ctx := c.Request.Context()
	importID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	record, err := queryPackImport(ctx, importID)
	if err != nil {
		if err.Error() == ImportNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[GetPackImport] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: record})
}
