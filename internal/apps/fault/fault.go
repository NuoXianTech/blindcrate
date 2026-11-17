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

// Package fault 可以直接展示给用户的错误: 创建时写明属于哪一类, 接口层按类别换成状态码, 见 reply.Fail
//
//   - 创建: 用 Invalid、NotFound、Conflict 等函数代替 errors.New, 提示写在各模块的 err.go 中
//   - 自定义类型: 带有其他字段的错误类型实现 Kind 方法即可, 如各模块的 InvalidError
//   - 判断: KindOf 沿着 Unwrap 查找, 没有写明类别的错误都是内部错误, 不能展示
//
// 同一个提示在不同的地方可以属于不同的类别, 如物品不存在: 修改这个物品时是 NotFound, 发放时引用了它是 Invalid。
package fault

import (
	"errors"
	"fmt"
	"net/http"
)

// 通用的提示
const (
	Unauthorized  = "未登录"
	InvalidParams = "参数不合法"
	InternalError = "服务器内部错误"
)

// Kind 错误的类别
type Kind int

const (
	KindInternal     Kind = iota // 内部错误, 不能展示, 记录日志后返回 InternalError
	KindInvalid                  // 提交的内容不合法
	KindUnauthorized             // 身份验证没有通过
	KindForbidden                // 当前不允许这样做, 如条件不满足、余额不足
	KindNotFound                 // 操作的对象不存在
	KindConflict                 // 与当前状态冲突, 如已被占用、已领取、重复提交
	KindTooFrequent              // 操作过于频繁
	KindUnavailable              // 依赖的服务暂不可用
)

// statuses 类别对应的状态码
var statuses = map[Kind]int{
	KindInvalid:      http.StatusBadRequest,
	KindUnauthorized: http.StatusUnauthorized,
	KindForbidden:    http.StatusForbidden,
	KindNotFound:     http.StatusNotFound,
	KindConflict:     http.StatusConflict,
	KindTooFrequent:  http.StatusTooManyRequests,
	KindUnavailable:  http.StatusServiceUnavailable,
}

// Status 类别对应的状态码, 内部错误为 500
func (k Kind) Status() int {
	if status, ok := statuses[k]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// Error 写明了类别的错误, Error 返回的提示可以直接展示
type Error struct {
	kind   Kind
	reason string
}

func (e *Error) Error() string { return e.reason }

// Kind 错误的类别
func (e *Error) Kind() Kind { return e.kind }

// kinded 写明了类别的错误, 自定义的错误类型实现 Kind 方法后与 *Error 一样处理
type kinded interface {
	error
	Kind() Kind
}

// KindOf 错误的类别, 没有写明类别的错误(数据库错误等)为 KindInternal
func KindOf(err error) Kind {
	var k kinded
	if errors.As(err, &k) {
		return k.Kind()
	}
	return KindInternal
}

// Message 可以展示的提示: 写明了类别的那一层错误的提示, 内部错误为 InternalError
func Message(err error) string {
	var k kinded
	if errors.As(err, &k) && k.Kind() != KindInternal {
		return k.Error()
	}
	return InternalError
}

// New 创建一个指定类别的错误, 有参数时 reason 为格式
func New(kind Kind, reason string, args ...any) error {
	if len(args) > 0 {
		reason = fmt.Sprintf(reason, args...)
	}
	return &Error{kind: kind, reason: reason}
}

// Invalid 提交的内容不合法
func Invalid(reason string, args ...any) error { return New(KindInvalid, reason, args...) }

// Unauthenticated 身份验证没有通过, 如用户名或密码错误
func Unauthenticated(reason string, args ...any) error { return New(KindUnauthorized, reason, args...) }

// Forbidden 当前不允许这样做
func Forbidden(reason string, args ...any) error { return New(KindForbidden, reason, args...) }

// NotFound 操作的对象不存在
func NotFound(reason string, args ...any) error { return New(KindNotFound, reason, args...) }

// Conflict 与当前状态冲突
func Conflict(reason string, args ...any) error { return New(KindConflict, reason, args...) }

// TooFrequent 操作过于频繁
func TooFrequent(reason string, args ...any) error { return New(KindTooFrequent, reason, args...) }

// Unavailable 依赖的服务暂不可用
func Unavailable(reason string, args ...any) error { return New(KindUnavailable, reason, args...) }
