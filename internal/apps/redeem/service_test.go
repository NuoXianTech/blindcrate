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

package redeem

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
)

// TestValidate 只校验不需要查库的部分
func TestValidate(t *testing.T) {
	valid := func() Batch {
		return Batch{Name: "开服礼包", MaxUses: 1, UserLimit: 1}
	}
	cases := []struct {
		name   string
		modify func(b *Batch)
		want   string
	}{
		{"name empty", func(b *Batch) { b.Name = "" }, NameInvalid},
		{"name too long", func(b *Batch) { b.Name = strings.Repeat("礼", maxNameLength+1) }, NameInvalid},
		{"max uses negative", func(b *Batch) { b.MaxUses = -1 }, MaxUsesInvalid},
		{"max uses too large", func(b *Batch) { b.MaxUses = maxCodeUses + 1 }, MaxUsesInvalid},
		{"user limit negative", func(b *Batch) { b.UserLimit = -1 }, UserLimitInvalid},
		{"user limit too large", func(b *Batch) { b.UserLimit = MaxBatchCodes + 1 }, UserLimitInvalid},
		{"reward empty", func(b *Batch) {}, RewardEmpty},
		{"reward amount", func(b *Batch) {
			b.Reward = reward.Bundle{Currencies: []reward.CurrencyLine{{Currency: "chip", Amount: 0}}}
		}, fmt.Sprintf(RewardInvalid, reward.AmountInvalid)},
	}
	for _, c := range cases {
		b := valid()
		c.modify(&b)
		if err := Validate(nil, &b); err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"ABCD2345EFGH":       "ABCD2345EFGH",
		"abcd-2345-efgh":     "ABCD2345EFGH",
		" abcd 2345\tefgh\n": "ABCD2345EFGH",
		"Vip2026":            "VIP2026",
		"新年 快乐":              "新年快乐",
		" - ":                "",
	}
	for input, want := range cases {
		got := Normalize(input)
		if got != want {
			t.Errorf("%q: got %q, want %q", input, got, want)
		}
		// 保存的是规范化后的兑换码, 再规范化一次需要不变, 否则原样输入时查不到
		if again := Normalize(got); again != got {
			t.Errorf("%q: not idempotent, got %q", input, again)
		}
	}
}

func TestValidCode(t *testing.T) {
	cases := []struct {
		code string
		want bool
	}{
		{"ABCD", true},
		{"新年快乐", true},
		{"VIP2026", true},
		{strings.Repeat("A", maxCodeLength), true},
		{strings.Repeat("码", maxCodeLength), true},
		{"", false},
		{"ABC", false},
		{strings.Repeat("A", maxCodeLength+1), false},
		{"AB_CD", false},
		{"AB CD", false},
		{"ABCD!", false},
		{"AB\x00CD", false},
	}
	for _, c := range cases {
		if got := ValidCode(c.code); got != c.want {
			t.Errorf("%q: got %v, want %v", c.code, got, c.want)
		}
	}
}

func TestGenerate(t *testing.T) {
	const count = 2000
	codes := Generate(count)
	if len(codes) != count {
		t.Fatalf("got %d codes, want %d", len(codes), count)
	}
	seen := make(map[string]bool, count)
	for _, code := range codes {
		if seen[code] {
			t.Fatalf("%s: duplicated", code)
		}
		seen[code] = true
		if len(code) != randomLength || strings.Trim(code, codeAlphabet) != "" {
			t.Fatalf("%s: unexpected format", code)
		}
		// 生成的兑换码原样保存, 需要已经是规范化后的合法兑换码
		if Normalize(code) != code || !ValidCode(code) {
			t.Fatalf("%s: not a normalized valid code", code)
		}
	}
	if got := Generate(0); len(got) != 0 {
		t.Errorf("zero: got %v", got)
	}
}

func TestExpired(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	at := func(d time.Duration) *time.Time {
		v := now.Add(d)
		return &v
	}
	cases := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"never", nil, false},
		{"future", at(time.Minute), false},
		// 到期时间不包含
		{"now", at(0), true},
		{"past", at(-time.Minute), true},
	}
	for _, c := range cases {
		b := Batch{ExpiresAt: c.expiresAt}
		if got := b.Expired(now); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
