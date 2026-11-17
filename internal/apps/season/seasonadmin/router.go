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

package seasonadmin

import (
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

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
// @Success 200 {object} reply.Response{data=ListSeasonsResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/seasons [get]
func ListSeasons(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListSeasonsRequest{}
	if !reply.Query(c, req) {
		return
	}
	total, seasons, err := QuerySeasons(ctx, req)
	if err != nil {
		reply.Fail(c, err, "[ListSeasons] query failed")
		return
	}
	reply.OK(c, ListSeasonsResponseData{Total: total, Seasons: seasons})
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

// CreateSeason
// @Tags admin
// @Summary 新建赛季
// @Description 时间不能与其他赛季重叠
// @Accept json
// @Produce json
// @Param request body CreateSeasonRequest true "request body"
// @Success 200 {object} reply.Response{data=season.View}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/seasons [post]
func CreateSeason(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreateSeasonRequest
	if !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	created, err := doCreateSeason(ctx, &req)
	if err != nil {
		reply.Fail(c, err, "[CreateSeason] create failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, season.ActionSeasonCreate, season.TargetSeason, created.ID, seasonFields(&created.Season))
	logger.InfoF(ctx, "[CreateSeason] %d created season %d %s", operator.ID, created.ID, created.Code)
	reply.OK(c, created)
}

// UpdateSeason
// @Tags admin
// @Summary 修改赛季
// @Description 需提交全部字段; 赛季开始后不能修改开始时间, 结束后只能修改名称, 时间不能与其他赛季重叠
// @Accept json
// @Produce json
// @Param id path int true "season id"
// @Param request body SeasonRequest true "request body"
// @Success 200 {object} reply.Response{data=season.View}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/seasons/{id} [put]
func UpdateSeason(c *gin.Context) {
	ctx := c.Request.Context()
	var req SeasonRequest
	seasonID, ok := reply.ID(c, "id")
	if !ok || !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	updated, changes, err := doUpdateSeason(ctx, seasonID, &req)
	if err != nil {
		reply.Fail(c, err, "[UpdateSeason] update failed")
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, season.ActionSeasonUpdate, season.TargetSeason, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateSeason] %d updated season %d %s", operator.ID, updated.ID, updated.Code)
	}
	reply.OK(c, updated)
}

// DeleteSeason
// @Tags admin
// @Summary 删除赛季
// @Description 只能删除未开始的赛季
// @Produce json
// @Param id path int true "season id"
// @Success 200 {object} reply.Response
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/seasons/{id} [delete]
func DeleteSeason(c *gin.Context) {
	ctx := c.Request.Context()
	seasonID, ok := reply.ID(c, "id")
	if !ok {
		return
	}
	operator := auth.CurrentUser(c)
	deleted, err := doDeleteSeason(ctx, seasonID)
	if err != nil {
		reply.Fail(c, err, "[DeleteSeason] delete failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, season.ActionSeasonDelete, season.TargetSeason, deleted.ID, seasonFields(deleted))
	logger.InfoF(ctx, "[DeleteSeason] %d deleted season %d %s", operator.ID, deleted.ID, deleted.Code)
	reply.OK(c, nil)
}

// EndSeason
// @Tags admin
// @Summary 提前结束赛季
// @Description 把进行中的赛季的结束时间改为当前时间, 之后的对局不再计入该赛季
// @Produce json
// @Param id path int true "season id"
// @Success 200 {object} reply.Response{data=season.View}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/seasons/{id}/end [post]
func EndSeason(c *gin.Context) {
	ctx := c.Request.Context()
	seasonID, ok := reply.ID(c, "id")
	if !ok {
		return
	}
	operator := auth.CurrentUser(c)
	ended, changes, err := doEndSeason(ctx, seasonID)
	if err != nil {
		reply.Fail(c, err, "[EndSeason] end failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, season.ActionSeasonEnd, season.TargetSeason, ended.ID, changes)
	logger.InfoF(ctx, "[EndSeason] %d ended season %d %s", operator.ID, ended.ID, ended.Code)
	reply.OK(c, ended)
}
