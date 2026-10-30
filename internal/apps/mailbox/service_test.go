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

package mailbox

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
)

func TestClaimable(t *testing.T) {
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	chip := reward.Bundle{Currencies: []reward.CurrencyLine{{Currency: "chip", Amount: 100}}}
	cases := []struct {
		name string
		mail Mail
		want bool
	}{
		{"unclaimed", Mail{Attachments: chip, ExpiresAt: now.Add(time.Hour)}, true},
		{"no attachments", Mail{ExpiresAt: now.Add(time.Hour)}, false},
		{"claimed", Mail{Attachments: chip, ClaimedAt: &now, ExpiresAt: now.Add(time.Hour)}, false},
		{"at expiry", Mail{Attachments: chip, ExpiresAt: now}, false},
	}
	for _, c := range cases {
		if got := c.mail.Claimable(now); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// TestValidateLetter 只校验不需要查库的部分, 附件为空时不查库
func TestValidateLetter(t *testing.T) {
	l := Letter{Sender: "  ", Title: "  维护补偿 ", Content: " 感谢耐心等待 \n"}
	if err := ValidateLetter(nil, &l); err != nil {
		t.Fatalf("valid: got %v", err)
	}
	if l.Sender != defaultSender || l.Title != "维护补偿" || l.Content != "感谢耐心等待" {
		t.Errorf("trimmed: got %+v", l)
	}
	cases := []struct {
		name   string
		letter Letter
		want   string
	}{
		{"title empty", Letter{Title: " "}, TitleInvalid},
		{"title long", Letter{Title: strings.Repeat("补", maxTitle+1)}, TitleInvalid},
		{"sender long", Letter{Sender: strings.Repeat("系", maxSender+1), Title: "t"}, SenderInvalid},
		{"content long", Letter{Title: "t", Content: strings.Repeat("文", maxContent+1)}, ContentInvalid},
		{"attachment amount", Letter{Title: "t", Attachments: reward.Bundle{Items: []reward.ItemLine{{ItemID: 1, Quantity: 0}}}}, fmt.Sprintf(AttachmentInvalid, reward.AmountInvalid)},
	}
	for _, c := range cases {
		err := ValidateLetter(nil, &c.letter)
		if err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}

func TestExpiryRange(t *testing.T) {
	for _, days := range []int{-1, MaxExpireDays + 1} {
		if _, err := Expiry(t.Context(), days); err == nil || err.Error() != ExpireInvalid {
			t.Errorf("days %d: got %v, want %s", days, err, ExpireInvalid)
		}
	}
	got, err := Expiry(t.Context(), 7)
	if err != nil {
		t.Fatalf("7 days: got %v", err)
	}
	if d := time.Until(got); d < 7*24*time.Hour-time.Minute || d > 7*24*time.Hour {
		t.Errorf("7 days: got %s from now", d)
	}
}

func TestRegisterSource(t *testing.T) {
	for _, code := range []string{"", "Admin", "too_long_source_code", SourceAdmin} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("source %q: want panic", code)
				}
			}()
			RegisterSource(code, "label")
		}()
	}
}
