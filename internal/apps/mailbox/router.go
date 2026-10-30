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

package mailbox

import (
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 mailbox 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListMailsRequest 邮件查询参数
type ListMailsRequest struct {
	Current int `json:"current" form:"current" binding:"min=1"`
	Size    int `json:"size" form:"size" binding:"min=1,max=100"`
}

// ListMailsResponseData 邮件分页结果
type ListMailsResponseData struct {
	Total int64      `json:"total"`
	Mails []MailView `json:"mails"`
}

// ListMails
// @Tags mailbox
// @Summary 获取信箱中的邮件
// @Description 从新到旧, 不含已删除和已过期的邮件; 会先把可以收到的全服邮件放进信箱
// @Produce json
// @Param request query ListMailsRequest true "request query"
// @Success 200 {object} Response{data=ListMailsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/mails [get]
func ListMails(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListMailsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	total, mails, err := QueryMails(ctx, user, (req.Current-1)*req.Size, req.Size)
	if err != nil {
		logger.ErrorF(ctx, "[ListMails] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListMailsResponseData{Total: total, Mails: mails}})
}

// GetCounts
// @Tags mailbox
// @Summary 获取未读和附件未领取的封数
// @Description 会先把可以收到的全服邮件放进信箱; 收到 mailbox:new 推送或切回页面时调用
// @Produce json
// @Success 200 {object} Response{data=Counts}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/mails/counts [get]
func GetCounts(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	counts, err := QueryCounts(ctx, user)
	if err != nil {
		logger.ErrorF(ctx, "[GetMailCounts] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: counts})
}

// mailID 解析路径中的邮件 ID, 不合法时返回 false 并写入 400
func mailID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return 0, false
	}
	return id, true
}

// ReadMail
// @Tags mailbox
// @Summary 标记邮件为已读
// @Produce json
// @Param id path int true "mail id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/mails/{id}/read [post]
func ReadMail(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := mailID(c)
	if !ok {
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	if err := Read(ctx, user.ID, id); err != nil {
		if err.Error() == MailNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[ReadMail] read failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{})
}

// ReadAllMailsResponseData 全部已读的结果
type ReadAllMailsResponseData struct {
	Read int64 `json:"read"` // 本次标记的封数
}

// ReadAllMails
// @Tags mailbox
// @Summary 把信箱中的邮件全部标记为已读
// @Description 不领取附件
// @Produce json
// @Success 200 {object} Response{data=ReadAllMailsResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/mails/read-all [post]
func ReadAllMails(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	count, err := ReadAll(ctx, user.ID)
	if err != nil {
		logger.ErrorF(ctx, "[ReadAllMails] read failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ReadAllMailsResponseData{Read: count}})
}

// ClaimMail
// @Tags mailbox
// @Summary 领取邮件的附件
// @Description 同时标记为已读; 已下架的物品按回收价折成货币发放。附件中的货币已停用时返回 403, 可以之后再领取
// @Produce json
// @Param id path int true "mail id"
// @Success 200 {object} Response{data=ClaimResult}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/mails/{id}/claim [post]
func ClaimMail(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := mailID(c)
	if !ok {
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	result, err := Claim(ctx, user.ID, id)
	if err != nil {
		switch err.Error() {
		case MailNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case NoAttachments, AlreadyClaimed, MailExpired:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		case AttachmentUnavailable, AttachmentTooLarge:
			c.JSON(http.StatusForbidden, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[ClaimMail] claim failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	logger.InfoF(ctx, "[ClaimMail] %d claimed mail %d", user.ID, id)
	c.JSON(http.StatusOK, Response{Data: result})
}

// ClaimAllMails
// @Tags mailbox
// @Summary 一键领取全部附件
// @Description 从旧到新, 一次最多 100 封, 同时标记为已读; 附件暂时无法领取的跳过, 见 skipped
// @Produce json
// @Success 200 {object} Response{data=ClaimResult}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/mails/claim-all [post]
func ClaimAllMails(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	result, err := ClaimAll(ctx, user.ID)
	if err != nil {
		logger.ErrorF(ctx, "[ClaimAllMails] claim failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if result.Claimed > 0 {
		logger.InfoF(ctx, "[ClaimAllMails] %d claimed %d mails, skipped %d", user.ID, result.Claimed, result.Skipped)
	}
	c.JSON(http.StatusOK, Response{Data: result})
}

// DeleteMail
// @Tags mailbox
// @Summary 删除邮件
// @Description 附件未领取且未过期的邮件不能删除, 返回 409
// @Produce json
// @Param id path int true "mail id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/mails/{id} [delete]
func DeleteMail(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := mailID(c)
	if !ok {
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	if err := Delete(ctx, user.ID, id); err != nil {
		switch err.Error() {
		case MailNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case UnclaimedAttachments:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[DeleteMail] delete failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	c.JSON(http.StatusOK, Response{})
}
