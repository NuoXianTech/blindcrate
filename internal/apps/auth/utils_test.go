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

package auth

import (
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
)

// fakeSession 只保存值的 session
type fakeSession struct {
	values map[interface{}]interface{}
}

func newFakeSession() *fakeSession {
	return &fakeSession{values: map[interface{}]interface{}{}}
}

func (s *fakeSession) ID() string                           { return "fake" }
func (s *fakeSession) Get(key interface{}) interface{}      { return s.values[key] }
func (s *fakeSession) Set(key interface{}, val interface{}) { s.values[key] = val }
func (s *fakeSession) Delete(key interface{})               { delete(s.values, key) }
func (s *fakeSession) Clear()                               { s.values = map[interface{}]interface{}{} }
func (s *fakeSession) AddFlash(interface{}, ...string)      {}
func (s *fakeSession) Flashes(...string) []interface{}      { return nil }
func (s *fakeSession) Options(sessions.Options)             {}
func (s *fakeSession) Save() error                          { return nil }

func TestSessionMatchesUser(t *testing.T) {
	registered := time.Date(2026, 11, 1, 8, 30, 0, 123456789, time.UTC)
	user := &User{ID: 7, Username: "alice", CreatedAt: registered}

	session := newFakeSession()
	setSessionUser(session, user)
	if GetUserIDFromSession(session) != 7 {
		t.Fatalf("user id = %d, want 7", GetUserIDFromSession(session))
	}
	if !sessionMatchesUser(session, user) {
		t.Fatal("session of the account itself should match")
	}

	// 数据库只保存到微秒, 读回来的注册时间比注册时的少了纳秒
	loaded := &User{ID: 7, Username: "alice", CreatedAt: registered.Truncate(time.Microsecond).In(time.Local)}
	if !sessionMatchesUser(session, loaded) {
		t.Fatal("session should match the account read back from the database")
	}

	// 数据库重建后另一个人注册到了同一个 ID
	other := &User{ID: 7, Username: "bob", CreatedAt: registered.Add(time.Hour)}
	if sessionMatchesUser(session, other) {
		t.Fatal("session should not match another account with the same id")
	}

	// 没有账号标识的旧 session
	legacy := newFakeSession()
	legacy.Set(UserIDKey, uint64(7))
	legacy.Set(UserNameKey, "alice")
	if sessionMatchesUser(legacy, user) {
		t.Fatal("session without a stamp should not match")
	}
}
