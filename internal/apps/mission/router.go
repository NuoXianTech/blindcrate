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

package mission

import (
	"errors"
	"net/http"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 mission 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListMissions
// @Tags mission
// @Summary 获取任务
// @Description 启用的任务按每日、每周、成就分组, 每项带当前周期的进度和领取情况; 不满足任务条件的也返回, 见 requirements
// @Produce json
// @Success 200 {object} Response{data=MissionsData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/missions [get]
func ListMissions(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	data, err := QueryMissions(ctx, user.ID)
	if err != nil {
		logger.ErrorF(ctx, "[ListMissions] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: data})
}

// ClaimMission
// @Tags mission
// @Summary 领取任务奖励
// @Description 进度达到目标且满足任务条件时领取当前周期的奖励; 已下架的物品按回收价折成货币发放。不满足条件或奖励暂时无法发放时返回 403
// @Produce json
// @Param code path string true "mission code"
// @Success 200 {object} Response{data=ClaimResult}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/missions/{code}/claim [post]
func ClaimMission(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Param("code")
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	result, err := Claim(ctx, user.ID, code)
	if err != nil {
		var rejected *RejectedError
		switch {
		case errors.As(err, &rejected):
			c.JSON(http.StatusForbidden, Response{ErrorMsg: err.Error()})
		case err.Error() == MissionNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case err.Error() == NotCompleted, err.Error() == AlreadyClaimed:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		case err.Error() == RewardUnavailable, err.Error() == RewardTooLarge:
			c.JSON(http.StatusForbidden, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[ClaimMission] claim failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	logger.InfoF(ctx, "[ClaimMission] %d claimed mission %s", user.ID, code)
	c.JSON(http.StatusOK, Response{Data: result})
}
