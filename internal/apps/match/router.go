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

package match

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的 match 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// JoinMatchmakingRequest 加入匹配请求体
type JoinMatchmakingRequest struct {
	Mode string `json:"mode" binding:"required,max=32"`
}

// JoinMatchmaking
// @Tags match
// @Summary 加入匹配队列
// @Description 凑够一局时立即开局, 此时返回的 match_id 不为 0, 并推送 match:updated; 否则排队等待, 玩法允许人机补位时, 排队人数达到玩法的最少真人数且等到补位时间后用人机补齐。已在同一个队列中时不重复加入。不满足玩法的条件(如余额不足)时返回 403; 开局时不满足条件会被移出队列并推送 match:matchmaking_rejected
// @Accept json
// @Produce json
// @Param request body JoinMatchmakingRequest true "request body"
// @Success 200 {object} Response{data=MatchmakingStatus}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/matchmaking [post]
func JoinMatchmaking(c *gin.Context) {
	ctx := c.Request.Context()
	var req JoinMatchmakingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	status, err := Join(ctx, user.ID, req.Mode)
	if err != nil {
		var rejected *RejectedError
		if errors.As(err, &rejected) {
			c.JSON(http.StatusForbidden, Response{ErrorMsg: rejected.Reason})
			return
		}
		switch err.Error() {
		case ModeNotFound:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case AlreadyQueued, InMatch:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[JoinMatchmaking] join failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	c.JSON(http.StatusOK, Response{Data: status})
}

// LeaveMatchmaking
// @Tags match
// @Summary 离开匹配队列
// @Description 不在队列中时什么都不做; 已经开局时无法离开
// @Produce json
// @Success 200 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/matchmaking [delete]
func LeaveMatchmaking(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	if err := Leave(ctx, user.ID); err != nil {
		logger.ErrorF(ctx, "[LeaveMatchmaking] leave failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{})
}

// GetMatchmaking
// @Tags match
// @Summary 获取当前用户的匹配状态
// @Produce json
// @Success 200 {object} Response{data=MatchmakingStatus}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/matchmaking [get]
func GetMatchmaking(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	status, err := QueryMatchmakingStatus(ctx, user.ID)
	if err != nil {
		logger.ErrorF(ctx, "[GetMatchmaking] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: status})
}

// GetCurrentMatch
// @Tags match
// @Summary 获取当前用户正在进行中的对局
// @Description 没有进行中的对局时 data 为 null; 已释放(玩家已结算、可以离开)的对局不算; 断线重连后用它恢复对局
// @Produce json
// @Success 200 {object} Response{data=MatchView}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/matches/current [get]
func GetCurrentMatch(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	view, err := QueryCurrentMatch(ctx, user.ID)
	if err != nil {
		logger.ErrorF(ctx, "[GetCurrentMatch] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: view})
}

// ListMatchesRequest 对局记录查询参数
type ListMatchesRequest struct {
	Current int    `json:"current" form:"current" binding:"min=1"`
	Size    int    `json:"size" form:"size" binding:"min=1,max=100"`
	Status  string `json:"status" form:"status" binding:"omitempty,oneof=playing finished cancelled"`
}

// ListMatchesResponseData 对局记录分页结果
type ListMatchesResponseData struct {
	Total   int64       `json:"total"`
	Matches []MatchView `json:"matches"`
}

// ListMatches
// @Tags match
// @Summary 获取当前用户参与过的对局
// @Produce json
// @Param request query ListMatchesRequest true "request query"
// @Success 200 {object} Response{data=ListMatchesResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/matches [get]
func ListMatches(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListMatchesRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	total, matches, err := QueryUserMatches(ctx, user.ID, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListMatches] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListMatchesResponseData{Total: total, Matches: matches}})
}

// GetMatch
// @Tags match
// @Summary 获取当前用户参与过的一局对局
// @Produce json
// @Param id path int true "match id"
// @Success 200 {object} Response{data=MatchView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/matches/{id} [get]
func GetMatch(c *gin.Context) {
	ctx := c.Request.Context()
	matchID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	view, err := QueryUserMatch(ctx, user.ID, matchID)
	if err != nil {
		if err.Error() == MatchNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[GetMatch] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: view})
}
