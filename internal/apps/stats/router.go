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

package stats

import (
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/gin-gonic/gin"
)

// GetStatsRequest 个人数据查询参数
type GetStatsRequest struct {
	Scope string `json:"scope" form:"scope" binding:"max=64"` // 统计范围, 如 all、mode:pve、venue:dock, 默认 all
}

// GetStats
// @Tags stats
// @Summary 获取当前用户的个人数据
// @Description 返回一个统计范围的累计数据和最近 7 天每天的数据; 只能查看自己的数据, 练习局不计入
// @Produce json
// @Param request query GetStatsRequest true "request query"
// @Success 200 {object} reply.Response{data=UserStats}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/stats [get]
func GetStats(c *gin.Context) {
	ctx := c.Request.Context()
	req := &GetStatsRequest{}
	if !reply.Query(c, req) {
		return
	}
	if req.Scope == "" {
		req.Scope = ScopeAll
	}
	user := auth.CurrentUser(c)
	result, err := QueryUserStats(db.DB(ctx), user.ID, req.Scope, time.Now())
	if err != nil {
		reply.Fail(c, err, "[GetStats] query failed")
		return
	}
	reply.OK(c, result)
}
