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

package reply

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/fault"
	"github.com/gin-gonic/gin"
)

// call 用 handler 处理一个请求, 返回状态码和响应
func call(t *testing.T, method, target, body string, handler gin.HandlerFunc) (int, Response) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Handle(method, "/x/:id", handler)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode %q failed: %v", w.Body.String(), err)
	}
	return w.Code, resp
}

func TestFail(t *testing.T) {
	for _, c := range []struct {
		err    error
		status int
		msg    string
	}{
		{fault.Invalid("名称不合法"), http.StatusBadRequest, "名称不合法"},
		{fault.NotFound("物品不存在"), http.StatusNotFound, "物品不存在"},
		{fault.Conflict("代码已被占用"), http.StatusConflict, "代码已被占用"},
		{errors.New("pq: connection reset"), http.StatusInternalServerError, fault.InternalError},
	} {
		status, resp := call(t, http.MethodGet, "/x/1", "", func(c2 *gin.Context) { Fail(c2, c.err, "[Test] failed") })
		if status != c.status || resp.ErrorMsg != c.msg || resp.Data != nil {
			t.Errorf("%v: got %d %+v, want %d %q", c.err, status, resp, c.status, c.msg)
		}
	}
}

func TestParams(t *testing.T) {
	type body struct {
		Name string `json:"name" form:"name" binding:"required,max=4"`
	}
	handler := func(c *gin.Context) {
		var req body
		id, ok := ID(c, "id")
		if !ok || !JSON(c, &req) {
			return
		}
		OK(c, map[string]interface{}{"id": id, "name": req.Name})
	}
	for _, c := range []struct {
		target, body string
		status       int
	}{
		{"/x/7", `{"name":"ab"}`, http.StatusOK},
		{"/x/abc", `{"name":"ab"}`, http.StatusBadRequest},
		{"/x/-1", `{"name":"ab"}`, http.StatusBadRequest},
		{"/x/7", `{"name":"abcde"}`, http.StatusBadRequest},
		{"/x/7", `{`, http.StatusBadRequest},
	} {
		status, resp := call(t, http.MethodPost, c.target, c.body, handler)
		if status != c.status {
			t.Errorf("%s %s: status = %d, want %d", c.target, c.body, status, c.status)
		}
		if status != http.StatusOK && resp.ErrorMsg != fault.InvalidParams {
			t.Errorf("%s %s: message = %q", c.target, c.body, resp.ErrorMsg)
		}
	}
	status, resp := call(t, http.MethodGet, "/x/1?name=abcde", "", func(c *gin.Context) {
		var req body
		if Query(c, &req) {
			OK(c, nil)
		}
	})
	if status != http.StatusBadRequest || resp.ErrorMsg != fault.InvalidParams {
		t.Errorf("query: got %d %+v", status, resp)
	}
}
