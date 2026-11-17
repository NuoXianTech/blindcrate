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

package auctionadmin

import (
	"github.com/NuoXianTech/blindcrate/internal/apps/auction"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/gin-gonic/gin"
)

// GetAuction
// @Tags admin
// @Summary 获取一局暗拍的详情
// @Description 与玩家看到的公开部分相同, 附带每个座位的带入资金、出价、估值和盈亏
// @Produce json
// @Param match_id path int true "match id"
// @Success 200 {object} reply.Response{data=auction.AdminView}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/auctions/{match_id} [get]
func GetAuction(c *gin.Context) {
	ctx := c.Request.Context()
	matchID, ok := reply.ID(c, "match_id")
	if !ok {
		return
	}
	view, err := auction.QueryAdminAuction(ctx, matchID)
	if err != nil {
		reply.Fail(c, err, "[GetAuction] query failed")
		return
	}
	reply.OK(c, view)
}
