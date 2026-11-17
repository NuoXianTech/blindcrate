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

package ranked

import (
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/gin-gonic/gin"
)

// GetRating
// @Tags ranked
// @Summary 获取当前用户的段位分
// @Description 当前赛季的段位分和段位, 赛季之外为赛季 0 的段位分; 本赛季还没打过排位时为初始分
// @Produce json
// @Success 200 {object} reply.Response{data=View}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/ranked/rating [get]
func GetRating(c *gin.Context) {
	ctx := c.Request.Context()
	user := auth.CurrentUser(c)
	view, err := Query(db.DB(ctx), user.ID, time.Now())
	if err != nil {
		reply.Fail(c, err, "[GetRating] query failed")
		return
	}
	reply.OK(c, view)
}

// ListTiersResponseData 段位列表
type ListTiersResponseData struct {
	Tiers []Tier `json:"tiers"` // 按分数线从低到高
}

// ListTiers
// @Tags ranked
// @Summary 获取段位列表
// @Description 全部段位, 按分数线从低到高; 段位分低于最低的分数线时没有段位
// @Produce json
// @Success 200 {object} reply.Response{data=ListTiersResponseData}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/ranked/tiers [get]
func ListTiers(c *gin.Context) {
	ctx := c.Request.Context()
	tiers, err := QueryTiers(db.DB(ctx))
	if err != nil {
		reply.Fail(c, err, "[ListTiers] query failed")
		return
	}
	reply.OK(c, ListTiersResponseData{Tiers: tiers})
}
