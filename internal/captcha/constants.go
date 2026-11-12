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

// Package captcha 人机验证, 使用 Cloudflare Turnstile
//
// 站点密钥、私钥和各场景的开关在系统设置的人机验证中配置, 见 settings.CaptchaSettings, 修改后立即生效。
//
// 使用: 在路由上加 Required 中间件, 场景开启时请求需要在 TokenHeader 请求头中带上前端组件拿到的令牌, 令牌只能使用一次;
// 没有令牌时返回 400, 验证没有通过时返回 403。
//
// 没有填写站点密钥或私钥时视为关闭; 读取设置出错、连不上 Cloudflare 或私钥填错时放行并记录日志: 人机验证只用于防刷, 不影响正常业务, 也避免管理员无法登录后台改回来。
package captcha

import "time"

// 场景, 每个场景可以单独开关
const (
	SceneLogin    = "login"    // 登录
	SceneRegister = "register" // 注册和发送注册验证码
	SceneCheckin  = "checkin"  // 签到
	SceneShop     = "shop"     // 商店购买
)

// TokenHeader 带令牌的请求头
const TokenHeader = "X-Captcha-Token"

const (
	// maxTokenLength 令牌的最大长度, Turnstile 的令牌不超过 2048 个字符
	maxTokenLength = 2048
	// verifyTimeout 请求 Cloudflare 的超时时间
	verifyTimeout = 5 * time.Second
)

// verifyURL Turnstile 的验证接口, 测试时替换
var verifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
