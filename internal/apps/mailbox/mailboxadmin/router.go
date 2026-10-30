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

// Package mailboxadmin 游戏邮件的后台接口: 发给指定用户、全服邮件和邮件查询
package mailboxadmin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/mailbox"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// letterErrorStatus 邮件内容错误对应的状态码, 不是内容错误时返回 0
func letterErrorStatus(err error) int {
	var invalid *mailbox.InvalidError
	if errors.As(err, &invalid) {
		return http.StatusBadRequest
	}
	return 0
}

// ListMailSourcesResponseData 邮件来源
type ListMailSourcesResponseData struct {
	Sources []mailbox.Source `json:"sources"`
}

// ListMailSources
// @Tags admin
// @Summary 获取邮件来源
// @Description 按登记顺序, 用于邮件查询的筛选和展示
// @Produce json
// @Success 200 {object} Response{data=ListMailSourcesResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Router /api/v1/admin/mails/sources [get]
func ListMailSources(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Data: ListMailSourcesResponseData{Sources: mailbox.Sources()}})
}

// SendMailsRequest 发给指定用户的请求体
//
// 附件不在这里校验, 由 mailbox.ValidateLetter 校验, 提示更具体。
type SendMailsRequest struct {
	UserIDs        []uint64      `json:"user_ids" binding:"required,min=1,max=1000,dive,min=1"`
	Sender         string        `json:"sender" binding:"max=32"` // 为空时为 "系统"
	Title          string        `json:"title" binding:"required,max=64"`
	Content        string        `json:"content" binding:"max=2000"`
	Attachments    reward.Bundle `json:"attachments" binding:"-"`
	ExpireDays     int           `json:"expire_days" binding:"min=0,max=365"`                  // 有效期, 0 表示使用默认有效期
	IdempotencyKey string        `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次打开发送框生成一个, 重复提交只发送一次
}

// SendMailsResponseData 发送结果
type SendMailsResponseData struct {
	Sent int64 `json:"sent"` // 实际发送的封数, 重复提交时不计已发送的
}

// SendMails
// @Tags admin
// @Summary 发邮件给指定用户
// @Description 一次最多 1000 人, 用户不存在时整体不发送; 幂等键相同的重复提交不会重复发送
// @Accept json
// @Produce json
// @Param request body SendMailsRequest true "request body"
// @Success 200 {object} Response{data=SendMailsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/mails [post]
func SendMails(c *gin.Context) {
	ctx := c.Request.Context()
	var req SendMailsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	sent, letter, err := doSendMails(ctx, &req)
	if err != nil {
		if status := letterErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[SendMails] send failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if sent > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, mailbox.ActionSend, mailbox.TargetMail, 0, map[string]interface{}{
			"user_ids":    req.UserIDs,
			"title":       letter.Title,
			"attachments": letter.Attachments,
			"sent":        sent,
		})
		logger.InfoF(ctx, "[SendMails] %d sent %d mails", operator.ID, sent)
	}
	c.JSON(http.StatusOK, Response{Data: SendMailsResponseData{Sent: sent}})
}

// ListMailsRequest 邮件查询参数
type ListMailsRequest struct {
	Current int     `json:"current" form:"current" binding:"min=1"`
	Size    int     `json:"size" form:"size" binding:"min=1,max=100"`
	UserID  *uint64 `json:"user_id" form:"user_id"`
	Source  string  `json:"source" form:"source" binding:"max=16"`
	Keyword string  `json:"keyword" form:"keyword" binding:"max=64"` // 按标题搜索, 不区分大小写
}

// ListMailsResponseData 邮件分页结果
type ListMailsResponseData struct {
	Total int64      `json:"total"`
	Mails []MailItem `json:"mails"`
}

// ListMails
// @Tags admin
// @Summary 查询邮件
// @Description 从新到旧, 包括已删除的邮件, 带上收件人的用户名
// @Produce json
// @Param request query ListMailsRequest true "request query"
// @Success 200 {object} Response{data=ListMailsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/mails [get]
func ListMails(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListMailsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, mails, err := QueryMails(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListMails] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListMailsResponseData{Total: total, Mails: mails}})
}

// ListBroadcastsRequest 全服邮件查询参数
type ListBroadcastsRequest struct {
	Current int    `json:"current" form:"current" binding:"min=1"`
	Size    int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword string `json:"keyword" form:"keyword" binding:"max=64"` // 按标题搜索, 不区分大小写
}

// ListBroadcastsResponseData 全服邮件分页结果
type ListBroadcastsResponseData struct {
	Total      int64           `json:"total"`
	Broadcasts []BroadcastItem `json:"broadcasts"`
}

// ListMailBroadcasts
// @Tags admin
// @Summary 获取全服邮件列表
// @Description 从新到旧, 带上已放进信箱和已领取附件的人数
// @Produce json
// @Param request query ListBroadcastsRequest true "request query"
// @Success 200 {object} Response{data=ListBroadcastsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/mail-broadcasts [get]
func ListMailBroadcasts(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListBroadcastsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, items, err := QueryBroadcasts(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListMailBroadcasts] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListBroadcastsResponseData{Total: total, Broadcasts: items}})
}

// BroadcastRequest 发送全服邮件的请求体
//
// 附件和收件条件不在这里校验, 由 mailbox.ValidateLetter 和 condition.Validate 校验, 提示更具体。
type BroadcastRequest struct {
	Sender          string                `json:"sender" binding:"max=32"` // 为空时为 "系统"
	Title           string                `json:"title" binding:"required,max=64"`
	Content         string                `json:"content" binding:"max=2000"`
	Attachments     reward.Bundle         `json:"attachments" binding:"-"`
	Requirements    []condition.Condition `json:"requirements" binding:"-"`                             // 收件条件, 需要全部满足; 玩家打开信箱时检查, 之后满足的也能收到
	IncludeNewUsers bool                  `json:"include_new_users"`                                    // 发送之后注册的用户也能收到, 默认只发给已注册的用户
	ExpireDays      int                   `json:"expire_days" binding:"min=0,max=365"`                  // 有效期, 0 表示使用默认有效期
	IdempotencyKey  string                `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次打开发送框生成一个, 重复提交只发送一次
}

// CreateMailBroadcast
// @Tags admin
// @Summary 发送全服邮件
// @Description 只写一条记录, 玩家打开信箱时放进信箱; 发送后不能修改, 只能撤回。幂等键相同的重复提交返回已有的全服邮件
// @Accept json
// @Produce json
// @Param request body BroadcastRequest true "request body"
// @Success 200 {object} Response{data=mailbox.Broadcast}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/mail-broadcasts [post]
func CreateMailBroadcast(c *gin.Context) {
	ctx := c.Request.Context()
	var req BroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, replayed, err := doCreateBroadcast(ctx, operator.ID, &req)
	if err != nil {
		if status := letterErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateMailBroadcast] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if replayed {
		c.JSON(http.StatusOK, Response{Data: created})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, mailbox.ActionBroadcast, mailbox.TargetBroadcast, created.ID, map[string]interface{}{
		"title":             created.Title,
		"attachments":       created.Attachments,
		"requirements":      created.Requirements,
		"include_new_users": req.IncludeNewUsers,
		"expires_at":        created.ExpiresAt,
	})
	logger.InfoF(ctx, "[CreateMailBroadcast] %d created mail broadcast %d", operator.ID, created.ID)
	c.JSON(http.StatusOK, Response{Data: created})
}

// RevokeMailBroadcastResponseData 撤回结果
type RevokeMailBroadcastResponseData struct {
	Broadcast *mailbox.Broadcast `json:"broadcast"`
	Removed   int64              `json:"removed"` // 移出信箱的封数, 已领取附件的邮件保留
}

// RevokeMailBroadcast
// @Tags admin
// @Summary 撤回全服邮件
// @Description 不再放进信箱; 已放进信箱但附件未领取的邮件移出信箱, 已领取的附件不收回
// @Produce json
// @Param id path int true "broadcast id"
// @Success 200 {object} Response{data=RevokeMailBroadcastResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/mail-broadcasts/{id}/revoke [post]
func RevokeMailBroadcast(c *gin.Context) {
	ctx := c.Request.Context()
	broadcastID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	revoked, removed, err := doRevokeBroadcast(ctx, operator.ID, broadcastID)
	if err != nil {
		switch err.Error() {
		case mailbox.BroadcastNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case mailbox.BroadcastRevoked:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[RevokeMailBroadcast] revoke failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, mailbox.ActionRevoke, mailbox.TargetBroadcast, revoked.ID, map[string]interface{}{
		"title":   revoked.Title,
		"removed": removed,
	})
	logger.InfoF(ctx, "[RevokeMailBroadcast] %d revoked mail broadcast %d, removed %d mails", operator.ID, revoked.ID, removed)
	c.JSON(http.StatusOK, Response{Data: RevokeMailBroadcastResponseData{Broadcast: revoked, Removed: removed}})
}
