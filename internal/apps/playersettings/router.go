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
package playersettings

import (
	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/ws"
	"github.com/gin-gonic/gin"
)

type UpdateRequest struct {
	Version            *int64 `json:"version" binding:"required,min=0"`
	KeepOneCollectible *bool  `json:"keep_one_collectible" binding:"required"`
}

// GetSettings
// @Tags player-settings
// @Summary 获取自己的游戏设置
// @Produce json
// @Success 200 {object} reply.Response{data=Settings}
// @Failure 401 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/player-settings [get]
func GetSettings(c *gin.Context) {
	result, err := Read(db.DB(c.Request.Context()), auth.CurrentUser(c).ID)
	if err != nil {
		reply.Fail(c, err, "[PlayerSettings] read failed")
		return
	}
	reply.OK(c, result)
}

// UpdateSettings
// @Tags player-settings
// @Summary 修改自己的游戏设置
// @Description 修改后立即生效; 版本不同返回 409, 不覆盖其他设备刚保存的设置
// @Accept json
// @Produce json
// @Param request body UpdateRequest true "request body"
// @Success 200 {object} reply.Response{data=Settings}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/player-settings [patch]
func UpdateSettings(c *gin.Context) {
	var req UpdateRequest
	if !reply.JSON(c, &req) {
		return
	}
	userID := auth.CurrentUser(c).ID
	result, changed, err := Update(c.Request.Context(), userID, &req)
	if err != nil {
		reply.Fail(c, err, "[PlayerSettings] update failed")
		return
	}
	if changed {
		audit.Record(c, userID, audit.ScopeUser, ActionUpdate, "player_settings", userID, map[string]interface{}{
			"keep_one_collectible": result.KeepOneCollectible, "version": result.Version,
		})
		if err := ws.SendToUser(c.Request.Context(), userID, MessageChanged, result); err != nil {
			logger.ErrorF(c.Request.Context(), "[PlayerSettings] push failed: %v", err)
		}
	}
	reply.OK(c, result)
}
