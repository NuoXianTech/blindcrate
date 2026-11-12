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
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
)

// client 请求 Cloudflare 用的客户端
var client = &http.Client{Timeout: verifyTimeout}

// verifyResponse Turnstile 验证接口的响应
type verifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// checkScene 场景不存在时 panic, 在注册路由时调用
func checkScene(scene string) {
	switch scene {
	case SceneLogin, SceneRegister, SceneCheckin, SceneShop:
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
	}
	return false
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
		return false, nil, err
	}
	defer resp.Body.Close()
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
