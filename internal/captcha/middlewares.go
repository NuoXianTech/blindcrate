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

package captcha

import (
	"net/http"
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/otel_trace"
	"github.com/gin-gonic/gin"
)

// Required 人机验证的中间件, 场景开启时请求需要带上通过验证的令牌; 建议放在限流中间件之后, 被限流的请求不再请求 Cloudflare
func Required(scene string) gin.HandlerFunc {
	checkScene(scene)
	return func(c *gin.Context) {
		if Check(c, scene) {
			c.Next()
		}
	}
}

// userOf 已登录的用户 ID, 没有登录(登录、注册等场景)时为 0
func userOf(c *gin.Context) uint64 {
	if user, ok := auth.GetUserFromContext(c); ok && user != nil {
		return user.ID
	}
	return 0
}

// Check 请求是否可以继续: 场景没有开启、用户在验证冷却内或令牌通过验证时返回 true; 返回 false 时已经写入响应, 调用方直接返回
//
// 场景要读了请求体才知道时在处理函数中调用, 其他情况用 Required。
func Check(c *gin.Context, scene string) bool {
	checkScene(scene)
	// init trace
	ctx, span := otel_trace.Start(c.Request.Context(), "CaptchaCheck")
	defer span.End()

	s, err := settings.Captcha.Get(ctx)
	if err != nil {
		logger.ErrorF(ctx, "[Captcha] query settings failed: %v", err)
		return true
	}
	if !enabled(s, scene) {
		return true
	}
	userID := userOf(c)
	if passedRecently(ctx, userID, s.Cooldown) {
		return true
	}

	token := strings.TrimSpace(c.GetHeader(TokenHeader))
	if token == "" {
		reply.Abort(c, http.StatusBadRequest, TokenRequired)
		return false
	}
	if len(token) > maxTokenLength {
		reply.Abort(c, http.StatusForbidden, VerifyFailed)
		return false
	}
	// 熔断中不请求 Cloudflare, 直接放行; 没有真正验证过, 不计入冷却
	if paused() {
		return true
	}
	passed, codes, err := verify(ctx, s.SecretKey, token, c.ClientIP())
	if err != nil {
		logger.ErrorF(ctx, "[Captcha] verify %s failed: %v", scene, err)
		return true
	}
	if !passed {
		logger.WarnF(ctx, "[Captcha] verify %s rejected: %v", scene, codes)
		reply.Abort(c, http.StatusForbidden, VerifyFailed)
		return false
	}
	if s.Cooldown > 0 {
		markPassed(ctx, userID)
	}
	return true
}

// Status 当前用户在一个场景中这次要不要人机验证
type Status struct {
	Required bool   `json:"required"` // 场景已开启且不在验证冷却内
	SiteKey  string `json:"site_key"` // Turnstile 的站点密钥, 不需要验证时为空
}

// GetStatus
// @Tags captcha
// @Summary 这次操作要不要人机验证
// @Description 点击签到、购买、领取任务、匹配等操作前调用: 场景已开启且当前用户不在验证冷却内时 required 为 true, 需要先完成验证并在请求头 X-Captcha-Token 中带上令牌。场景为 checkin、shop、mission、match 或 ranked
// @Produce json
// @Param scene path string true "场景"
// @Success 200 {object} reply.Response{data=Status}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/settings/captcha/{scene} [get]
func GetStatus(c *gin.Context) {
	ctx := c.Request.Context()
	scene := c.Param("scene")
	switch scene {
	case SceneCheckin, SceneShop, SceneMission, SceneMatch, SceneRanked:
	default:
		reply.Abort(c, http.StatusBadRequest, SceneInvalid)
		return
	}
	s, err := settings.Captcha.Get(ctx)
	if err != nil {
		reply.Fail(c, err, "[GetCaptchaStatus] query settings failed")
		return
	}
	if !enabled(s, scene) || passedRecently(ctx, userOf(c), s.Cooldown) {
		reply.OK(c, Status{})
		return
	}
	reply.OK(c, Status{Required: true, SiteKey: s.SiteKey})
}
