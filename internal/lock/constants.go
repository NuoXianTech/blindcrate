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

// Package lock 基于 Redis 的分布式锁, 多个实例之间互斥, 如同一局对战同一时间只有一个请求在推进状态
//
// 锁带有过期时间, 持有者崩溃后到期自动释放; 业务耗时可能超过 ttl 时需要调用 Refresh 续期, 否则锁会提前失效。
// Redis 锁不能代替数据库约束: 锁失效的瞬间可能有两个持有者, 写库仍需要条件更新或版本号兜底。
//
// key 为被锁的对象, 格式为 <app>:<对象>:<ID>, 如 match:42。
// 与限流不同, Redis 出错时不放行, 直接返回错误。
package lock

import "time"

// keyPrefix 锁 key 的前缀, 完整的 key 如 blindcrate:lock:match:42
const keyPrefix = "blindcrate:lock:"

// retryInterval Acquire 等锁时重试的间隔
const retryInterval = 50 * time.Millisecond
