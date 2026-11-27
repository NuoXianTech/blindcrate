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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/redis/go-redis/v9"
)

// client 请求 Cloudflare 用的客户端
var client = &http.Client{Timeout: verifyTimeout}

// breaker 连不上 Cloudflare 时的熔断
var breaker struct {
	sync.Mutex
	failures int       // 连续没有连上的次数
	until    time.Time // 熔断到什么时候
}

// verifyResponse Turnstile 验证接口的响应
type verifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// checkScene 场景不存在时 panic, 在注册路由时调用
func checkScene(scene string) {
	switch scene {
	case SceneLogin, SceneRegister, SceneCheckin, SceneShop, SceneMission, SceneMatch, SceneRanked:
	default:
		panic(fmt.Sprintf("captcha: unknown scene %q", scene))
	}
}

// enabled 场景是否需要人机验证, 没有填写站点密钥或私钥时视为关闭
func enabled(s *settings.CaptchaSettings, scene string) bool {
	if s.SiteKey == "" || s.SecretKey == "" {
		return false
	}
	switch scene {
	case SceneLogin:
		return s.LoginEnabled
	case SceneRegister:
		return s.RegisterEnabled
	case SceneCheckin:
		return s.CheckinEnabled
	case SceneShop:
		return s.ShopEnabled
	case SceneMission:
		return s.MissionEnabled
	case SceneMatch:
		return s.MatchEnabled
	case SceneRanked:
		return s.RankedEnabled
	}
	return false
}

// paused 是否在熔断中, 熔断中的请求不验证; 到期后只让一个请求去试探, 其余的仍然不验证
func paused() bool {
	breaker.Lock()
	defer breaker.Unlock()
	if breaker.failures < breakerThreshold {
		return false
	}
	now := time.Now()
	if now.Before(breaker.until) {
		return true
	}
	breaker.until = now.Add(breakerCooldown)
	return false
}

// record 记录一次请求有没有连上 Cloudflare, 返回是否进入熔断
func record(reached bool) bool {
	breaker.Lock()
	defer breaker.Unlock()
	if reached {
		breaker.failures = 0
		return false
	}
	breaker.failures++
	if breaker.failures < breakerThreshold {
		return false
	}
	breaker.until = time.Now().Add(breakerCooldown)
	return true
}

// verify 向 Cloudflare 验证令牌, 返回是否通过和没有通过时的错误码; 请求失败、响应无法解析或私钥不被接受时返回 error
func verify(ctx context.Context, secret, token, ip string) (bool, []string, error) {
	form := url.Values{"secret": {secret}, "response": {token}}
	if ip != "" {
		form.Set("remoteip", ip)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		// 请求方自己断开连接时不算连不上
		if ctx.Err() == nil && record(false) {
			err = fmt.Errorf("%w; verification paused for %s", err, breakerCooldown)
		}
		return false, nil, err
	}
	defer resp.Body.Close()
	record(true)
	if resp.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var result verifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, nil, err
	}
	// 私钥填错时所有人都无法通过, 按出错处理
	for _, code := range result.ErrorCodes {
		if code == "missing-input-secret" || code == "invalid-input-secret" {
			return false, nil, fmt.Errorf("secret key rejected: %s", code)
		}
	}
	return result.Success, result.ErrorCodes, nil
}

// inCooldown 在 passedAt 通过验证后, now 时是否还在 cooldown 秒的冷却内; 冷却为 0 时每次都要验证
func inCooldown(passedAt, now time.Time, cooldown int) bool {
	if cooldown <= 0 {
		return false
	}
	elapsed := now.Sub(passedAt)
	// 时钟回拨时不算在冷却内
	return elapsed >= 0 && elapsed < time.Duration(cooldown)*time.Second
}

// passedRecently 用户是否在冷却时间内通过过验证; 保存的是通过的时间, 修改冷却时间后立即按新的值判断
//
// 没有登录(userID 为 0)、没有启用 Redis 或读取出错时视为没有通过过, 照常验证。
func passedRecently(ctx context.Context, userID uint64, cooldown int) bool {
	if userID == 0 || cooldown <= 0 || db.Redis == nil {
		return false
	}
	millis, err := db.Redis.Get(ctx, passedKeyPrefix+strconv.FormatUint(userID, 10)).Int64()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			logger.ErrorF(ctx, "[Captcha] query last pass of %d failed: %v", userID, err)
		}
		return false
	}
	return inCooldown(time.UnixMilli(millis), time.Now(), cooldown)
}

// markPassed 记下用户刚通过了验证; 写入失败只记录日志, 下次照常验证
func markPassed(ctx context.Context, userID uint64) {
	if userID == 0 || db.Redis == nil {
		return
	}
	if err := db.Redis.Set(ctx, passedKeyPrefix+strconv.FormatUint(userID, 10), time.Now().UnixMilli(), passedTTL).Err(); err != nil {
		logger.ErrorF(ctx, "[Captcha] save last pass of %d failed: %v", userID, err)
	}
}
