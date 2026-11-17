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

package rankedadmin

import (
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/auction"
	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/ranked"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// ListRankedTiersResponseData 段位列表
type ListRankedTiersResponseData struct {
	Tiers []ranked.Tier `json:"tiers"` // 按分数线从低到高
}

// ListRankedTiers
// @Tags admin
// @Summary 获取段位列表
// @Description 全部段位, 按分数线从低到高
// @Produce json
// @Success 200 {object} reply.Response{data=ListRankedTiersResponseData}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/ranked/tiers [get]
func ListRankedTiers(c *gin.Context) {
	ctx := c.Request.Context()
	tiers, err := queryRankedTiers(ctx)
	if err != nil {
		reply.Fail(c, err, "[ListRankedTiers] query failed")
		return
	}
	reply.OK(c, ListRankedTiersResponseData{Tiers: tiers})
}

// RankedTierRequest 新建或修改段位请求体
type RankedTierRequest struct {
	Name      string `json:"name" binding:"required,max=32"`
	MinRating int32  `json:"min_rating"`                     // 分数线, 段位分不低于它时属于这个段位, 0-1000000
	Color     string `json:"color" binding:"required,max=7"` // #rrggbb
}

// CreateRankedTier
// @Tags admin
// @Summary 新建段位
// @Accept json
// @Produce json
// @Param request body RankedTierRequest true "request body"
// @Success 200 {object} reply.Response{data=ranked.Tier}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/ranked/tiers [post]
func CreateRankedTier(c *gin.Context) {
	ctx := c.Request.Context()
	var req RankedTierRequest
	if !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	created, err := doCreateRankedTier(ctx, &req)
	if err != nil {
		reply.Fail(c, err, "[CreateRankedTier] create failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, ranked.ActionRankedTierCreate, ranked.TargetRankedTier, created.ID, rankedTierFields(created))
	logger.InfoF(ctx, "[CreateRankedTier] %d created ranked tier %d %s", operator.ID, created.ID, created.Name)
	reply.OK(c, created)
}

// UpdateRankedTier
// @Tags admin
// @Summary 修改段位
// @Description 需提交全部字段; 段位按段位分实时计算, 修改分数线后所有人的段位随即变化
// @Accept json
// @Produce json
// @Param id path int true "ranked tier id"
// @Param request body RankedTierRequest true "request body"
// @Success 200 {object} reply.Response{data=ranked.Tier}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/ranked/tiers/{id} [put]
func UpdateRankedTier(c *gin.Context) {
	ctx := c.Request.Context()
	var req RankedTierRequest
	tierID, ok := reply.ID(c, "id")
	if !ok || !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	updated, changes, err := doUpdateRankedTier(ctx, tierID, &req)
	if err != nil {
		reply.Fail(c, err, "[UpdateRankedTier] update failed")
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, ranked.ActionRankedTierUpdate, ranked.TargetRankedTier, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateRankedTier] %d updated ranked tier %d %s", operator.ID, updated.ID, updated.Name)
	}
	reply.OK(c, updated)
}

// DeleteRankedTier
// @Tags admin
// @Summary 删除段位
// @Description 删除后原来在这个段位的人按剩下的分数线重新计算段位, 段位分不受影响
// @Produce json
// @Param id path int true "ranked tier id"
// @Success 200 {object} reply.Response
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/ranked/tiers/{id} [delete]
func DeleteRankedTier(c *gin.Context) {
	ctx := c.Request.Context()
	tierID, ok := reply.ID(c, "id")
	if !ok {
		return
	}
	operator := auth.CurrentUser(c)
	deleted, err := doDeleteRankedTier(ctx, tierID)
	if err != nil {
		reply.Fail(c, err, "[DeleteRankedTier] delete failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, ranked.ActionRankedTierDelete, ranked.TargetRankedTier, deleted.ID, rankedTierFields(deleted))
	logger.InfoF(ctx, "[DeleteRankedTier] %d deleted ranked tier %d %s", operator.ID, deleted.ID, deleted.Name)
	reply.OK(c, nil)
}

// ListRankedSuspectsRequest 可疑名单查询参数
type ListRankedSuspectsRequest struct {
	Current  int   `json:"current" form:"current" binding:"min=1"`
	Size     int   `json:"size" form:"size" binding:"min=1,max=100"`
	Reviewed *bool `json:"reviewed" form:"reviewed"` // 为空返回全部; false 只返回待处理的, 即没处理过或处理后又一起打过排位; true 只返回其余的
}

// ListRankedSuspectsResponseData 可疑名单分页结果
type ListRankedSuspectsResponseData struct {
	Total    int64             `json:"total"`
	Suspects []auction.Suspect `json:"suspects"`
}

// ListRankedSuspects
// @Tags admin
// @Summary 获取排位的可疑名单
// @Description 最近若干天已结束的排位中, 一起打的局数较多, 或同局时其中一人多次低价成交的两个账号; 天数和次数见排位设置。其中一人低价成交多的排在前面
// @Produce json
// @Param request query ListRankedSuspectsRequest true "request query"
// @Success 200 {object} reply.Response{data=ListRankedSuspectsResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/ranked/suspects [get]
func ListRankedSuspects(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListRankedSuspectsRequest{}
	if !reply.Query(c, req) {
		return
	}
	total, suspects, err := auction.QuerySuspects(ctx, req.Reviewed, (req.Current-1)*req.Size, req.Size)
	if err != nil {
		reply.Fail(c, err, "[ListRankedSuspects] query failed")
		return
	}
	reply.OK(c, ListRankedSuspectsResponseData{Total: total, Suspects: suspects})
}

// ReviewRankedSuspectRequest 处理可疑账号请求体
type ReviewRankedSuspectRequest struct {
	UserA       uint64 `json:"user_a" binding:"required"`
	UserB       uint64 `json:"user_b" binding:"required,gtfield=UserA"` // 大于 user_a
	LastMatchID uint64 `json:"last_match_id" binding:"required"`        // 处理到的对局, 即名单中的 last_match_id
	Note        string `json:"note" binding:"max=255"`                  // 处理结果, 如已封禁、正常组队
}

// ReviewRankedSuspect
// @Tags admin
// @Summary 把一对可疑账号标记为已处理
// @Description 封禁等处理在用户管理中进行; 已处理过时覆盖上一次的记录, 之后两人又一起打了排位时重新进入待处理
// @Accept json
// @Produce json
// @Param request body ReviewRankedSuspectRequest true "request body"
// @Success 200 {object} reply.Response{data=auction.SuspectReview}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/ranked/suspects/review [post]
func ReviewRankedSuspect(c *gin.Context) {
	ctx := c.Request.Context()
	var req ReviewRankedSuspectRequest
	if !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	note := strings.TrimSpace(req.Note)
	review, err := auction.ReviewSuspect(ctx, operator.ID, req.UserA, req.UserB, req.LastMatchID, note)
	if err != nil {
		reply.Fail(c, err, "[ReviewRankedSuspect] review failed")
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, ranked.ActionSuspectReview, ranked.TargetSuspect, req.UserA, map[string]interface{}{
		"user_b":        req.UserB,
		"last_match_id": req.LastMatchID,
		"note":          note,
	})
	logger.InfoF(ctx, "[ReviewRankedSuspect] %d reviewed suspects %d and %d up to match %d", operator.ID, req.UserA, req.UserB, req.LastMatchID)
	reply.OK(c, review)
}
