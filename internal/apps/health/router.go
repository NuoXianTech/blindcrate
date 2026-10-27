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

package health

import (
	"net/http"

	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 health 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// Health
// @Tags health
// @Summary 存活检查
// @Produce json
// @Success 200 {object} Response
// @Router /api/v1/health [get]
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, Response{})
}

// Ready
// @Tags health
// @Summary 就绪检查
// @Description 检查已启用的 PostgreSQL 和 Redis 是否可用
// @Produce json
// @Success 200 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/ready [get]
func Ready(c *gin.Context) {
	ctx := c.Request.Context()
	// check postgres
	if config.Config.Database.Enabled {
		sqlDB, err := db.DB(ctx).DB()
		if err != nil {
			logger.ErrorF(ctx, "[Ready] postgres check failed: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, Response{})
			return
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			logger.ErrorF(ctx, "[Ready] postgres check failed: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, Response{})
			return
		}
	}
	// check redis
	if config.Config.Redis.Enabled {
		if err := db.Redis.Ping(ctx).Err(); err != nil {
			logger.ErrorF(ctx, "[Ready] redis check failed: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, Response{})
			return
		}
	}
	c.JSON(http.StatusOK, Response{})
}
