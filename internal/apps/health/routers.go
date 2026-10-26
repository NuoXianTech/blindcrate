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

type HealthResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// Health godoc
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /api/v1/health [get]
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{})
}

type ReadyResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// Ready godoc
// @Tags health
// @Produce json
// @Success 200 {object} ReadyResponse
// @Router /api/v1/ready [get]
func Ready(c *gin.Context) {
	// init
	ctx := c.Request.Context()
	// check postgres
	if config.Config.Database.Enabled {
		sqlDB, err := db.DB(ctx).DB()
		if err != nil {
			logger.ErrorF(c.Request.Context(), "[Ready] postgres check failed: %s", err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, ReadyResponse{})
			return
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			logger.ErrorF(c.Request.Context(), "[Ready] postgres check failed: %s", err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, ReadyResponse{})
			return
		}
	}
	// check redis
	if config.Config.Redis.Enabled {
		if err := db.Redis.Ping(ctx).Err(); err != nil {
			logger.ErrorF(c.Request.Context(), "[Ready] redis check failed: %s", err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, ReadyResponse{})
			return
		}
	}
	// check clickhouse
	// response
	c.JSON(http.StatusOK, ReadyResponse{})
}
