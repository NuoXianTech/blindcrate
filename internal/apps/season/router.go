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

package season

import (
	"net/http"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 season 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// GetCurrentResponseData 当前和下一个赛季
type GetCurrentResponseData struct {
	Current *View `json:"current"` // 进行中的赛季, 没有时为空
	Next    *View `json:"next"`    // 下一个未开始的赛季, 没有时为空
}

// GetCurrent
// @Tags season
// @Summary 获取当前赛季
// @Description 进行中的赛季和下一个未开始的赛季, 都可能为空
// @Produce json
// @Success 200 {object} Response{data=GetCurrentResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/seasons/current [get]
func GetCurrent(c *gin.Context) {
	ctx := c.Request.Context()
	tx := db.DB(ctx)
	now := time.Now()
	current, err := Current(tx, now)
	if err != nil {
		logger.ErrorF(ctx, "[GetCurrentSeason] query current failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	next, err := Next(tx, now)
	if err != nil {
		logger.ErrorF(ctx, "[GetCurrentSeason] query next failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	data := GetCurrentResponseData{}
	if current != nil {
		view := NewView(current, now)
		data.Current = &view
	}
	if next != nil {
		view := NewView(next, now)
		data.Next = &view
	}
	c.JSON(http.StatusOK, Response{Data: data})
}
