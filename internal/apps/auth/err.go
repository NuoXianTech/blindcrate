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

package auth

const (
	UnAuthorized       = "未登录"
	InvalidCredentials = "用户名或密码错误"
	InvalidParams      = "参数不合法"
	BannedAccount      = "账号已被封禁"
	InternalError      = "服务器内部错误"
	Forbidden          = "无权限"
	UsernameInvalid    = "用户名需为 3-32 位字母、数字或下划线"
	PasswordInvalid    = "密码长度需为 8-72 位"
	UsernameTaken      = "用户名已被占用"
	WrongPassword      = "原密码错误"
	EmailInvalid       = "邮箱格式不正确"
	EmailTaken         = "邮箱已被其他账号绑定"
	CodeTooFrequent    = "验证码发送过于频繁, 请稍后再试"
	CodeInvalid        = "验证码错误或已过期"
	MailUnavailable    = "邮件服务暂不可用"
)
