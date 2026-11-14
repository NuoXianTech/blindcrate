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

package captcha

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
)

func TestEnabled(t *testing.T) {
	s := &settings.CaptchaSettings{SiteKey: "site", SecretKey: "secret", LoginEnabled: true, ShopEnabled: true}
	for scene, want := range map[string]bool{SceneLogin: true, SceneRegister: false, SceneCheckin: false, SceneShop: true} {
		if got := enabled(s, scene); got != want {
			t.Errorf("enabled(%q) = %t, want %t", scene, got, want)
		}
	}
	// 密钥没有填全时视为关闭
	for _, s := range []*settings.CaptchaSettings{
		{SecretKey: "secret", LoginEnabled: true},
		{SiteKey: "site", LoginEnabled: true},
	} {
		if enabled(s, SceneLogin) {
			t.Errorf("enabled without keys: %+v", s)
		}
	}
}

func TestCheckScene(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("unknown scene should panic")
		}
	}()
	Required("lottery")
}

func TestVerify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.PostFormValue("secret") != "secret" || r.PostFormValue("remoteip") != "203.0.113.7" {
			t.Errorf("request = %s %v", r.Method, r.PostForm)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.PostFormValue("response") {
		case "good":
			_, _ = w.Write([]byte(`{"success":true,"error-codes":[]}`))
		case "used":
			_, _ = w.Write([]byte(`{"success":false,"error-codes":["timeout-or-duplicate"]}`))
		case "secret":
			_, _ = w.Write([]byte(`{"success":false,"error-codes":["invalid-input-secret"]}`))
		case "html":
			_, _ = w.Write([]byte(`<html>`))
		default:
			w.WriteHeader(http.StatusBadGateway)
		}
	}))
	defer server.Close()
	old := verifyURL
	verifyURL = server.URL
	defer func() { verifyURL = old }()

	ctx := context.Background()
	if passed, _, err := verify(ctx, "secret", "good", "203.0.113.7"); err != nil || !passed {
		t.Errorf("good: passed = %t, err = %v", passed, err)
	}
	if passed, codes, err := verify(ctx, "secret", "used", "203.0.113.7"); err != nil || passed || !slices.Equal(codes, []string{"timeout-or-duplicate"}) {
		t.Errorf("used: passed = %t, codes = %v, err = %v", passed, codes, err)
	}
	for _, token := range []string{"secret", "html", "down"} {
		if passed, _, err := verify(ctx, "secret", token, "203.0.113.7"); err == nil || passed {
			t.Errorf("%s: passed = %t, err = %v", token, passed, err)
		}
	}
	// Cloudflare 有响应就不算连不上
	if paused() {
		t.Error("paused after responses from cloudflare")
	}
}

func TestBreaker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"error-codes":[]}`))
	}))
	defer server.Close()
	// 已经关闭的服务, 连接会被拒绝
	down := httptest.NewServer(http.NotFoundHandler())
	down.Close()
	old := verifyURL
	defer func() {
		verifyURL = old
		breaker.failures, breaker.until = 0, time.Time{}
	}()

	ctx := context.Background()
	verifyURL = down.URL
	for i := 1; i <= breakerThreshold; i++ {
		if paused() {
			t.Fatalf("paused after %d failures", i-1)
		}
		if _, _, err := verify(ctx, "secret", "good", ""); err == nil {
			t.Fatal("verify against a closed server should fail")
		}
	}
	if !paused() {
		t.Fatalf("not paused after %d failures", breakerThreshold)
	}

	// 请求方断开连接不计入
	breaker.failures = 0
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	for range breakerThreshold {
		_, _, _ = verify(canceled, "secret", "good", "")
	}
	if paused() {
		t.Error("paused by canceled requests")
	}

	// 到期后只放一个请求去试探, 仍然连不上时继续熔断
	breaker.failures = breakerThreshold
	breaker.until = time.Now().Add(-time.Second)
	if paused() {
		t.Fatal("no probe after cooldown")
	}
	if !paused() {
		t.Error("second request also probes")
	}
	_, _, _ = verify(ctx, "secret", "good", "")
	if !paused() {
		t.Error("not paused after a failed probe")
	}

	// 试探连上了就恢复验证
	breaker.until = time.Now().Add(-time.Second)
	verifyURL = server.URL
	if paused() {
		t.Fatal("no probe after cooldown")
	}
	if passed, _, err := verify(ctx, "secret", "good", ""); err != nil || !passed {
		t.Errorf("probe: passed = %t, err = %v", passed, err)
	}
	if paused() {
		t.Error("still paused after a successful probe")
	}
}
