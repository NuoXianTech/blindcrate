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

package venue

import (
	"net/http"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 venue 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListVenuesResponseData 地图列表
type ListVenuesResponseData struct {
	Venues []VenueView `json:"venues"`
}

// ListVenues
// @Tags venue
// @Summary 获取启用的地图
// @Description 按排序值排列, 附带当前用户对每个入场条件的达成情况和仓库价值区间; 价值区间为保存地图时的模拟结果
// @Produce json
// @Success 200 {object} Response{data=ListVenuesResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/venues [get]
func ListVenues(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	venues, err := QueryVenues(ctx, user.ID)
	if err != nil {
		logger.ErrorF(ctx, "[ListVenues] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListVenuesResponseData{Venues: venues}})
}

// GetVenue
// @Tags venue
// @Summary 获取地图详情
// @Description 在列表的基础上附带仓库各档位掉落池的公示概率; 掉落池不可用时该档位的 odds 为 null
// @Produce json
// @Param code path string true "venue code"
// @Success 200 {object} Response{data=VenueDetail}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/venues/{code} [get]
func GetVenue(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	detail, err := QueryVenue(ctx, user.ID, c.Param("code"))
	if err != nil {
		if err.Error() == VenueNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[GetVenue] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: detail})
}
