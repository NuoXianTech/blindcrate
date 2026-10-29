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

package venueadmin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListVenuesRequest 地图列表查询参数
type ListVenuesRequest struct {
	Current   int    `json:"current" form:"current" binding:"min=1"`
	Size      int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"` // 按代码或名称搜索, 不区分大小写
	IsEnabled *bool  `json:"is_enabled" form:"is_enabled"`
}

// ListVenuesResponseData 地图列表分页结果
type ListVenuesResponseData struct {
	Total  int64         `json:"total"`
	Venues []venue.Venue `json:"venues"`
}

// ListVenues
// @Tags admin
// @Summary 获取地图列表
// @Description 按排序值和 ID 排列, 不含已删除的地图
// @Produce json
// @Param request query ListVenuesRequest true "request query"
// @Success 200 {object} Response{data=ListVenuesResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/venues [get]
func ListVenues(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListVenuesRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, venues, err := QueryVenues(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListVenues] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListVenuesResponseData{Total: total, Venues: venues}})
}

// VenueRequest 新建、修改或试算地图请求体
//
// 入场代价、入场条件和各项规则不在这里校验, 由 venue.Validate 校验, 提示更具体。
type VenueRequest struct {
	Code          string                `json:"code" binding:"required,max=24"` // 小写字母、数字和下划线, 创建后不能修改
	Name          string                `json:"name" binding:"required,max=64"`
	Description   string                `json:"description" binding:"max=255"`
	ImageURL      string                `json:"image_url" binding:"omitempty,max=1024,image_url"`
	SortOrder     int32                 `json:"sort_order" binding:"min=-1000000,max=1000000"` // 大厅中从小到大排列
	IsEnabled     bool                  `json:"is_enabled"`
	PveEnabled    bool                  `json:"pve_enabled"`
	RankedEnabled bool                  `json:"ranked_enabled"`
	BidCurrency   string                `json:"bid_currency" binding:"required,max=32"`
	EntryCost     reward.Bundle         `json:"entry_cost" binding:"-"` // 数量为正数, 开局时扣除
	Requirements  []condition.Condition `json:"requirements" binding:"-"`
	Warehouse     venue.WarehouseRule   `json:"warehouse" binding:"-"`
	Auction       venue.AuctionRule     `json:"auction" binding:"-"`
	Bots          venue.BotRule         `json:"bots" binding:"-"`
	Pve           venue.PveRule         `json:"pve" binding:"-"`
	Consumables   venue.ConsumableRule  `json:"consumables" binding:"-"` // 对局中使用道具的规则
}

// CreateVenue
// @Tags admin
// @Summary 新建地图
// @Description 保存前模拟 10000 个仓库, 摆放失败率超过 1% 时拒绝; 模拟结果写入 simulation
// @Accept json
// @Produce json
// @Param request body VenueRequest true "request body"
// @Success 200 {object} Response{data=venue.Venue}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/venues [post]
func CreateVenue(c *gin.Context) {
	ctx := c.Request.Context()
	var req VenueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateVenue(ctx, &req)
	if err != nil {
		var invalid *venue.InvalidError
		switch {
		case errors.As(err, &invalid):
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case err.Error() == venue.CodeTaken:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[CreateVenue] create failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionVenueCreate, audit.TargetVenue, created.ID, venueFields(created))
	logger.InfoF(ctx, "[CreateVenue] %d created venue %d %s", operator.ID, created.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateVenue
// @Tags admin
// @Summary 修改地图
// @Description 需提交全部字段, 代码不能修改; 每次保存都会重新模拟仓库; 进行中的对局使用开局时的快照, 不受影响
// @Accept json
// @Produce json
// @Param id path int true "venue id"
// @Param request body VenueRequest true "request body"
// @Success 200 {object} Response{data=venue.Venue}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/venues/{id} [put]
func UpdateVenue(c *gin.Context) {
	ctx := c.Request.Context()
	var req VenueRequest
	venueID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateVenue(ctx, venueID, &req)
	if err != nil {
		var invalid *venue.InvalidError
		switch {
		case errors.As(err, &invalid), err.Error() == venue.CodeImmutable:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case err.Error() == venue.VenueNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[UpdateVenue] update failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionVenueUpdate, audit.TargetVenue, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateVenue] %d updated venue %d %s", operator.ID, updated.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteVenue
// @Tags admin
// @Summary 删除地图
// @Description 软删除, 代码不能再被新地图使用; 排队中的玩家开局时被拒绝, 进行中的对局照常打完
// @Produce json
// @Param id path int true "venue id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/venues/{id} [delete]
func DeleteVenue(c *gin.Context) {
	ctx := c.Request.Context()
	venueID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeleteVenue(ctx, venueID)
	if err != nil {
		if err.Error() == venue.VenueNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteVenue] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionVenueDelete, audit.TargetVenue, deleted.ID, map[string]interface{}{
		"code": deleted.Code,
		"name": deleted.Name,
	})
	logger.InfoF(ctx, "[DeleteVenue] %d deleted venue %d %s", operator.ID, deleted.ID, deleted.Code)
	c.JSON(http.StatusOK, Response{})
}

// SimulateVenue
// @Tags admin
// @Summary 试算地图
// @Description 按保存时的规则校验并模拟 10000 个仓库, 返回价值分布, 不保存
// @Accept json
// @Produce json
// @Param request body VenueRequest true "request body"
// @Success 200 {object} Response{data=venue.Simulation}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/venues/simulate [post]
func SimulateVenue(c *gin.Context) {
	ctx := c.Request.Context()
	var req VenueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	simulation, err := doSimulateVenue(ctx, &req)
	if err != nil {
		var invalid *venue.InvalidError
		if errors.As(err, &invalid) {
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[SimulateVenue] simulate failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: simulation})
}
