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

// Package reply 接口的统一响应: 响应壳、成功和失败的写法, 以及读取请求参数
//
// 处理函数只写业务: 参数用 JSON、Query、ID 读取, 不合法时已经写入 400; 业务返回的错误交给 Fail, 按 fault 的类别换成状态码。
package reply

import (
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/fault"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// OK 返回成功, data 可以为 nil
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Data: data})
}

// Error 返回指定状态码和提示
func Error(c *gin.Context, status int, msg string) {
	c.JSON(status, Response{ErrorMsg: msg})
}

// Abort 在中间件中返回指定状态码和提示, 并中止后面的处理
func Abort(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, Response{ErrorMsg: msg})
}

// Fail 把业务返回的错误转成响应
//
// 写明了类别的错误按类别返回状态码和提示(见 fault); 其余为内部错误, 记录日志后返回 500。
// action 为日志的开头, 如 "[CreateItem] create failed"。
func Fail(c *gin.Context, err error, action string) {
	if kind := fault.KindOf(err); kind != fault.KindInternal {
		c.JSON(kind.Status(), Response{ErrorMsg: fault.Message(err)})
		return
	}
	logger.ErrorF(c.Request.Context(), "%s: %v", action, err)
	c.JSON(http.StatusInternalServerError, Response{ErrorMsg: fault.InternalError})
}

// JSON 读取请求体并按 binding 校验, 不合法时写入 400 并返回 false
func JSON(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		Error(c, http.StatusBadRequest, fault.InvalidParams)
		return false
	}
	return true
}

// Query 读取查询参数并按 binding 校验, 不合法时写入 400 并返回 false
func Query(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindQuery(req); err != nil {
		Error(c, http.StatusBadRequest, fault.InvalidParams)
		return false
	}
	return true
}

// ID 读取路径中的数字 ID, 不合法时写入 400 并返回 false
func ID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, fault.InvalidParams)
		return 0, false
	}
	return id, true
}
