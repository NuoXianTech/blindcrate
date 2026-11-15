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

package crate

import (
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 crate 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// GetCrate
// @Tags crate
// @Summary 获取箱子和其中可以开出的内容
// @Description 返回当前用户持有的数量和箱子中每一项的概率, 概率与开启时一致; 物品不是箱子时返回 404, 箱子的掉落池不可用时返回 403
// @Produce json
// @Param item_id path int true "item id"
// @Success 200 {object} Response{data=View}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/crates/{item_id} [get]
func GetCrate(c *gin.Context) {
	ctx := c.Request.Context()
	itemID, err := strconv.ParseUint(c.Param("item_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	view, err := Query(ctx, user.ID, itemID)
	if err != nil {
		switch err.Error() {
		case CrateNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case CrateUnavailable:
			c.JSON(http.StatusForbidden, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[GetCrate] query failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	c.JSON(http.StatusOK, Response{Data: view})
}

// OpenCrateRequest 开启请求体
type OpenCrateRequest struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次开启生成一个, 重复提交只开启一次并返回原来的结果
}

// OpenCrate
// @Tags crate
// @Summary 开启背包中的一个箱子
// @Description 扣除 1 个箱子并发放抽中的货币或物品, 全部成功或全部失败; 结果在返回前已经结算, 重复提交返回原来的结果。物品不是箱子时返回 404; 背包中没有这个箱子, 或箱子暂时无法开启时返回 403
// @Accept json
// @Produce json
// @Param item_id path int true "item id"
// @Param request body OpenCrateRequest true "request body"
// @Success 200 {object} Response{data=Result}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/crates/{item_id}/open [post]
func OpenCrate(c *gin.Context) {
	ctx := c.Request.Context()
	itemID, err := strconv.ParseUint(c.Param("item_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	var req OpenCrateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	result, replayed, err := Open(ctx, user.ID, itemID, req.IdempotencyKey)
	if err != nil {
		switch err.Error() {
		case CrateNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case CrateUnavailable, inventory.InsufficientQuantity:
			c.JSON(http.StatusForbidden, Response{ErrorMsg: err.Error()})
		case IdempotencyConflict, DuplicateRequest:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[OpenCrate] open failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	// 重复提交只返回原来的结果, 不重复记录
	if !replayed {
		logger.InfoF(ctx, "[OpenCrate] %d opened crate %d in record %d", user.ID, itemID, result.ID)
	}
	c.JSON(http.StatusOK, Response{Data: result})
}
