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

package auditadmin

import (
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/gin-gonic/gin"
)

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
// @Success 200 {object} reply.Response{data=ListAuditLogsResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/audit-logs [get]
func ListAuditLogs(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListAuditLogsRequest{}
	if !reply.Query(c, req) {
		return
	}
	total, logs, err := QueryAuditLogs(ctx, req)
	if err != nil {
		reply.Fail(c, err, "[ListAuditLogs] query failed")
		return
	}
	reply.OK(c, ListAuditLogsResponseData{Total: total, Logs: logs})
}

// ListAuditActionsResponseData 操作类型和名称
type ListAuditActionsResponseData struct {
	Actions []audit.Action `json:"actions"`
}

// ListAuditActions
// @Tags admin
// @Summary 获取操作日志的操作类型和名称
// @Description 用于展示日志和筛选, 按登记顺序
// @Produce json
// @Success 200 {object} reply.Response{data=ListAuditActionsResponseData}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Router /api/v1/admin/audit-logs/actions [get]
func ListAuditActions(c *gin.Context) {
	reply.OK(c, ListAuditActionsResponseData{Actions: audit.Actions()})
}
