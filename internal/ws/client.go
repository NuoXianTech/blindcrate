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
	"errors"
	"runtime/debug"
	"sync"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/otel_trace"
	"github.com/coder/websocket"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Client 一个 WebSocket 连接, 同一用户可以同时有多个连接(多个页面或设备)
type Client struct {
	userID    uint64
	sessionID string
	conn      *websocket.Conn
	sendCh    chan []byte               // 待发送的消息, 由 writeLoop 依次写出
	closeCh   chan websocket.CloseError // 其他 goroutine 发起的关闭, 由 writeLoop 完成关闭握手
	cancel    context.CancelFunc        // 取消后连接立即断开
	kickOnce  sync.Once
}

func newClient(userID uint64, sessionID string, conn *websocket.Conn, cancel context.CancelFunc) *Client {
	return &Client{
		userID:    userID,
		sessionID: sessionID,
		conn:      conn,
		sendCh:    make(chan []byte, sendQueueSize),
		closeCh:   make(chan websocket.CloseError, 1),
		cancel:    cancel,
	}
}

// UserID 连接所属的用户
func (c *Client) UserID() uint64 {
	return c.userID
}

// serve 处理连接直到断开: 当前 goroutine 读消息, 写消息和心跳各占一个 goroutine
func (c *Client) serve(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Go(func() { c.writeLoop(ctx) })
	wg.Go(func() { c.heartbeatLoop(ctx) })

	c.readLoop(ctx)

	// 读取结束说明连接已断开, 让另外两个 goroutine 退出
	c.cancel()
	wg.Wait()
}

// send 把消息放入发送队列, 不阻塞调用方; 队列已满说明客户端处理不过来, 直接断开, 由客户端重连后重新拉取数据
func (c *Client) send(frame []byte) {
	select {
	case c.sendCh <- frame:
	default:
		c.kickOnce.Do(func() {
			logger.WarnF(context.Background(), "[WebSocket] send queue of %d is full, disconnect", c.userID)
			c.cancel()
		})
	}
}

// close 以指定的关闭码断开连接, 不阻塞调用方
func (c *Client) close(code websocket.StatusCode, reason string) {
	select {
	case c.closeCh <- websocket.CloseError{Code: code, Reason: reason}:
	default:
		// 已经有一个关闭请求
	}
}

// readLoop 读取并处理客户端消息, 连接断开时返回
func (c *Client) readLoop(ctx context.Context) {
	for {
		typ, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageText {
			_ = c.conn.Close(websocket.StatusUnsupportedData, BinaryUnsupported)
			return
		}
		c.handle(ctx, data)
	}
}

// writeLoop 依次写出发送队列中的消息, 收到关闭请求时完成关闭握手后退出
func (c *Client) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ce := <-c.closeCh:
			_ = c.conn.Close(ce.Code, ce.Reason)
			return
		case frame := <-c.sendCh:
			if err := c.write(ctx, frame); err != nil {
				_ = c.conn.CloseNow()
				return
			}
		}
	}
}

// write 写出一条消息, 超时后连接会被直接关闭
func (c *Client) write(ctx context.Context, frame []byte) error {
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return c.conn.Write(ctx, websocket.MessageText, frame)
}

// heartbeatLoop 定期 ping 客户端, 并检查 session 是否仍然有效(登出、改密码、过期等)
func (c *Client) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		// 客户端没有及时回复 pong, 视为连接已断开
		if err := c.ping(ctx); err != nil {
			_ = c.conn.CloseNow()
			return
		}

		active, err := auth.IsSessionActive(ctx, c.userID, c.sessionID)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			// Redis 暂时不可用时保留连接, 下次心跳再检查
			logger.ErrorF(ctx, "[WebSocket] check session of %d failed: %v", c.userID, err)
			continue
		}
		if !active {
			_ = c.conn.Close(StatusUnauthorized, SessionExpired)
			return
		}
	}
}

// ping 发送 ping 并等待客户端回复 pong
func (c *Client) ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, pongTimeout)
	defer cancel()
	return c.conn.Ping(ctx)
}

// handle 处理一条客户端消息并回复
func (c *Client) handle(ctx context.Context, raw []byte) {
	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil || msg.Type == "" {
		c.reply(ctx, Message{Type: TypeError, ID: msg.ID, ErrorMsg: MessageInvalid}, nil)
		return
	}
	if msg.Type == TypePing {
		c.reply(ctx, Message{Type: TypePong, ID: msg.ID}, nil)
		return
	}
	handler, ok := handlers[msg.Type]
	if !ok {
		c.reply(ctx, Message{Type: msg.Type, ID: msg.ID, ErrorMsg: MessageTypeUnknown}, nil)
		return
	}

	// 每条消息单独记录 trace, 与 HTTP 请求一样按采样率采样
	ctx, span := otel_trace.Start(ctx, "WebSocket_"+msg.Type, trace.WithNewRoot(), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	span.SetAttributes(
		attribute.String("ws.type", msg.Type),
		attribute.Int("ws.message_size", len(raw)),
	)

	start := time.Now()
	result, err := c.call(ctx, handler, msg.Data)
	latency := time.Since(start)

	reply := Message{Type: msg.Type, ID: msg.ID}
	if err != nil {
		reply.ErrorMsg = err.Error()
		result = nil
		span.SetStatus(codes.Error, reply.ErrorMsg)
	}
	c.reply(ctx, reply, result)

	logger.InfoF(
		ctx,
		"[WebSocket] %s %d\nStartTime: %s\nLatency: %d\nErrorMsg: %s",
		msg.Type,
		c.userID,
		start.Format(time.RFC3339),
		latency.Milliseconds(),
		reply.ErrorMsg,
	)
}

// call 调用处理函数, panic 时按内部错误回复, 不影响连接上的后续消息
func (c *Client) call(ctx context.Context, handler HandlerFunc, data json.RawMessage) (result interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorF(ctx, "[WebSocket] handler panic: %v\n%s", r, debug.Stack())
			result, err = nil, errors.New(auth.InternalError)
		}
	}()
	return handler(ctx, c, data)
}

// reply 把回复放入发送队列
func (c *Client) reply(ctx context.Context, msg Message, data interface{}) {
	frame, err := encode(msg, data)
	if err != nil {
		logger.ErrorF(ctx, "[WebSocket] encode reply of %s failed: %v", msg.Type, err)
		frame, _ = encode(Message{Type: msg.Type, ID: msg.ID, ErrorMsg: auth.InternalError}, nil)
	}
	c.send(frame)
}
