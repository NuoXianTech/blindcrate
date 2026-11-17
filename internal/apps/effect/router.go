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
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// GetItemUse
// @Tags inventory
// @Summary 获取一个可以在背包中使用的物品和使用后可以得到的内容
// @Description 返回当前用户持有的数量、使用方式和可以得到的内容, 按概率得到时带每一项的概率, 与使用时一致; 物品不能在背包中使用时返回 404, 现在不能使用(如箱子的掉落池不可用)时返回 403
// @Produce json
// @Param item_id path int true "item id"
// @Success 200 {object} reply.Response{data=UseView}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/inventory/items/{item_id}/use [get]
func GetItemUse(c *gin.Context) {
	itemID, ok := reply.ID(c, "item_id")
	if !ok {
		return
	}
	user := auth.CurrentUser(c)
	view, err := QueryUse(c.Request.Context(), user.ID, itemID)
	if err != nil {
		reply.Fail(c, err, "[GetItemUse] failed")
		return
	}
	reply.OK(c, view)
}

// UseItem
// @Tags inventory
// @Summary 使用背包中的一个物品
// @Description 扣除 1 个物品, 发放得到的货币和物品并执行效果, 全部成功或全部失败; 结果在返回前已经结算, 重复提交返回原来的结果。物品不能在背包中使用时返回 404; 背包中没有这个物品, 或现在不能使用时返回 403
// @Accept json
// @Produce json
// @Param item_id path int true "item id"
// @Param request body InventoryUseRequest true "request body"
// @Success 200 {object} reply.Response{data=UseResult}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 429 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/inventory/items/{item_id}/use [post]
func UseItem(c *gin.Context) {
	ctx := c.Request.Context()
	itemID, ok := reply.ID(c, "item_id")
	if !ok {
		return
	}
	var req InventoryUseRequest
	if !reply.JSON(c, &req) {
		return
	}
	user := auth.CurrentUser(c)
	result, replayed, err := UseInInventory(ctx, user.ID, itemID, &req)
	if err != nil {
		reply.Fail(c, err, "[UseItem] failed")
		return
	}
	// 重复提交只返回原来的结果, 不重复记录
	if !replayed {
		logger.InfoF(ctx, "[UseItem] %d used item %d in record %d", user.ID, itemID, result.ID)
	}
	reply.OK(c, result)
}
