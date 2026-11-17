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
	"github.com/NuoXianTech/blindcrate/internal/apps/fault"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

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
// @Success 200 {object} reply.Response{data=ListMailsResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/mails [get]
func ListMails(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListMailsRequest{}
	if !reply.Query(c, req) {
		return
	}
	user := auth.CurrentUser(c)
	total, mails, err := QueryMails(ctx, user, (req.Current-1)*req.Size, req.Size)
	if err != nil {
		reply.Fail(c, err, "[ListMails] query failed")
		return
	}
	reply.OK(c, ListMailsResponseData{Total: total, Mails: mails})
}

// GetCounts
// @Tags mailbox
// @Summary 获取未读和附件未领取的封数
// @Description 会先把可以收到的全服邮件放进信箱; 收到 mailbox:new 推送或切回页面时调用
// @Produce json
// @Success 200 {object} reply.Response{data=Counts}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/mails/counts [get]
func GetCounts(c *gin.Context) {
	ctx := c.Request.Context()
	user := auth.CurrentUser(c)
	counts, err := QueryCounts(ctx, user)
	if err != nil {
		reply.Fail(c, err, "[GetMailCounts] query failed")
		return
	}
	reply.OK(c, counts)
}

// mailID 解析路径中的邮件 ID, 不合法时返回 false 并写入 400
func mailID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		reply.Error(c, http.StatusBadRequest, fault.InvalidParams)
		return 0, false
	}
	return id, true
}

// ReadMail
// @Tags mailbox
// @Summary 标记邮件为已读
// @Produce json
// @Param id path int true "mail id"
// @Success 200 {object} reply.Response
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/mails/{id}/read [post]
func ReadMail(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := mailID(c)
	if !ok {
		return
	}
	user := auth.CurrentUser(c)
	if err := Read(ctx, user.ID, id); err != nil {
		reply.Fail(c, err, "[ReadMail] read failed")
		return
	}
	reply.OK(c, nil)
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
// @Success 200 {object} reply.Response{data=ReadAllMailsResponseData}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/mails/read-all [post]
func ReadAllMails(c *gin.Context) {
	ctx := c.Request.Context()
	user := auth.CurrentUser(c)
	count, err := ReadAll(ctx, user.ID)
	if err != nil {
		reply.Fail(c, err, "[ReadAllMails] read failed")
		return
	}
	reply.OK(c, ReadAllMailsResponseData{Read: count})
}

// ClaimMail
// @Tags mailbox
// @Summary 领取邮件的附件
// @Description 同时标记为已读; 已下架的物品按回收价折成货币发放。附件中的货币已停用时返回 403, 可以之后再领取
// @Produce json
// @Param id path int true "mail id"
// @Success 200 {object} reply.Response{data=ClaimResult}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/mails/{id}/claim [post]
func ClaimMail(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := mailID(c)
	if !ok {
		return
	}
	user := auth.CurrentUser(c)
	result, err := Claim(ctx, user.ID, id)
	if err != nil {
		reply.Fail(c, err, "[ClaimMail] claim failed")
		return
	}
	logger.InfoF(ctx, "[ClaimMail] %d claimed mail %d", user.ID, id)
	reply.OK(c, result)
}

// ClaimAllMails
// @Tags mailbox
// @Summary 一键领取全部附件
// @Description 从旧到新, 一次最多 100 封, 同时标记为已读; 附件暂时无法领取的跳过, 见 skipped
// @Produce json
// @Success 200 {object} reply.Response{data=ClaimResult}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/mails/claim-all [post]
func ClaimAllMails(c *gin.Context) {
	ctx := c.Request.Context()
	user := auth.CurrentUser(c)
	result, err := ClaimAll(ctx, user.ID)
	if err != nil {
		reply.Fail(c, err, "[ClaimAllMails] claim failed")
		return
	}
	if result.Claimed > 0 {
		logger.InfoF(ctx, "[ClaimAllMails] %d claimed %d mails, skipped %d", user.ID, result.Claimed, result.Skipped)
	}
	reply.OK(c, result)
}

// DeleteMail
// @Tags mailbox
// @Summary 删除邮件
// @Description 附件未领取且未过期的邮件不能删除, 返回 409
// @Produce json
// @Param id path int true "mail id"
// @Success 200 {object} reply.Response
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/mails/{id} [delete]
func DeleteMail(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := mailID(c)
	if !ok {
		return
	}
	user := auth.CurrentUser(c)
	if err := Delete(ctx, user.ID, id); err != nil {
		reply.Fail(c, err, "[DeleteMail] delete failed")
		return
	}
	reply.OK(c, nil)
}
