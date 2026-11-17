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

package checkinadmin

import (
	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/checkin"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// DaysData 签到周期中每一天的奖励
type DaysData struct {
	Days    []checkin.Day `json:"days"`     // 按天排列, 从第 1 天开始连续
	MaxDays int           `json:"max_days"` // 最多的天数
}

// ListCheckinDays
// @Tags admin
// @Summary 获取签到奖励
// @Produce json
// @Success 200 {object} reply.Response{data=DaysData}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/checkin/days [get]
func ListCheckinDays(c *gin.Context) {
	ctx := c.Request.Context()
	days, err := checkin.QueryDays(db.DB(ctx))
	if err != nil {
		reply.Fail(c, err, "[ListCheckinDays] query failed")
		return
	}
	reply.OK(c, DaysData{Days: days, MaxDays: checkin.MaxDays})
}

// UpdateDaysRequest 整体保存签到奖励的请求体
type UpdateDaysRequest struct {
	Days []DayRequest `json:"days" binding:"required"` // 第 1 项为第 1 天, 1-31 天
}

// DayRequest 一天的奖励
type DayRequest struct {
	Reward reward.Bundle `json:"reward" binding:"-"` // 数量为正数, 不能为空
}

// UpdateCheckinDays
// @Tags admin
// @Summary 保存签到奖励
// @Description 整体保存, 第 1 项为第 1 天; 天数减少时多出的天数删除。玩家的连续天数不变, 超过新的周期时从第 1 天重新开始
// @Accept json
// @Produce json
// @Param request body UpdateDaysRequest true "request body"
// @Success 200 {object} reply.Response{data=DaysData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/checkin/days [put]
func UpdateCheckinDays(c *gin.Context) {
	ctx := c.Request.Context()
	var req UpdateDaysRequest
	if !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	rewards := make([]reward.Bundle, len(req.Days))
	for i := range req.Days {
		rewards[i] = req.Days[i].Reward
	}
	days, changes, err := doUpdateDays(ctx, rewards)
	if err != nil {
		reply.Fail(c, err, "[UpdateCheckinDays] update failed")
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, checkin.ActionUpdateDays, checkin.TargetCheckinDays, 0, changes)
		logger.InfoF(ctx, "[UpdateCheckinDays] %d updated checkin days to %d days", operator.ID, len(days))
	}
	reply.OK(c, DaysData{Days: days, MaxDays: checkin.MaxDays})
}
