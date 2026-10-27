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
	"net/http"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/coder/websocket"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Response 统一的 ws 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// Connect
// @Tags ws
// @Summary 建立 WebSocket 连接
// @Description 消息为 JSON 文本 {type, id, error_msg, data}, 每 15 秒 ping 一次; 关闭码 4001 表示登录已失效或被强制下线, 不要重连; 4029 表示连接数过多
// @Produce json
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/ws [get]
func Connect(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		OriginPatterns: config.Config.WS.OriginPatterns,
	})
	if err != nil {
		// Accept 失败时已经写入了错误响应
		logger.WarnF(ctx, "[WebSocket] accept failed: %v", err)
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(maxMessageSize)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	client := newClient(user.ID, sessions.Default(c).ID(), conn, cancel)
	if !addClient(client) {
		_ = conn.Close(StatusTooManyConnections, TooManyConnections)
		return
	}
	defer removeClient(client)
	client.serve(ctx)
}
