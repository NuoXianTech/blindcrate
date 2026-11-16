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

package effect

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 effect 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// respondUseError 把查看和使用的错误转成响应
func respondUseError(c *gin.Context, name string, err error) {
	var unusable *UseError
	if errors.As(err, &unusable) {
		c.JSON(http.StatusForbidden, Response{ErrorMsg: unusable.Reason})
		return
	}
	switch err.Error() {
	case ItemNotUsable:
		c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
	case inventory.InsufficientQuantity:
		c.JSON(http.StatusForbidden, Response{ErrorMsg: err.Error()})
	case ChoiceInvalid, InputInvalid:
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
	case IdempotencyConflict, DuplicateRequest:
		c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
	default:
		logger.ErrorF(c.Request.Context(), "[%s] failed: %v", name, err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
	}
}

// GetItemUse
// @Tags inventory
// @Summary 获取一个可以在背包中使用的物品和使用后可以得到的内容
// @Description 返回当前用户持有的数量、使用方式和可以得到的内容, 按概率得到时带每一项的概率, 与使用时一致; 物品不能在背包中使用时返回 404, 现在不能使用(如箱子的掉落池不可用)时返回 403
// @Produce json
// @Param item_id path int true "item id"
// @Success 200 {object} Response{data=UseView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/inventory/items/{item_id}/use [get]
func GetItemUse(c *gin.Context) {
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
	view, err := QueryUse(c.Request.Context(), user.ID, itemID)
	if err != nil {
		respondUseError(c, "GetItemUse", err)
		return
	}
	c.JSON(http.StatusOK, Response{Data: view})
}

// UseItem
// @Tags inventory
// @Summary 使用背包中的一个物品
// @Description 扣除 1 个物品, 发放得到的货币和物品并执行效果, 全部成功或全部失败; 结果在返回前已经结算, 重复提交返回原来的结果。物品不能在背包中使用时返回 404; 背包中没有这个物品, 或现在不能使用时返回 403
// @Accept json
// @Produce json
// @Param item_id path int true "item id"
// @Param request body InventoryUseRequest true "request body"
// @Success 200 {object} Response{data=UseResult}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/inventory/items/{item_id}/use [post]
func UseItem(c *gin.Context) {
	ctx := c.Request.Context()
	itemID, err := strconv.ParseUint(c.Param("item_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	var req InventoryUseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	result, replayed, err := UseInInventory(ctx, user.ID, itemID, &req)
	if err != nil {
		respondUseError(c, "UseItem", err)
		return
	}
	// 重复提交只返回原来的结果, 不重复记录
	if !replayed {
		logger.InfoF(ctx, "[UseItem] %d used item %d in record %d", user.ID, itemID, result.ID)
	}
	c.JSON(http.StatusOK, Response{Data: result})
}
