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

package matchadmin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListMatchesRequest 对局列表查询参数
type ListMatchesRequest struct {
	Current int     `json:"current" form:"current" binding:"min=1"`
	Size    int     `json:"size" form:"size" binding:"min=1,max=100"`
	Prefix  string  `json:"prefix" form:"prefix" binding:"max=32"` // 玩法代码冒号前的部分, 如 pve
	Suffix  string  `json:"suffix" form:"suffix" binding:"max=32"` // 玩法代码冒号后的部分, 如地图代码
	Status  string  `json:"status" form:"status" binding:"omitempty,oneof=playing finished cancelled"`
	UserID  *uint64 `json:"user_id" form:"user_id"`   // 只看该用户参与的对局
	OtherID *uint64 `json:"other_id" form:"other_id"` // 只看该用户也参与的对局, 与 user_id 一起用于查看两个人一起打的对局
}

// ListMatchesResponseData 对局列表分页结果
type ListMatchesResponseData struct {
	Total   int64               `json:"total"`
	Matches []match.MatchDetail `json:"matches"`
}

// ListMatches
// @Tags admin
// @Summary 获取对局列表
// @Description 包含每个座位是真人还是人机, 按开始时间从新到旧
// @Produce json
// @Param request query ListMatchesRequest true "request query"
// @Success 200 {object} Response{data=ListMatchesResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/matches [get]
func ListMatches(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListMatchesRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, matches, err := QueryMatches(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListMatches] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListMatchesResponseData{Total: total, Matches: matches}})
}

// CancelMatchRequest 取消对局请求体
type CancelMatchRequest struct {
	Reason string `json:"reason" binding:"required,max=255"` // 记录在操作日志中
}

// CancelMatch
// @Tags admin
// @Summary 取消进行中的对局
// @Description 用于对局卡住等异常情况; 玩法会退还入场费等, 玩法已下线时需要手动处理
// @Accept json
// @Produce json
// @Param id path int true "match id"
// @Param request body CancelMatchRequest true "request body"
// @Success 200 {object} Response{data=match.Match}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/matches/{id}/cancel [post]
func CancelMatch(c *gin.Context) {
	ctx := c.Request.Context()
	var req CancelMatchRequest
	matchID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.Reason) == "" {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	cancelled, err := match.Cancel(ctx, matchID)
	if err != nil {
		switch err.Error() {
		case match.MatchNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case match.MatchNotPlaying:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[CancelMatch] cancel failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionMatchCancel, audit.TargetMatch, cancelled.ID, map[string]interface{}{
		"mode":   cancelled.Mode,
		"phase":  cancelled.Phase,
		"round":  cancelled.Round,
		"reason": strings.TrimSpace(req.Reason),
	})
	logger.InfoF(ctx, "[CancelMatch] %d cancelled match %d %s", operator.ID, cancelled.ID, cancelled.Mode)
	c.JSON(http.StatusOK, Response{Data: cancelled})
}
