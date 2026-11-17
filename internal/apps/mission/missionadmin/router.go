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
	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/mission"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

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
// @Success 200 {object} reply.Response{data=ListMissionsResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/missions [get]
func ListMissions(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListMissionsRequest{}
	if !reply.Query(c, req) {
		return
	}
	total, missions, err := QueryMissions(ctx, req)
	if err != nil {
		reply.Fail(c, err, "[ListMissions] query failed")
		return
	}
	reply.OK(c, ListMissionsResponseData{Total: total, Missions: missions})
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
// @Success 200 {object} reply.Response{data=ListMissionOptionsResponseData}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/missions/options [get]
func ListMissionOptions(c *gin.Context) {
	ctx := c.Request.Context()
	metrics, err := stats.ValueMetrics(db.DB(ctx))
	if err != nil {
		reply.Fail(c, err, "[ListMissionOptions] query failed")
		return
	}
	reply.OK(c, ListMissionOptionsResponseData{Cycles: mission.Cycles(), Metrics: metrics})
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

// CreateMission
// @Tags admin
// @Summary 新建任务
// @Accept json
// @Produce json
// @Param request body MissionRequest true "request body"
// @Success 200 {object} reply.Response{data=mission.Mission}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/missions [post]
func CreateMission(c *gin.Context) {
	ctx := c.Request.Context()
	var req MissionRequest
	if !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	created, err := doCreateMission(ctx, &req)
	if err != nil {
		reply.Fail(c, err, "[CreateMission] create failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, mission.ActionCreate, mission.TargetMission, created.ID, missionFields(created))
	logger.InfoF(ctx, "[CreateMission] %d created mission %d %s", operator.ID, created.ID, created.Code)
	reply.OK(c, created)
}

// UpdateMission
// @Tags admin
// @Summary 修改任务
// @Description 需提交全部字段, 代码和周期不能修改; 修改目标后, 本周期已领取的不受影响
// @Accept json
// @Produce json
// @Param id path int true "mission id"
// @Param request body MissionRequest true "request body"
// @Success 200 {object} reply.Response{data=mission.Mission}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/missions/{id} [put]
func UpdateMission(c *gin.Context) {
	ctx := c.Request.Context()
	var req MissionRequest
	missionID, ok := reply.ID(c, "id")
	if !ok || !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	updated, changes, err := doUpdateMission(ctx, missionID, &req)
	if err != nil {
		reply.Fail(c, err, "[UpdateMission] update failed")
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, mission.ActionUpdate, mission.TargetMission, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateMission] %d updated mission %d %s", operator.ID, updated.ID, updated.Code)
	}
	reply.OK(c, updated)
}

// DeleteMission
// @Tags admin
// @Summary 删除任务
// @Description 软删除, 代码不能再被新任务使用; 已领取的奖励不收回
// @Produce json
// @Param id path int true "mission id"
// @Success 200 {object} reply.Response
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/missions/{id} [delete]
func DeleteMission(c *gin.Context) {
	ctx := c.Request.Context()
	missionID, ok := reply.ID(c, "id")
	if !ok {
		return
	}
	operator := auth.CurrentUser(c)
	deleted, err := doDeleteMission(ctx, missionID)
	if err != nil {
		reply.Fail(c, err, "[DeleteMission] delete failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, mission.ActionDelete, mission.TargetMission, deleted.ID, missionFields(deleted))
	logger.InfoF(ctx, "[DeleteMission] %d deleted mission %d %s", operator.ID, deleted.ID, deleted.Code)
	reply.OK(c, nil)
}
