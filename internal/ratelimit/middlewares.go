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

package ratelimit

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/otel_trace"
	"github.com/gin-gonic/gin"
)

// ByIP 按客户端 IP 限流的中间件, 客户端 IP 的识别依赖 app.trusted_proxies 配置
func ByIP(scope string, rule Rule) gin.HandlerFunc {
	rule.check()
	return func(c *gin.Context) {
		// init trace
		ctx, span := otel_trace.Start(c.Request.Context(), "RateLimitByIP")
		defer span.End()

		if allowed, wait := Allow(ctx, IPKey(scope, c.ClientIP()), rule); !allowed {
			abortTooManyRequests(c, wait)
			return
		}
		c.Next()
	}
}

// ByUser 按当前登录用户限流的中间件, 需要放在 auth.LoginRequired 之后
func ByUser(scope string, rule Rule) gin.HandlerFunc {
	rule.check()
	return func(c *gin.Context) {
		// init trace
		ctx, span := otel_trace.Start(c.Request.Context(), "RateLimitByUser")
		defer span.End()

		user, ok := auth.GetUserFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error_msg": auth.UnAuthorized, "data": nil})
			return
		}
		if allowed, wait := Allow(ctx, UserKey(scope, user.ID), rule); !allowed {
			abortTooManyRequests(c, wait)
			return
		}
		c.Next()
	}
}

// abortTooManyRequests 返回 429, Retry-After 为还需等待的秒数
func abortTooManyRequests(c *gin.Context, wait time.Duration) {
	c.Header("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error_msg": TooManyRequests, "data": nil})
}
