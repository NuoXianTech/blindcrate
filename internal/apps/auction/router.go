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
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/gin-gonic/gin"
)

// GetAuction
// @Tags auction
// @Summary 获取当前用户参与的一局暗拍
// @Description 进行中时只公开货仓大小、已经扫出的轮廓和自己的出价、反馈与情报, 结算后公开全部藏品、每个座位的出价和用过的道具; 断线重连后用它恢复, 之后以 ws 推送的 auction:updated 为准。出价通过 ws 消息 auction:bid 提交, 道具通过 auction:use 使用
// @Produce json
// @Param match_id path int true "match id"
// @Success 200 {object} reply.Response{data=AuctionView}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/auctions/{match_id} [get]
func GetAuction(c *gin.Context) {
	ctx := c.Request.Context()
	matchID, ok := reply.ID(c, "match_id")
	if !ok {
		return
	}
	user := auth.CurrentUser(c)
	view, err := QueryAuction(ctx, user.ID, matchID)
	if err != nil {
		reply.Fail(c, err, "[GetAuction] query failed")
		return
	}
	reply.OK(c, view)
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
// @Success 200 {object} reply.Response{data=ListUsableItemsResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/auctions/{match_id}/items [get]
func ListUsableItems(c *gin.Context) {
	ctx := c.Request.Context()
	matchID, ok := reply.ID(c, "match_id")
	if !ok {
		return
	}
	user := auth.CurrentUser(c)
	items, err := QueryUsableItems(ctx, user.ID, matchID)
	if err != nil {
		reply.Fail(c, err, "[ListUsableItems] query failed")
		return
	}
	reply.OK(c, ListUsableItemsResponseData{Items: items})
}

// ListCarryResponseData 自己在各地图选的带入金额
type ListCarryResponseData struct {
	Preferences []CarryPreference `json:"preferences"`
}

// ListCarry
// @Tags auction
// @Summary 获取自己在各地图选的带入金额
// @Description 没选过的地图不在列表中, 准入时按地图的带入资金上限; 地图之后调低了上限时按新的上限截断
// @Produce json
// @Success 200 {object} reply.Response{data=ListCarryResponseData}
// @Failure 401 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/auctions/carry [get]
func ListCarry(c *gin.Context) {
	ctx := c.Request.Context()
	user := auth.CurrentUser(c)
	prefs, err := QueryCarry(ctx, user.ID)
	if err != nil {
		reply.Fail(c, err, "[ListCarry] query failed")
		return
	}
	reply.OK(c, ListCarryResponseData{Preferences: prefs})
}

// PutCarry
// @Tags auction
// @Summary 选择在一张地图的带入金额
// @Description 整个仓库共用这笔带入资金, 超出 [起拍价, 带入资金上限] 时按边界保存; 下一局准入时按它冻结, 可用余额不够时拒绝入场
// @Accept json
// @Produce json
// @Param body body CarryRequest true "带入金额"
// @Success 200 {object} reply.Response{data=CarryPreference}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/auctions/carry [put]
func PutCarry(c *gin.Context) {
	ctx := c.Request.Context()
	var req CarryRequest
	if !reply.JSON(c, &req) {
		return
	}
	user := auth.CurrentUser(c)
	pref, err := SetCarry(ctx, user.ID, &req)
	if err != nil {
		reply.Fail(c, err, "[PutCarry] save failed")
		return
	}
	reply.OK(c, pref)
}
