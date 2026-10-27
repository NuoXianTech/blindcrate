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

	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

type PublicSettingsResponse struct {
	ErrorMsg string                            `json:"error_msg"`
	Data     map[string]map[string]interface{} `json:"data"` // 设置组 -> 字段 -> 值, 只包含公开字段
}

// PublicSettings 获取公开设置, 如站点名称、是否开放注册, 无需登录
// @Tags settings
// @Produce json
// @Success 200 {object} PublicSettingsResponse
// @Failure 500 {object} PublicSettingsResponse
// @Router /api/v1/settings/public [get]
func PublicSettings(c *gin.Context) {
	data, err := QueryPublic(c.Request.Context())
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PublicSettings] query failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, PublicSettingsResponse{ErrorMsg: QueryFailed})
		return
	}
	c.JSON(http.StatusOK, PublicSettingsResponse{Data: data})
}
