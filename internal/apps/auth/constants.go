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

import "time"

// session 和 gin.Context 中使用的键
const (
	UserNameKey = "username"
	UserIDKey   = "user_id"
	UserObjKey  = "user_obj"
)

const (
	// sessionKeyPrefix gin-contrib/sessions(redistore) 存储 session 的默认 key 前缀
	sessionKeyPrefix = "session_"
	// userSessionsKeyPrefix 记录用户所有 session ID 的集合, 用于让其他设备下线
	userSessionsKeyPrefix = "blindcrate:user_sessions:"
	// emailCodeKeyPrefix 邮箱验证码(email/code/attempts)
	emailCodeKeyPrefix = "blindcrate:email_code:"
	// emailCodeCooldownKeyPrefix 邮箱验证码发送冷却
	emailCodeCooldownKeyPrefix = "blindcrate:email_code_cooldown:"
)

const (
	emailCodeTTL         = 10 * time.Minute
	emailCodeCooldown    = time.Minute
	emailCodeMaxAttempts = 5
)
