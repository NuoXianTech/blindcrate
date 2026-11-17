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

package settings

import (
	"net/http"

	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// PublicSettings
// @Tags settings
// @Summary 获取公开设置
// @Description 如站点名称、是否开放注册, 无需登录; 返回 设置组 -> 字段 -> 值, 只包含公开字段
// @Produce json
// @Success 200 {object} reply.Response{data=map[string]object}
// @Failure 500 {object} reply.Response
// @Router /api/v1/settings/public [get]
func PublicSettings(c *gin.Context) {
	ctx := c.Request.Context()
	data, err := QueryPublic(ctx)
	if err != nil {
		logger.ErrorF(ctx, "[PublicSettings] query failed: %v", err)
		reply.Error(c, http.StatusInternalServerError, QueryFailed)
		return
	}
	reply.OK(c, data)
}
