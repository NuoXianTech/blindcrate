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
	"net/http"
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

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
// @Success 200 {object} Response{data=ListSeasonsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/seasons [get]
func ListSeasons(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListSeasonsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, seasons, err := QuerySeasons(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListSeasons] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListSeasonsResponseData{Total: total, Seasons: seasons}})
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

// seasonErrorStatus 赛季配置错误对应的状态码, 不是配置错误时返回 0
func seasonErrorStatus(err error) int {
	switch err.Error() {
	case season.CodeInvalid, season.NameInvalid, season.TimeInvalid, season.CarryInvalid,
		season.RewardsTooMany, season.RewardRatingInvalid, season.RewardRatingTaken, season.RewardEmpty,
		season.RewardLinesTooMany, season.RewardAmountInvalid, season.RewardNotFound:
		return http.StatusBadRequest
	case season.SeasonNotFound:
		return http.StatusNotFound
	case season.CodeTaken, season.TimeOverlapped, season.StartLocked, season.SeasonEnded, season.DeleteStarted, season.EndNotActive:
		return http.StatusConflict
	}
	return 0
}

// CreateSeason
// @Tags admin
// @Summary 新建赛季
// @Description 时间不能与其他赛季重叠
// @Accept json
// @Produce json
// @Param request body CreateSeasonRequest true "request body"
// @Success 200 {object} Response{data=season.View}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/seasons [post]
func CreateSeason(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreateSeasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateSeason(ctx, &req)
	if err != nil {
		if status := seasonErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateSeason] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionSeasonCreate, audit.TargetSeason, created.ID, seasonFields(&created.Season))
	logger.InfoF(ctx, "[CreateSeason] %d created season %d %s", operator.ID, created.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateSeason
// @Tags admin
// @Summary 修改赛季
// @Description 需提交全部字段; 赛季开始后不能修改开始时间, 结束后只能修改名称, 时间不能与其他赛季重叠
// @Accept json
// @Produce json
// @Param id path int true "season id"
// @Param request body SeasonRequest true "request body"
// @Success 200 {object} Response{data=season.View}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/seasons/{id} [put]
func UpdateSeason(c *gin.Context) {
	ctx := c.Request.Context()
	var req SeasonRequest
	seasonID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateSeason(ctx, seasonID, &req)
	if err != nil {
		if status := seasonErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateSeason] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionSeasonUpdate, audit.TargetSeason, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateSeason] %d updated season %d %s", operator.ID, updated.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteSeason
// @Tags admin
// @Summary 删除赛季
// @Description 只能删除未开始的赛季
// @Produce json
// @Param id path int true "season id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/seasons/{id} [delete]
func DeleteSeason(c *gin.Context) {
	ctx := c.Request.Context()
	seasonID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeleteSeason(ctx, seasonID)
	if err != nil {
		if status := seasonErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteSeason] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionSeasonDelete, audit.TargetSeason, deleted.ID, seasonFields(deleted))
	logger.InfoF(ctx, "[DeleteSeason] %d deleted season %d %s", operator.ID, deleted.ID, deleted.Code)
	c.JSON(http.StatusOK, Response{})
}

// EndSeason
// @Tags admin
// @Summary 提前结束赛季
// @Description 把进行中的赛季的结束时间改为当前时间, 之后的对局不再计入该赛季
// @Produce json
// @Param id path int true "season id"
// @Success 200 {object} Response{data=season.View}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/seasons/{id}/end [post]
func EndSeason(c *gin.Context) {
	ctx := c.Request.Context()
	seasonID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	ended, changes, err := doEndSeason(ctx, seasonID)
	if err != nil {
		if status := seasonErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[EndSeason] end failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionSeasonEnd, audit.TargetSeason, ended.ID, changes)
	logger.InfoF(ctx, "[EndSeason] %d ended season %d %s", operator.ID, ended.ID, ended.Code)
	c.JSON(http.StatusOK, Response{Data: ended})
}
