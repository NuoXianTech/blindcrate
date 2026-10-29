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

package leaderboardadmin

import (
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListLeaderboardsRequest 榜单列表查询参数
type ListLeaderboardsRequest struct {
	Current   int    `json:"current" form:"current" binding:"min=1"`
	Size      int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"` // 按代码或名称搜索, 不区分大小写
	IsEnabled *bool  `json:"is_enabled" form:"is_enabled"`
}

// ListLeaderboardsResponseData 榜单列表分页结果
type ListLeaderboardsResponseData struct {
	Total  int64               `json:"total"`
	Boards []leaderboard.Board `json:"boards"`
}

// ListLeaderboards
// @Tags admin
// @Summary 获取榜单列表
// @Description 按排序值排列, 包含停用的榜单
// @Produce json
// @Param request query ListLeaderboardsRequest true "request query"
// @Success 200 {object} Response{data=ListLeaderboardsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/leaderboards [get]
func ListLeaderboards(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListLeaderboardsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, boards, err := QueryLeaderboards(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListLeaderboards] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListLeaderboardsResponseData{Total: total, Boards: boards}})
}

// ListLeaderboardMetricsResponseData 可以排名的指标
type ListLeaderboardMetricsResponseData struct {
	Metrics []stats.Metric `json:"metrics"`
}

// ListLeaderboardMetrics
// @Tags admin
// @Summary 获取可以排名的指标
// @Description 已注册的计数指标和每种货币的净收益
// @Produce json
// @Success 200 {object} Response{data=ListLeaderboardMetricsResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/leaderboards/metrics [get]
func ListLeaderboardMetrics(c *gin.Context) {
	ctx := c.Request.Context()
	metrics, err := leaderboard.QueryMetrics(ctx)
	if err != nil {
		logger.ErrorF(ctx, "[ListLeaderboardMetrics] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListLeaderboardMetricsResponseData{Metrics: metrics}})
}

// LeaderboardRequest 新建或修改榜单请求体
type LeaderboardRequest struct {
	Code        string `json:"code" binding:"required,max=64"` // 自然键, 小写字母、数字和下划线
	Name        string `json:"name" binding:"required,max=64"`
	Description string `json:"description" binding:"max=255"`
	Metric      string `json:"metric" binding:"required,max=64"`              // 排名的指标, 见可以排名的指标
	Scope       string `json:"scope" binding:"required,max=64"`               // 统计范围, 如 all、mode:ranked、venue:dock
	Period      string `json:"period" binding:"required"`                     // total 累计、day 当天或 season 当前赛季
	Size        int32  `json:"size"`                                          // 展示前多少名, 10-500
	MinMatches  int64  `json:"min_matches"`                                   // 同一范围和周期内至少完成多少局才上榜
	SortOrder   int32  `json:"sort_order" binding:"min=-1000000,max=1000000"` // 玩家页面的页签从小到大排列
	IsEnabled   bool   `json:"is_enabled"`                                    // 停用后玩家看不到
}

// leaderboardErrorStatus 榜单配置错误对应的状态码, 不是配置错误时返回 0
func leaderboardErrorStatus(err error) int {
	switch err.Error() {
	case leaderboard.CodeInvalid, leaderboard.NameInvalid, leaderboard.MetricInvalid, leaderboard.ScopeInvalid,
		leaderboard.PeriodInvalid, leaderboard.SizeInvalid, leaderboard.MinMatchesInvalid, leaderboard.RatingBoardInvalid:
		return http.StatusBadRequest
	case leaderboard.BoardNotFound:
		return http.StatusNotFound
	case leaderboard.CodeTaken:
		return http.StatusConflict
	}
	return 0
}

// CreateLeaderboard
// @Tags admin
// @Summary 新建榜单
// @Accept json
// @Produce json
// @Param request body LeaderboardRequest true "request body"
// @Success 200 {object} Response{data=leaderboard.Board}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/leaderboards [post]
func CreateLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()
	var req LeaderboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateLeaderboard(ctx, &req)
	if err != nil {
		if status := leaderboardErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateLeaderboard] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLeaderboardCreate, audit.TargetLeaderboard, created.ID, leaderboardFields(created))
	logger.InfoF(ctx, "[CreateLeaderboard] %d created leaderboard %d %s", operator.ID, created.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateLeaderboard
// @Tags admin
// @Summary 修改榜单
// @Description 需提交全部字段; 修改后缓存的前 N 名随即失效
// @Accept json
// @Produce json
// @Param id path int true "leaderboard id"
// @Param request body LeaderboardRequest true "request body"
// @Success 200 {object} Response{data=leaderboard.Board}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/leaderboards/{id} [put]
func UpdateLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()
	var req LeaderboardRequest
	boardID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateLeaderboard(ctx, boardID, &req)
	if err != nil {
		if status := leaderboardErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateLeaderboard] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLeaderboardUpdate, audit.TargetLeaderboard, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateLeaderboard] %d updated leaderboard %d %s", operator.ID, updated.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteLeaderboard
// @Tags admin
// @Summary 删除榜单
// @Description 只删除榜单配置, 统计数据不受影响
// @Produce json
// @Param id path int true "leaderboard id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/leaderboards/{id} [delete]
func DeleteLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()
	boardID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeleteLeaderboard(ctx, boardID)
	if err != nil {
		if err.Error() == leaderboard.BoardNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteLeaderboard] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLeaderboardDelete, audit.TargetLeaderboard, deleted.ID, leaderboardFields(deleted))
	logger.InfoF(ctx, "[DeleteLeaderboard] %d deleted leaderboard %d %s", operator.ID, deleted.ID, deleted.Code)
	c.JSON(http.StatusOK, Response{})
}
