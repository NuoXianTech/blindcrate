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

package loot

import (
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/gin-gonic/gin"
)

// GetPoolOdds
// @Tags loot
// @Summary 获取掉落池的公示概率
// @Description 只返回已启用的掉落池中可抽取的条目, 概率为每次抽取抽中该条目的概率
// @Produce json
// @Param code path string true "pool code"
// @Success 200 {object} reply.Response{data=PoolOdds}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/loot-pools/{code}/odds [get]
func GetPoolOdds(c *gin.Context) {
	ctx := c.Request.Context()
	odds, err := QueryOdds(ctx, c.Param("code"))
	if err != nil {
		reply.Fail(c, err, "[GetPoolOdds] query failed")
		return
	}
	reply.OK(c, odds)
}
