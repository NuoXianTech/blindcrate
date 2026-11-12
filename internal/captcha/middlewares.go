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

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/otel_trace"
	"github.com/gin-gonic/gin"
)

// Required 人机验证的中间件, 场景开启时请求需要带上通过验证的令牌; 建议放在限流中间件之后, 被限流的请求不再请求 Cloudflare
func Required(scene string) gin.HandlerFunc {
	checkScene(scene)
	return func(c *gin.Context) {
		// init trace
		ctx, span := otel_trace.Start(c.Request.Context(), "CaptchaRequired")
		defer span.End()

		s, err := settings.Captcha.Get(ctx)
		if err != nil {
			logger.ErrorF(ctx, "[Captcha] query settings failed: %v", err)
			c.Next()
			return
		}
		if !enabled(s, scene) {
			c.Next()
			return
		}

		token := strings.TrimSpace(c.GetHeader(TokenHeader))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error_msg": TokenRequired, "data": nil})
			return
		}
		if len(token) > maxTokenLength {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error_msg": VerifyFailed, "data": nil})
			return
		}
		passed, codes, err := verify(ctx, s.SecretKey, token, c.ClientIP())
		if err != nil {
			logger.ErrorF(ctx, "[Captcha] verify %s failed: %v", scene, err)
			c.Next()
			return
		}
		if !passed {
			logger.WarnF(ctx, "[Captcha] verify %s rejected: %v", scene, codes)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error_msg": VerifyFailed, "data": nil})
			return
		}
		c.Next()
	}
}
