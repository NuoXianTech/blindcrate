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

package fault

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

// custom 自定义的错误类型, 实现 Kind 后与 *Error 一样处理
type custom struct{}

func (*custom) Error() string { return "custom" }
func (*custom) Kind() Kind    { return KindForbidden }

func TestKindOf(t *testing.T) {
	for _, c := range []struct {
		name   string
		err    error
		kind   Kind
		status int
		msg    string
	}{
		{"invalid", Invalid("bad"), KindInvalid, http.StatusBadRequest, "bad"},
		{"formatted", NotFound("no %s", "item"), KindNotFound, http.StatusNotFound, "no item"},
		{"unauthenticated", Unauthenticated("who"), KindUnauthorized, http.StatusUnauthorized, "who"},
		{"forbidden", Forbidden("no"), KindForbidden, http.StatusForbidden, "no"},
		{"conflict", Conflict("taken"), KindConflict, http.StatusConflict, "taken"},
		{"too frequent", TooFrequent("slow"), KindTooFrequent, http.StatusTooManyRequests, "slow"},
		{"unavailable", Unavailable("down"), KindUnavailable, http.StatusServiceUnavailable, "down"},
		{"custom type", &custom{}, KindForbidden, http.StatusForbidden, "custom"},
		{"wrapped keeps the kind and hides the wrapper", fmt.Errorf("tx: %w", Conflict("taken")), KindConflict, http.StatusConflict, "taken"},
		{"plain", errors.New("conn reset"), KindInternal, http.StatusInternalServerError, InternalError},
		{"plain with a user-facing text is still internal", errors.New("bad"), KindInternal, http.StatusInternalServerError, InternalError},
	} {
		if got := KindOf(c.err); got != c.kind {
			t.Errorf("%s: kind = %d, want %d", c.name, got, c.kind)
		}
		if got := KindOf(c.err).Status(); got != c.status {
			t.Errorf("%s: status = %d, want %d", c.name, got, c.status)
		}
		if got := Message(c.err); got != c.msg {
			t.Errorf("%s: message = %q, want %q", c.name, got, c.msg)
		}
	}
}

func TestPercentInReason(t *testing.T) {
	// 没有参数时提示原样使用, 其中的百分号不当作格式
	if got := Invalid("100% off").Error(); got != "100% off" {
		t.Errorf("reason = %q", got)
	}
}
