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

package redeem

import (
	"net/http"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 redeem 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// RedeemRequest 兑换请求体
type RedeemRequest struct {
	Code string `json:"code" binding:"required,max=64"` // 不区分大小写, 其中的空格和连字符会被忽略
}

// RedeemCode
// @Tags redeem
// @Summary 使用兑换码
// @Description 兑换成功后奖励直接发放, 已下架的物品按回收价折成货币。兑换码不存在或所在的批次已停用时返回 404; 已过期、已被兑换完或自己已兑换过时返回 409; 不满足使用条件、达到每人可兑换的个数或奖励暂时无法发放时返回 403
// @Accept json
// @Produce json
// @Param request body RedeemRequest true "request body"
// @Success 200 {object} Response{data=Result}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/redeem [post]
func RedeemCode(c *gin.Context) {
	ctx := c.Request.Context()
	var req RedeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	result, err := Redeem(ctx, user.ID, req.Code)
	if err != nil {
		reply.Fail(c, err, "[RedeemCode] redeem failed")
		return
	}
	// 兑换码本身不记进日志
	logger.InfoF(ctx, "[RedeemCode] %d redeemed a code of %s", user.ID, result.Name)
	c.JSON(http.StatusOK, Response{Data: result})
}
