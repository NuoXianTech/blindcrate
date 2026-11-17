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

package checkin

import (
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// GetCheckin
// @Tags checkin
// @Summary 获取签到情况
// @Description 今天是否已签到、连续和累计签到天数、整个周期的奖励和今天对应第几天; 签到关闭时 enabled 为 false
// @Produce json
// @Success 200 {object} reply.Response{data=StatusView}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/checkin [get]
func GetCheckin(c *gin.Context) {
	ctx := c.Request.Context()
	user := auth.CurrentUser(c)
	view, err := QueryStatus(ctx, user.ID)
	if err != nil {
		reply.Fail(c, err, "[GetCheckin] query failed")
		return
	}
	reply.OK(c, view)
}

// DoCheckin
// @Tags checkin
// @Summary 签到
// @Description 每天一次, 按连续签到的天数领取对应一天的奖励; 已下架的物品按回收价折成货币。签到关闭、没有配置奖励或奖励暂时无法发放时返回 403
// @Produce json
// @Success 200 {object} reply.Response{data=CheckinResult}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Param X-Captcha-Token header string false "人机验证令牌, 该场景开启人机验证时必填"
// @Router /api/v1/checkin [post]
func DoCheckin(c *gin.Context) {
	ctx := c.Request.Context()
	user := auth.CurrentUser(c)
	result, err := Checkin(ctx, user.ID)
	if err != nil {
		reply.Fail(c, err, "[DoCheckin] checkin failed")
		return
	}
	logger.InfoF(ctx, "[DoCheckin] %d checked in, streak %d, day %d", user.ID, result.Streak, result.Day)
	reply.OK(c, result)
}
