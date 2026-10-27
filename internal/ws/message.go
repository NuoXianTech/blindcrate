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

package ws

import (
	"context"
	"encoding/json"
	"fmt"
)

// Message 客户端与服务端之间收发的消息
type Message struct {
	Type     string          `json:"type"`
	ID       string          `json:"id,omitempty"`        // 客户端请求时填写, 回复时原样带回
	ErrorMsg string          `json:"error_msg,omitempty"` // 处理失败的原因, 成功时为空
	Data     json.RawMessage `json:"data,omitempty"`
}

// HandlerFunc 处理一种客户端消息, 返回值作为回复的 data
//
// 返回的错误信息会直接发给客户端, 内部错误需要自行记录日志, 再返回 auth.InternalError。
// 处理函数在读取消息的 goroutine 中执行, 同一连接的消息按顺序处理; 耗时过长会阻塞该连接后续的消息,
// 超过 pongTimeout 还会因读不到 pong 被当作断线, 耗时的操作交给异步任务。
type HandlerFunc func(ctx context.Context, client *Client, data json.RawMessage) (interface{}, error)

// handlers 已注册的处理函数, 只在 API 服务启动前写入
var handlers = map[string]HandlerFunc{}

// Handle 注册一种客户端消息的处理函数, 需要在 API 服务启动前调用; 类型为空、与内置类型冲突或重复注册时 panic
func Handle(msgType string, handler HandlerFunc) {
	switch msgType {
	case "", TypePing, TypePong, TypeError:
		panic(fmt.Sprintf("ws: invalid message type %q", msgType))
	}
	if _, ok := handlers[msgType]; ok {
		panic(fmt.Sprintf("ws: handler for %q already registered", msgType))
	}
	handlers[msgType] = handler
}

// encode 把 data 填入消息并编码为 JSON, data 为 nil 时不带 data 字段
func encode(msg Message, data interface{}) ([]byte, error) {
	if data != nil {
		raw, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		msg.Data = raw
	}
	return json.Marshal(msg)
}
