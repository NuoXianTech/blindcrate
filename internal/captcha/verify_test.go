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
}
