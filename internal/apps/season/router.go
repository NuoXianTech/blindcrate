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
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/gin-gonic/gin"
)

// GetCurrentResponseData 当前和下一个赛季
type GetCurrentResponseData struct {
	Current *View        `json:"current"` // 进行中的赛季, 没有时为空
	Next    *View        `json:"next"`    // 下一个未开始的赛季, 没有时为空
	Rewards []RewardView `json:"rewards"` // 进行中的赛季的奖励, 带货币和物品的名称; 没有进行中的赛季时为空列表
}

// GetCurrent
// @Tags season
// @Summary 获取当前赛季
// @Description 进行中的赛季和下一个未开始的赛季, 都可能为空; 带进行中的赛季的奖励
// @Produce json
// @Success 200 {object} reply.Response{data=GetCurrentResponseData}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/seasons/current [get]
func GetCurrent(c *gin.Context) {
	ctx := c.Request.Context()
	tx := db.DB(ctx)
	now := time.Now()
	current, err := Current(tx, now)
	if err != nil {
		reply.Fail(c, err, "[GetCurrentSeason] query current failed")
		return
	}
	next, err := Next(tx, now)
	if err != nil {
		reply.Fail(c, err, "[GetCurrentSeason] query next failed")
		return
	}
	data := GetCurrentResponseData{Rewards: []RewardView{}}
	if current != nil {
		view := NewView(current, now)
		data.Current = &view
		if data.Rewards, err = ResolveRewards(tx, current.Rewards); err != nil {
			reply.Fail(c, err, "[GetCurrentSeason] resolve rewards failed")
			return
		}
	}
	if next != nil {
		view := NewView(next, now)
		data.Next = &view
	}
	reply.OK(c, data)
}
