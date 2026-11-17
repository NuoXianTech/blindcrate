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
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/gin-gonic/gin"
)

// ListVenuesResponseData 地图列表
type ListVenuesResponseData struct {
	Venues []VenueView `json:"venues"`
}

// ListVenues
// @Tags venue
// @Summary 获取启用的地图
// @Description 按排序值排列, 附带当前用户对每个入场条件的达成情况和货仓价值区间; 价值区间为保存地图时的模拟结果
// @Produce json
// @Success 200 {object} reply.Response{data=ListVenuesResponseData}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/venues [get]
func ListVenues(c *gin.Context) {
	ctx := c.Request.Context()
	user := auth.CurrentUser(c)
	venues, err := QueryVenues(ctx, user.ID)
	if err != nil {
		reply.Fail(c, err, "[ListVenues] query failed")
		return
	}
	reply.OK(c, ListVenuesResponseData{Venues: venues})
}

// GetVenue
// @Tags venue
// @Summary 获取地图详情
// @Description 在列表的基础上附带货仓各档位掉落池的公示概率; 掉落池不可用时该档位的 odds 为 null
// @Produce json
// @Param code path string true "venue code"
// @Success 200 {object} reply.Response{data=VenueDetail}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/venues/{code} [get]
func GetVenue(c *gin.Context) {
	ctx := c.Request.Context()
	user := auth.CurrentUser(c)
	detail, err := QueryVenue(ctx, user.ID, c.Param("code"))
	if err != nil {
		reply.Fail(c, err, "[GetVenue] query failed")
		return
	}
	reply.OK(c, detail)
}
