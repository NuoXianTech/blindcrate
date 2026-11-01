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
	"errors"
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/codex"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

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
// @Success 200 {object} Response{data=ListSetsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/codex/sets [get]
func ListCodexSets(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListSetsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, sets, err := QuerySets(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListCodexSets] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListSetsResponseData{Total: total, Sets: sets}})
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

// setErrorStatus 收集组配置错误对应的状态码, 不是配置错误时返回 0
func setErrorStatus(err error) int {
	var invalid *codex.InvalidError
	switch {
	case errors.As(err, &invalid), err.Error() == codex.CodeImmutable:
		return http.StatusBadRequest
	case err.Error() == codex.SetNotFound:
		return http.StatusNotFound
	case err.Error() == codex.CodeTaken:
		return http.StatusConflict
	}
	return 0
}

// CreateCodexSet
// @Tags admin
// @Summary 新建收集组
// @Accept json
// @Produce json
// @Param request body SetRequest true "request body"
// @Success 200 {object} Response{data=codex.Set}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/codex/sets [post]
func CreateCodexSet(c *gin.Context) {
	ctx := c.Request.Context()
	var req SetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateSet(ctx, &req)
	if err != nil {
		if status := setErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateCodexSet] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, codex.ActionCreate, codex.TargetSet, created.ID, setFields(created))
	logger.InfoF(ctx, "[CreateCodexSet] %d created codex set %d %s", operator.ID, created.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateCodexSet
// @Tags admin
// @Summary 修改收集组
// @Description 需提交全部字段, 代码不能修改; 修改物品后, 已领取的不受影响
// @Accept json
// @Produce json
// @Param id path int true "set id"
// @Param request body SetRequest true "request body"
// @Success 200 {object} Response{data=codex.Set}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/codex/sets/{id} [put]
func UpdateCodexSet(c *gin.Context) {
	ctx := c.Request.Context()
	var req SetRequest
	setID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateSet(ctx, setID, &req)
	if err != nil {
		if status := setErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateCodexSet] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, codex.ActionUpdate, codex.TargetSet, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateCodexSet] %d updated codex set %d %s", operator.ID, updated.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteCodexSet
// @Tags admin
// @Summary 删除收集组
// @Description 软删除, 代码不能再被新收集组使用; 已领取的奖励不收回
// @Produce json
// @Param id path int true "set id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/codex/sets/{id} [delete]
func DeleteCodexSet(c *gin.Context) {
	ctx := c.Request.Context()
	setID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeleteSet(ctx, setID)
	if err != nil {
		if err.Error() == codex.SetNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteCodexSet] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, codex.ActionDelete, codex.TargetSet, deleted.ID, setFields(deleted))
	logger.InfoF(ctx, "[DeleteCodexSet] %d deleted codex set %d %s", operator.ID, deleted.ID, deleted.Code)
	c.JSON(http.StatusOK, Response{})
}
