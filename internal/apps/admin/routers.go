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
	"net/http"
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/audit"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

type ListUsersRequest struct {
	Current    int    `json:"current" form:"current" binding:"min=1"`
	Size       int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword    string `json:"keyword" form:"keyword" binding:"max=255"` // 按用户名或邮箱前缀搜索, 不区分大小写
	IsAdmin    *bool  `json:"is_admin" form:"is_admin"`
	IsDisabled *bool  `json:"is_disabled" form:"is_disabled"`
	IsBanned   *bool  `json:"is_banned" form:"is_banned"`
}

type ListUsersResponseData struct {
	Total int64       `json:"total"`
	Users []auth.User `json:"users"`
}

type ListUsersResponse struct {
	ErrorMsg string                 `json:"error_msg"`
	Data     *ListUsersResponseData `json:"data"`
}

// ListUsers 获取用户列表
// @Tags admin
// @Param request query ListUsersRequest true "request query"
// @Produce json
// @Success 200 {object} ListUsersResponse
// @Failure 400 {object} ListUsersResponse
// @Failure 401 {object} ListUsersResponse
// @Failure 403 {object} ListUsersResponse
// @Failure 500 {object} ListUsersResponse
// @Router /api/v1/admin/users [get]
func ListUsers(c *gin.Context) {
	// init req
	req := &ListUsersRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, ListUsersResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	// query
	total, users, err := QueryUsersList(c.Request.Context(), req)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[ListUsers] query failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, ListUsersResponse{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, ListUsersResponse{Data: &ListUsersResponseData{Total: total, Users: users}})
}

type UpdateUserRequest struct {
	Username  *string `json:"username"`   // 不传表示不修改
	Nickname  *string `json:"nickname"`   // 不传表示不修改
	AvatarURL *string `json:"avatar_url"` // 不传表示不修改, 传空字符串表示清除头像
}

type UpdateUserResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// UpdateUser 修改用户资料
// @Tags admin
// @Param id path int true "user id"
// @Param request body UpdateUserRequest true "request body"
// @Produce json
// @Success 200 {object} UpdateUserResponse
// @Failure 400 {object} UpdateUserResponse
// @Failure 401 {object} UpdateUserResponse
// @Failure 403 {object} UpdateUserResponse
// @Failure 404 {object} UpdateUserResponse
// @Failure 409 {object} UpdateUserResponse
// @Failure 500 {object} UpdateUserResponse
// @Router /api/v1/admin/users/{id} [patch]
func UpdateUser(c *gin.Context) {
	// init req
	var req UpdateUserRequest
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, UpdateUserResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, UpdateUserResponse{ErrorMsg: auth.UnAuthorized})
		return
	}
	// do update
	updated, err := auth.UpdateUserProfile(c.Request.Context(), userID, req.Username, req.Nickname, req.AvatarURL)
	if err != nil {
		switch err.Error() {
		case auth.UsernameInvalid, auth.NicknameInvalid, auth.AvatarURLInvalid:
			c.JSON(http.StatusBadRequest, UpdateUserResponse{ErrorMsg: err.Error()})
		case auth.UserNotFound:
			c.JSON(http.StatusNotFound, UpdateUserResponse{ErrorMsg: err.Error()})
		case auth.UsernameTaken:
			c.JSON(http.StatusConflict, UpdateUserResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[UpdateUser] update failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, UpdateUserResponse{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionUserUpdate, audit.TargetUser, updated.ID, auth.ProfileChanges(req.Username, req.Nickname, req.AvatarURL))
	logger.InfoF(c.Request.Context(), "[UpdateUser] %d updated %d %s", operator.ID, updated.ID, updated.Username)
	c.JSON(
		http.StatusOK,
		UpdateUserResponse{
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

type BanUserRequest struct {
	BannedUntil time.Time `json:"banned_until" binding:"required"` // RFC3339, 必须晚于当前时间; 永久封禁请使用停用
	Reason      string    `json:"reason" binding:"max=255"`
}

type BanUserResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// BanUser 封禁用户, 并让其所有设备下线
// @Tags admin
// @Param id path int true "user id"
// @Param request body BanUserRequest true "request body"
// @Produce json
// @Success 200 {object} BanUserResponse
// @Failure 400 {object} BanUserResponse
// @Failure 401 {object} BanUserResponse
// @Failure 403 {object} BanUserResponse
// @Failure 404 {object} BanUserResponse
// @Failure 500 {object} BanUserResponse
// @Router /api/v1/admin/users/{id}/ban [put]
func BanUser(c *gin.Context) {
	// init req
	var req BanUserRequest
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, BanUserResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, BanUserResponse{ErrorMsg: auth.UnAuthorized})
		return
	}
	// do ban
	if err := doBanUser(c.Request.Context(), operator.ID, userID, req.BannedUntil, req.Reason); err != nil {
		switch err.Error() {
		case CannotOperateSelf, BanUntilInvalid:
			c.JSON(http.StatusBadRequest, BanUserResponse{ErrorMsg: err.Error()})
		case auth.UserNotFound:
			c.JSON(http.StatusNotFound, BanUserResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[BanUser] ban failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, BanUserResponse{ErrorMsg: auth.InternalError})
		}
		return
	}
	// 所有设备下线
	if err := auth.RevokeUserSessions(c.Request.Context(), userID, ""); err != nil {
		logger.ErrorF(c.Request.Context(), "[BanUser] revoke sessions failed: %s", err.Error())
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionUserBan, audit.TargetUser, userID, map[string]interface{}{"banned_until": req.BannedUntil, "reason": req.Reason})
	logger.InfoF(c.Request.Context(), "[BanUser] %d banned %d until %s", operator.ID, userID, req.BannedUntil.Format(time.RFC3339))
	c.JSON(http.StatusOK, BanUserResponse{})
}

type UnbanUserResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// UnbanUser 解除封禁
// @Tags admin
// @Param id path int true "user id"
// @Produce json
// @Success 200 {object} UnbanUserResponse
// @Failure 400 {object} UnbanUserResponse
// @Failure 401 {object} UnbanUserResponse
// @Failure 403 {object} UnbanUserResponse
// @Failure 404 {object} UnbanUserResponse
// @Failure 500 {object} UnbanUserResponse
// @Router /api/v1/admin/users/{id}/ban [delete]
func UnbanUser(c *gin.Context) {
	// init req
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, UnbanUserResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, UnbanUserResponse{ErrorMsg: auth.UnAuthorized})
		return
	}
	// do unban
	if err := doUnbanUser(c.Request.Context(), operator.ID, userID); err != nil {
		switch err.Error() {
		case CannotOperateSelf:
			c.JSON(http.StatusBadRequest, UnbanUserResponse{ErrorMsg: err.Error()})
		case auth.UserNotFound:
			c.JSON(http.StatusNotFound, UnbanUserResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[UnbanUser] unban failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, UnbanUserResponse{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionUserUnban, audit.TargetUser, userID, nil)
	logger.InfoF(c.Request.Context(), "[UnbanUser] %d unbanned %d", operator.ID, userID)
	c.JSON(http.StatusOK, UnbanUserResponse{})
}

type SetUserDisabledRequest struct {
	Disabled *bool `json:"disabled" binding:"required"`
}

type SetUserDisabledResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// SetUserDisabled 停用或启用用户, 停用时让其所有设备下线
// @Tags admin
// @Param id path int true "user id"
// @Param request body SetUserDisabledRequest true "request body"
// @Produce json
// @Success 200 {object} SetUserDisabledResponse
// @Failure 400 {object} SetUserDisabledResponse
// @Failure 401 {object} SetUserDisabledResponse
// @Failure 403 {object} SetUserDisabledResponse
// @Failure 404 {object} SetUserDisabledResponse
// @Failure 500 {object} SetUserDisabledResponse
// @Router /api/v1/admin/users/{id}/disabled [put]
func SetUserDisabled(c *gin.Context) {
	// init req
	var req SetUserDisabledRequest
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, SetUserDisabledResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, SetUserDisabledResponse{ErrorMsg: auth.UnAuthorized})
		return
	}
	// do set
	if err := doSetUserDisabled(c.Request.Context(), operator.ID, userID, *req.Disabled); err != nil {
		switch err.Error() {
		case CannotOperateSelf:
			c.JSON(http.StatusBadRequest, SetUserDisabledResponse{ErrorMsg: err.Error()})
		case auth.UserNotFound:
			c.JSON(http.StatusNotFound, SetUserDisabledResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[SetUserDisabled] set failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, SetUserDisabledResponse{ErrorMsg: auth.InternalError})
		}
		return
	}
	// 停用时所有设备下线
	if *req.Disabled {
		if err := auth.RevokeUserSessions(c.Request.Context(), userID, ""); err != nil {
			logger.ErrorF(c.Request.Context(), "[SetUserDisabled] revoke sessions failed: %s", err.Error())
		}
	}
	action := audit.ActionUserEnable
	if *req.Disabled {
		action = audit.ActionUserDisable
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, action, audit.TargetUser, userID, nil)
	logger.InfoF(c.Request.Context(), "[SetUserDisabled] %d set %d disabled=%t", operator.ID, userID, *req.Disabled)
	c.JSON(http.StatusOK, SetUserDisabledResponse{})
}

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

type ListAuditLogsResponseData struct {
	Total int64          `json:"total"`
	Logs  []AuditLogItem `json:"logs"`
}

type ListAuditLogsResponse struct {
	ErrorMsg string                     `json:"error_msg"`
	Data     *ListAuditLogsResponseData `json:"data"`
}

// ListAuditLogs 获取操作日志列表(用户和管理员的操作)
// @Tags admin
// @Param request query ListAuditLogsRequest true "request query"
// @Produce json
// @Success 200 {object} ListAuditLogsResponse
// @Failure 400 {object} ListAuditLogsResponse
// @Failure 401 {object} ListAuditLogsResponse
// @Failure 403 {object} ListAuditLogsResponse
// @Failure 500 {object} ListAuditLogsResponse
// @Router /api/v1/admin/audit-logs [get]
func ListAuditLogs(c *gin.Context) {
	// init req
	req := &ListAuditLogsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, ListAuditLogsResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	// query
	total, logs, err := QueryAuditLogs(c.Request.Context(), req)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[ListAuditLogs] query failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, ListAuditLogsResponse{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, ListAuditLogsResponse{Data: &ListAuditLogsResponseData{Total: total, Logs: logs}})
}

type ListCurrenciesResponseData struct {
	Currencies []wallet.Currency `json:"currencies"`
}

type ListCurrenciesResponse struct {
	ErrorMsg string                      `json:"error_msg"`
	Data     *ListCurrenciesResponseData `json:"data"`
}

// ListCurrencies 获取全部货币定义(含已停用)
// @Tags admin
// @Produce json
// @Success 200 {object} ListCurrenciesResponse
// @Failure 401 {object} ListCurrenciesResponse
// @Failure 403 {object} ListCurrenciesResponse
// @Failure 500 {object} ListCurrenciesResponse
// @Router /api/v1/admin/wallet/currencies [get]
func ListCurrencies(c *gin.Context) {
	// query
	currencies, err := wallet.QueryCurrencies(c.Request.Context())
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[ListCurrencies] query failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, ListCurrenciesResponse{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, ListCurrenciesResponse{Data: &ListCurrenciesResponseData{Currencies: currencies}})
}

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

type ListWalletTransactionsResponseData struct {
	Total        int64                   `json:"total"`
	Transactions []WalletTransactionItem `json:"transactions"`
}

type ListWalletTransactionsResponse struct {
	ErrorMsg string                              `json:"error_msg"`
	Data     *ListWalletTransactionsResponseData `json:"data"`
}

// ListWalletTransactions 获取全部用户的余额流水
// @Tags admin
// @Param request query ListWalletTransactionsRequest true "request query"
// @Produce json
// @Success 200 {object} ListWalletTransactionsResponse
// @Failure 400 {object} ListWalletTransactionsResponse
// @Failure 401 {object} ListWalletTransactionsResponse
// @Failure 403 {object} ListWalletTransactionsResponse
// @Failure 500 {object} ListWalletTransactionsResponse
// @Router /api/v1/admin/wallet/transactions [get]
func ListWalletTransactions(c *gin.Context) {
	// init req
	req := &ListWalletTransactionsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, ListWalletTransactionsResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	// query
	total, transactions, err := QueryWalletTransactions(c.Request.Context(), req)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[ListWalletTransactions] query failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, ListWalletTransactionsResponse{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, ListWalletTransactionsResponse{Data: &ListWalletTransactionsResponseData{Total: total, Transactions: transactions}})
}

type ListUserBalancesResponseData struct {
	Balances []wallet.BalanceItem `json:"balances"`
}

type ListUserBalancesResponse struct {
	ErrorMsg string                        `json:"error_msg"`
	Data     *ListUserBalancesResponseData `json:"data"`
}

// ListUserBalances 获取指定用户的各货币余额(含已停用货币)
// @Tags admin
// @Param id path int true "user id"
// @Produce json
// @Success 200 {object} ListUserBalancesResponse
// @Failure 400 {object} ListUserBalancesResponse
// @Failure 401 {object} ListUserBalancesResponse
// @Failure 403 {object} ListUserBalancesResponse
// @Failure 404 {object} ListUserBalancesResponse
// @Failure 500 {object} ListUserBalancesResponse
// @Router /api/v1/admin/users/{id}/wallet/balances [get]
func ListUserBalances(c *gin.Context) {
	// init req
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ListUserBalancesResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	// query
	balances, err := queryUserBalances(c.Request.Context(), userID)
	if err != nil {
		if err.Error() == auth.UserNotFound {
			c.JSON(http.StatusNotFound, ListUserBalancesResponse{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(c.Request.Context(), "[ListUserBalances] query failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, ListUserBalancesResponse{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, ListUserBalancesResponse{Data: &ListUserBalancesResponseData{Balances: balances}})
}

type AdjustUserBalanceRequest struct {
	Currency       string `json:"currency" binding:"required,max=32"`
	Amount         int64  `json:"amount" binding:"required"`                            // 正数加款, 负数扣款, 扣款后余额不能为负
	Remark         string `json:"remark" binding:"required"`                            // 调账原因, 用户可在流水中看到
	IdempotencyKey string `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次打开调账表单生成一个, 重复提交只生效一次
}

type AdjustUserBalanceResponse struct {
	ErrorMsg string              `json:"error_msg"`
	Data     *wallet.Transaction `json:"data"`
}

// AdjustUserBalance 管理员给用户调账, 写入一条 admin_adjust 流水
// @Tags admin
// @Param id path int true "user id"
// @Param request body AdjustUserBalanceRequest true "request body"
// @Produce json
// @Success 200 {object} AdjustUserBalanceResponse
// @Failure 400 {object} AdjustUserBalanceResponse
// @Failure 401 {object} AdjustUserBalanceResponse
// @Failure 403 {object} AdjustUserBalanceResponse
// @Failure 404 {object} AdjustUserBalanceResponse
// @Failure 409 {object} AdjustUserBalanceResponse
// @Failure 500 {object} AdjustUserBalanceResponse
// @Router /api/v1/admin/users/{id}/wallet/adjust [post]
func AdjustUserBalance(c *gin.Context) {
	// init req
	var req AdjustUserBalanceRequest
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, AdjustUserBalanceResponse{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, AdjustUserBalanceResponse{ErrorMsg: auth.UnAuthorized})
		return
	}
	// do adjust
	txn, replayed, err := doAdjustUserBalance(c.Request.Context(), operator.ID, userID, &req)
	if err != nil {
		switch err.Error() {
		case CannotOperateSelf, RemarkInvalid, wallet.AmountInvalid, wallet.CurrencyNotFound, wallet.CurrencyDisabled, wallet.InsufficientBalance:
			c.JSON(http.StatusBadRequest, AdjustUserBalanceResponse{ErrorMsg: err.Error()})
		case auth.UserNotFound:
			c.JSON(http.StatusNotFound, AdjustUserBalanceResponse{ErrorMsg: err.Error()})
		case wallet.IdempotencyConflict, wallet.DuplicateRequest:
			c.JSON(http.StatusConflict, AdjustUserBalanceResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[AdjustUserBalance] adjust failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, AdjustUserBalanceResponse{ErrorMsg: auth.InternalError})
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
		logger.InfoF(c.Request.Context(), "[AdjustUserBalance] %d adjusted %d %s by %d, balance %d", operator.ID, userID, txn.Currency, txn.Amount, txn.BalanceAfter)
	}
	c.JSON(http.StatusOK, AdjustUserBalanceResponse{Data: txn})
}
