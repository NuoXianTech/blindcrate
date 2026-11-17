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

package codexadmin

import (
	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/codex"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// ListSetsRequest 收集组列表查询参数
type ListSetsRequest struct {
	Current   int    `json:"current" form:"current" binding:"min=1"`
	Size      int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"` // 按代码或名称搜索, 不区分大小写
	IsEnabled *bool  `json:"is_enabled" form:"is_enabled"`
}

// ListSetsResponseData 收集组列表分页结果
type ListSetsResponseData struct {
	Total int64       `json:"total"`
	Sets  []codex.Set `json:"sets"`
}

// ListCodexSets
// @Tags admin
// @Summary 获取收集组列表
// @Description 按排序值和 ID 排列, 不含已删除的收集组
// @Produce json
// @Param request query ListSetsRequest true "request query"
// @Success 200 {object} reply.Response{data=ListSetsResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/codex/sets [get]
func ListCodexSets(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListSetsRequest{}
	if !reply.Query(c, req) {
		return
	}
	total, sets, err := QuerySets(ctx, req)
	if err != nil {
		reply.Fail(c, err, "[ListCodexSets] query failed")
		return
	}
	reply.OK(c, ListSetsResponseData{Total: total, Sets: sets})
}

// SetRequest 新建或修改收集组请求体
//
// 物品和奖励不在这里校验, 由 codex.Validate 校验, 提示更具体。
type SetRequest struct {
	Code        string        `json:"code" binding:"required,max=32"` // 小写字母、数字和下划线, 创建后不能修改
	Name        string        `json:"name" binding:"required,max=64"`
	Description string        `json:"description" binding:"max=255"`
	ItemIDs     []uint64      `json:"item_ids" binding:"-"` // 未删除的藏品, 1-200 种, 重复的只算一次
	Reward      reward.Bundle `json:"reward" binding:"-"`   // 数量为正数, 不能为空
	SortOrder   int32         `json:"sort_order" binding:"min=-1000000,max=1000000"`
	IsEnabled   bool          `json:"is_enabled"`
}

// CreateCodexSet
// @Tags admin
// @Summary 新建收集组
// @Accept json
// @Produce json
// @Param request body SetRequest true "request body"
// @Success 200 {object} reply.Response{data=codex.Set}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/codex/sets [post]
func CreateCodexSet(c *gin.Context) {
	ctx := c.Request.Context()
	var req SetRequest
	if !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	created, err := doCreateSet(ctx, &req)
	if err != nil {
		reply.Fail(c, err, "[CreateCodexSet] create failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, codex.ActionCreate, codex.TargetSet, created.ID, setFields(created))
	logger.InfoF(ctx, "[CreateCodexSet] %d created codex set %d %s", operator.ID, created.ID, created.Code)
	reply.OK(c, created)
}

// UpdateCodexSet
// @Tags admin
// @Summary 修改收集组
// @Description 需提交全部字段, 代码不能修改; 修改物品后, 已领取的不受影响
// @Accept json
// @Produce json
// @Param id path int true "set id"
// @Param request body SetRequest true "request body"
// @Success 200 {object} reply.Response{data=codex.Set}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/codex/sets/{id} [put]
func UpdateCodexSet(c *gin.Context) {
	ctx := c.Request.Context()
	var req SetRequest
	setID, ok := reply.ID(c, "id")
	if !ok || !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	updated, changes, err := doUpdateSet(ctx, setID, &req)
	if err != nil {
		reply.Fail(c, err, "[UpdateCodexSet] update failed")
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, codex.ActionUpdate, codex.TargetSet, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateCodexSet] %d updated codex set %d %s", operator.ID, updated.ID, updated.Code)
	}
	reply.OK(c, updated)
}

// DeleteCodexSet
// @Tags admin
// @Summary 删除收集组
// @Description 软删除, 代码不能再被新收集组使用; 已领取的奖励不收回
// @Produce json
// @Param id path int true "set id"
// @Success 200 {object} reply.Response
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/codex/sets/{id} [delete]
func DeleteCodexSet(c *gin.Context) {
	ctx := c.Request.Context()
	setID, ok := reply.ID(c, "id")
	if !ok {
		return
	}
	operator := auth.CurrentUser(c)
	deleted, err := doDeleteSet(ctx, setID)
	if err != nil {
		reply.Fail(c, err, "[DeleteCodexSet] delete failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, codex.ActionDelete, codex.TargetSet, deleted.ID, setFields(deleted))
	logger.InfoF(ctx, "[DeleteCodexSet] %d deleted codex set %d %s", operator.ID, deleted.ID, deleted.Code)
	reply.OK(c, nil)
}
