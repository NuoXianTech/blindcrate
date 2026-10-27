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
	"slices"
	"sync"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
)

// event 经 Redis 转发到各 API 实例的事件
type event struct {
	Action  string          `json:"action"`
	UserIDs []uint64        `json:"user_ids,omitempty"`
	Message json.RawMessage `json:"message,omitempty"` // 编码好的消息, 原样发给客户端
}

var (
	// clients 当前 API 实例上的连接, 按用户分组
	clients   = map[uint64]map[*Client]struct{}{}
	clientsMu sync.RWMutex

	startOnce sync.Once
)

// Start 订阅 Redis 中的事件, 交给当前实例上的连接处理, API 服务启动时调用一次
func Start() error {
	var err error
	startOnce.Do(func() {
		ctx := context.Background()
		pubsub := db.Redis.Subscribe(ctx, eventChannel)
		// 等待订阅成功, 之后发布的事件都能收到
		if _, err = pubsub.Receive(ctx); err != nil {
			_ = pubsub.Close()
			return
		}
		go func() {
			for msg := range pubsub.Channel() {
				dispatch(msg.Payload)
			}
		}()
	})
	return err
}

// SendToUser 向用户的所有连接推送一条消息, 用户不在线时直接丢弃
func SendToUser(ctx context.Context, userID uint64, msgType string, data interface{}) error {
	return SendToUsers(ctx, []uint64{userID}, msgType, data)
}

// SendToUsers 向多个用户的所有连接推送同一条消息
func SendToUsers(ctx context.Context, userIDs []uint64, msgType string, data interface{}) error {
	if len(userIDs) == 0 {
		return nil
	}
	frame, err := encode(Message{Type: msgType}, data)
	if err != nil {
		return err
	}
	// 去重, 避免同一用户收到两次
	userIDs = slices.Compact(slices.Sorted(slices.Values(userIDs)))
	return publish(ctx, event{Action: actionSend, UserIDs: userIDs, Message: frame})
}

// Broadcast 向所有在线用户推送一条消息
func Broadcast(ctx context.Context, msgType string, data interface{}) error {
	frame, err := encode(Message{Type: msgType}, data)
	if err != nil {
		return err
	}
	return publish(ctx, event{Action: actionBroadcast, Message: frame})
}

// DisconnectUser 断开用户的所有连接, 用于封禁、停用等需要立即下线的场景, 客户端收到 4001 后不会重连
func DisconnectUser(ctx context.Context, userID uint64) error {
	return publish(ctx, event{Action: actionDisconnect, UserIDs: []uint64{userID}})
}

// publish 把事件发布到 Redis, 每个 API 实例都会收到
func publish(ctx context.Context, evt event) error {
	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	return db.Redis.Publish(ctx, eventChannel, payload).Err()
}

// dispatch 把收到的事件交给当前实例上对应的连接
func dispatch(payload string) {
	var evt event
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		logger.ErrorF(context.Background(), "[WebSocket] decode event failed: %v", err)
		return
	}
	switch evt.Action {
	case actionSend:
		for _, client := range findClients(evt.UserIDs) {
			client.send(evt.Message)
		}
	case actionBroadcast:
		for _, client := range allClients() {
			client.send(evt.Message)
		}
	case actionDisconnect:
		for _, client := range findClients(evt.UserIDs) {
			client.close(StatusUnauthorized, ForcedOffline)
		}
	default:
		logger.WarnF(context.Background(), "[WebSocket] unknown event action: %s", evt.Action)
	}
}

// addClient 登记连接, 用户在当前实例上的连接数已达上限时返回 false
func addClient(client *Client) bool {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	userClients := clients[client.userID]
	if len(userClients) >= maxConnsPerUser {
		return false
	}
	if userClients == nil {
		userClients = map[*Client]struct{}{}
		clients[client.userID] = userClients
	}
	userClients[client] = struct{}{}
	return true
}

// removeClient 移除已断开的连接
func removeClient(client *Client) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	userClients := clients[client.userID]
	delete(userClients, client)
	if len(userClients) == 0 {
		delete(clients, client.userID)
	}
}

// findClients 查找这些用户在当前实例上的连接
func findClients(userIDs []uint64) []*Client {
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	var result []*Client
	for _, userID := range userIDs {
		for client := range clients[userID] {
			result = append(result, client)
		}
	}
	return result
}

// allClients 当前实例上的所有连接
func allClients() []*Client {
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	var result []*Client
	for _, userClients := range clients {
		for client := range userClients {
			result = append(result, client)
		}
	}
	return result
}
