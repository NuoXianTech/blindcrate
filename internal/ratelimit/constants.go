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

// Package ratelimit 基于 Redis 的限流, 多个 API 实例共享计数
//
// 规则 Rule{Limit: N, Period: T} 允许连续操作 N 次, 之后每隔 T/N 恢复一次, 平均每 T 内不超过 N 次。
// 算法为 GCRA, 每个 key 只保存一个时间戳, 时间取自 Redis, 不受各实例时钟误差的影响。
//
// 使用:
//   - HTTP 接口: 在路由上加 ByIP 或 ByUser 中间件, 超出时返回 429 和 Retry-After 头; ByUser 需要放在 auth.LoginRequired 之后
//   - WebSocket 消息处理等其他场景: 调用 Allow, key 用 IPKey、UserKey 生成, 超出时把 TooManyRequests 返回给用户
//
// scope 为被限流的操作, 格式为 <app>:<动作>, 如 auth:login; 同一 scope 需要始终使用同一规则。
// Redis 出错时放行并记录日志, 限流只用于防刷, 不影响正常业务。
package ratelimit

// keyPrefix 限流 key 的前缀, 完整的 key 如 blindcrate:rate_limit:auth:login:ip:127.0.0.1
const keyPrefix = "blindcrate:rate_limit:"
