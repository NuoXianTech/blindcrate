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

package auction

import (
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 auction 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// GetAuction
// @Tags auction
// @Summary 获取当前用户参与的一局暗拍
// @Description 进行中时只公开已公开品质的藏品和自己的出价与反馈, 结算后公开全部藏品、每个座位的出价和用过的道具; 断线重连后用它恢复, 之后以 ws 推送的 auction:updated 为准。出价通过 ws 消息 auction:bid 提交, 道具通过 auction:use 使用
// @Produce json
// @Param match_id path int true "match id"
// @Success 200 {object} Response{data=AuctionView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/auctions/{match_id} [get]
func GetAuction(c *gin.Context) {
	ctx := c.Request.Context()
	matchID, err := strconv.ParseUint(c.Param("match_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	view, err := QueryAuction(ctx, user.ID, matchID)
	if err != nil {
		if err.Error() == match.MatchNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[GetAuction] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: view})
}

// ListUsableItemsResponseData 这局可以用的道具
type ListUsableItemsResponseData struct {
	Items []UsableItem `json:"items"`
}

// ListUsableItems
// @Tags auction
// @Summary 获取背包中这局可以用的道具
// @Description 效果属于对局场景、地图的道具规则允许且数量大于 0 的物品; 不检查阶段、剩余次数和练习局, 以 self.uses_left 为准。道具通过 ws 消息 auction:use 使用
// @Produce json
// @Param match_id path int true "match id"
// @Success 200 {object} Response{data=ListUsableItemsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/auctions/{match_id}/items [get]
func ListUsableItems(c *gin.Context) {
	ctx := c.Request.Context()
	matchID, err := strconv.ParseUint(c.Param("match_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	items, err := QueryUsableItems(ctx, user.ID, matchID)
	if err != nil {
		if err.Error() == match.MatchNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[ListUsableItems] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListUsableItemsResponseData{Items: items}})
}
