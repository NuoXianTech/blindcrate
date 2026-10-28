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

// Package ws WebSocket 长连接, 用于实时推送和玩法中的客户端请求
//
// 消息统一为 JSON 文本帧 {"type": "...", "id": "...", "error_msg": "...", "data": ...}:
//   - type 为消息类型, 格式为 <app>:<事件>, 如 auction:bid; ping、pong、error 为内置类型
//   - 客户端发来的每条消息都会收到一条 type 相同的回复(ping 回复 pong), id 原样带回, 用于对应请求和回复;
//     无法解析的消息回复 type 为 error
//   - 服务端主动推送的消息不带 id
//
// 推送: 调用 SendToUser、SendToUsers 或 Broadcast, 消息经 Redis Pub/Sub 转发到连接所在的 API 实例,
// 因此 worker 等其他进程也可以推送。推送不保证送达, 用户不在线或断线期间的消息直接丢弃, 客户端重连后需要重新拉取数据。
//
// 新增一种客户端消息: 在对应 app 的 ws.go 中实现 HandlerFunc, 并在 router.Serve 中调用 Handle 注册。
// 每个连接每秒最多把 messageRate 条消息交给处理函数, 超出的直接回复 ratelimit.TooManyRequests;
// 需要按用户或更严格的限制时, 在处理函数中调用 ratelimit.Allow。
package ws

import (
	"time"

	"github.com/coder/websocket"
)

// 内置的消息类型
const (
	TypePing  = "ping"
	TypePong  = "pong"
	TypeError = "error"
)

// 自定义的关闭码, 4000-4999 由应用自行定义
const (
	// StatusUnauthorized 登录已失效或被强制下线, 客户端收到后不应重连
	StatusUnauthorized websocket.StatusCode = 4001
	// StatusTooManyConnections 同一用户的连接数超过上限
	StatusTooManyConnections websocket.StatusCode = 4029
)

const (
	// heartbeatInterval 服务端 ping 客户端并检查 session 的间隔, 需要小于反向代理的空闲超时(Next.js 代理为 30 秒)
	heartbeatInterval = 15 * time.Second
	// pongTimeout 等待客户端回复 pong 的时间, 超时视为连接已断开
	pongTimeout = 10 * time.Second
	// writeTimeout 写出一条消息的超时时间
	writeTimeout = 10 * time.Second
)

const (
	// maxMessageSize 客户端单条消息的最大字节数, 超过时以 1009 关闭连接
	maxMessageSize = 8 << 10
	// sendQueueSize 每个连接待发送消息的队列长度, 队列满说明客户端处理不过来, 直接断开
	sendQueueSize = 256
	// maxConnsPerUser 同一用户在单个 API 实例上的最大连接数
	maxConnsPerUser = 10
	// messageRate 每个连接每秒最多交给处理函数的消息数
	messageRate = 10
	// messageBurst 每个连接允许连续交给处理函数的消息数, 之后按 messageRate 恢复
	messageBurst = 20
)

// eventChannel 各 API 实例订阅的 Redis 频道, 推送和断开连接的事件经它转发到连接所在的实例
const eventChannel = "blindcrate:ws:events"

// 事件的动作
const (
	actionSend       = "send"
	actionBroadcast  = "broadcast"
	actionDisconnect = "disconnect"
)
