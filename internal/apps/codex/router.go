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

package codex

import (
	"net/http"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 codex 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// GetCodex
// @Tags codex
// @Summary 获取图鉴
// @Description 全部启用的藏品, 按品质从高到低排列, 带是否已收集和第一次获得的时间; 以及启用的收集组的进度和领取情况
// @Produce json
// @Success 200 {object} Response{data=CodexData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/codex [get]
func GetCodex(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	data, err := QueryCodex(ctx, user.ID)
	if err != nil {
		logger.ErrorF(ctx, "[GetCodex] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: data})
}

// ClaimCodexSet
// @Tags codex
// @Summary 领取收集组奖励
// @Description 集齐组内全部启用的藏品后领取, 每个收集组只能领取一次; 已下架的物品按回收价折成货币发放。奖励暂时无法发放时返回 403
// @Produce json
// @Param code path string true "set code"
// @Success 200 {object} Response{data=ClaimResult}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/codex/sets/{code}/claim [post]
func ClaimCodexSet(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Param("code")
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	result, err := Claim(ctx, user.ID, code)
	if err != nil {
		switch err.Error() {
		case SetNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case NotCompleted, AlreadyClaimed:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		case RewardUnavailable, RewardTooLarge:
			c.JSON(http.StatusForbidden, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[ClaimCodexSet] claim failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	logger.InfoF(ctx, "[ClaimCodexSet] %d claimed codex set %s", user.ID, code)
	c.JSON(http.StatusOK, Response{Data: result})
}
