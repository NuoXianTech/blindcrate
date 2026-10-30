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

package missionadmin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/mission"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListMissionsRequest 任务列表查询参数
type ListMissionsRequest struct {
	Current   int    `json:"current" form:"current" binding:"min=1"`
	Size      int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"` // 按代码或名称搜索, 不区分大小写
	Cycle     string `json:"cycle" form:"cycle" binding:"max=16"`     // 按周期筛选, 为空时不筛选
	IsEnabled *bool  `json:"is_enabled" form:"is_enabled"`
}

// ListMissionsResponseData 任务列表分页结果
type ListMissionsResponseData struct {
	Total    int64             `json:"total"`
	Missions []mission.Mission `json:"missions"`
}

// ListMissions
// @Tags admin
// @Summary 获取任务列表
// @Description 按周期、排序值和 ID 排列, 不含已删除的任务
// @Produce json
// @Param request query ListMissionsRequest true "request query"
// @Success 200 {object} Response{data=ListMissionsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/missions [get]
func ListMissions(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListMissionsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, missions, err := QueryMissions(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListMissions] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListMissionsResponseData{Total: total, Missions: missions}})
}

// ListMissionOptionsResponseData 编辑任务时可以选择的周期和指标
type ListMissionOptionsResponseData struct {
	Cycles  []mission.Cycle `json:"cycles"`
	Metrics []stats.Metric  `json:"metrics"` // 已注册的计数指标和每种货币的净收益
}

// ListMissionOptions
// @Tags admin
// @Summary 获取任务可以选择的周期和指标
// @Produce json
// @Success 200 {object} Response{data=ListMissionOptionsResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/missions/options [get]
func ListMissionOptions(c *gin.Context) {
	ctx := c.Request.Context()
	metrics, err := stats.ValueMetrics(db.DB(ctx))
	if err != nil {
		logger.ErrorF(ctx, "[ListMissionOptions] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListMissionOptionsResponseData{Cycles: mission.Cycles(), Metrics: metrics}})
}

// MissionRequest 新建或修改任务请求体
//
// 奖励和任务条件不在这里校验, 由 mission.Validate 校验, 提示更具体。
type MissionRequest struct {
	Code         string                `json:"code" binding:"required,max=32"` // 小写字母、数字和下划线, 创建后不能修改
	Name         string                `json:"name" binding:"required,max=64"`
	Description  string                `json:"description" binding:"max=255"`
	Cycle        string                `json:"cycle" binding:"required"`  // daily、weekly 或 once
	Scope        string                `json:"scope" binding:"required"`  // 统计范围, 如 all、mode:ranked、venue:dock
	Metric       string                `json:"metric" binding:"required"` // 统计指标, 见可以选择的指标
	Target       int64                 `json:"target"`                    // 1-1000000000000
	Reward       reward.Bundle         `json:"reward" binding:"-"`        // 数量为正数, 不能为空
	Requirements []condition.Condition `json:"requirements" binding:"-"`  // 领取条件, 需要全部满足
	SortOrder    int32                 `json:"sort_order" binding:"min=-1000000,max=1000000"`
	IsEnabled    bool                  `json:"is_enabled"`
}

// missionErrorStatus 任务配置错误对应的状态码, 不是配置错误时返回 0
func missionErrorStatus(err error) int {
	var invalid *mission.InvalidError
	switch {
	case errors.As(err, &invalid), err.Error() == mission.CodeImmutable:
		return http.StatusBadRequest
	case err.Error() == mission.MissionNotFound:
		return http.StatusNotFound
	case err.Error() == mission.CodeTaken:
		return http.StatusConflict
	}
	return 0
}

// CreateMission
// @Tags admin
// @Summary 新建任务
// @Accept json
// @Produce json
// @Param request body MissionRequest true "request body"
// @Success 200 {object} Response{data=mission.Mission}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/missions [post]
func CreateMission(c *gin.Context) {
	ctx := c.Request.Context()
	var req MissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateMission(ctx, &req)
	if err != nil {
		if status := missionErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateMission] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, mission.ActionCreate, mission.TargetMission, created.ID, missionFields(created))
	logger.InfoF(ctx, "[CreateMission] %d created mission %d %s", operator.ID, created.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateMission
// @Tags admin
// @Summary 修改任务
// @Description 需提交全部字段, 代码和周期不能修改; 修改目标后, 本周期已领取的不受影响
// @Accept json
// @Produce json
// @Param id path int true "mission id"
// @Param request body MissionRequest true "request body"
// @Success 200 {object} Response{data=mission.Mission}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/missions/{id} [put]
func UpdateMission(c *gin.Context) {
	ctx := c.Request.Context()
	var req MissionRequest
	missionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateMission(ctx, missionID, &req)
	if err != nil {
		if status := missionErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateMission] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, mission.ActionUpdate, mission.TargetMission, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateMission] %d updated mission %d %s", operator.ID, updated.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteMission
// @Tags admin
// @Summary 删除任务
// @Description 软删除, 代码不能再被新任务使用; 已领取的奖励不收回
// @Produce json
// @Param id path int true "mission id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/missions/{id} [delete]
func DeleteMission(c *gin.Context) {
	ctx := c.Request.Context()
	missionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeleteMission(ctx, missionID)
	if err != nil {
		if err.Error() == mission.MissionNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteMission] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, mission.ActionDelete, mission.TargetMission, deleted.ID, missionFields(deleted))
	logger.InfoF(ctx, "[DeleteMission] %d deleted mission %d %s", operator.ID, deleted.ID, deleted.Code)
	c.JSON(http.StatusOK, Response{})
}
