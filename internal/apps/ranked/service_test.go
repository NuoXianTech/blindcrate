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

package ranked

import (
	"testing"
	"time"
)

func TestDeltaOf(t *testing.T) {
	deltas := []int{30, 10, -10, -30}
	cases := []struct {
		rank int16
		want int32
	}{
		{0, 0},
		{1, 30},
		{2, 10},
		{4, -30},
		{5, 0},
	}
	for _, c := range cases {
		if got := deltaOf(deltas, c.rank); got != c.want {
			t.Errorf("rank %d: got %d, want %d", c.rank, got, c.want)
		}
	}
}

func TestApply(t *testing.T) {
	if got := apply(1000, 30); got != 1030 {
		t.Errorf("gain: got %d", got)
	}
	if got := apply(20, -30); got != 0 {
		t.Errorf("floor: got %d", got)
	}
}

func TestCarry(t *testing.T) {
	cases := []struct {
		name    string
		last    int32
		percent int16
		want    int32
	}{
		{"half above", 1400, 50, 1200},
		{"half below", 800, 50, 900},
		{"full reset", 1400, 0, 1000},
		{"full keep", 1400, 100, 1400},
		{"truncate", 1001, 50, 1000},
	}
	for _, c := range cases {
		if got := carry(1000, c.last, c.percent); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestTierOf(t *testing.T) {
	tiers := []Tier{{Name: "青铜", MinRating: 0}, {Name: "白银", MinRating: 1100}, {Name: "黄金", MinRating: 1250}}
	cases := []struct {
		rating  int32
		current string
		next    string
	}{
		{0, "青铜", "白银"},
		{1099, "青铜", "白银"},
		{1100, "白银", "黄金"},
		{1250, "黄金", ""},
		{5000, "黄金", ""},
	}
	name := func(tier *Tier) string {
		if tier == nil {
			return ""
		}
		return tier.Name
	}
	for _, c := range cases {
		current, next := TierOf(tiers, c.rating)
		if name(current) != c.current || name(next) != c.next {
			t.Errorf("rating %d: got %s/%s, want %s/%s", c.rating, name(current), name(next), c.current, c.next)
		}
	}
	if current, next := TierOf(tiers[1:], 900); current != nil || next == nil || next.Name != "白银" {
		t.Errorf("below lowest: got %v/%v", current, next)
	}
	if current, next := TierOf(nil, 900); current != nil || next != nil {
		t.Errorf("no tiers: got %v/%v", current, next)
	}
}

func TestValidateTier(t *testing.T) {
	valid := Tier{Name: "白银", MinRating: 1100, Color: "#c0c0c0"}
	if err := ValidateTier(&valid); err != nil {
		t.Fatalf("valid: got %v", err)
	}
	cases := []struct {
		name   string
		modify func(t *Tier)
		want   string
	}{
		{"name empty", func(t *Tier) { t.Name = "" }, TierNameInvalid},
		{"rating negative", func(t *Tier) { t.MinRating = -1 }, TierRatingInvalid},
		{"rating over", func(t *Tier) { t.MinRating = MaxTierRating + 1 }, TierRatingInvalid},
		{"color upper", func(t *Tier) { t.Color = "#C0C0C0" }, TierColorInvalid},
		{"color short", func(t *Tier) { t.Color = "#ccc" }, TierColorInvalid},
	}
	for _, c := range cases {
		tier := valid
		c.modify(&tier)
		if err := ValidateTier(&tier); err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}

func TestFormatRemaining(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{time.Second, "1 分钟"},
		{5 * time.Minute, "5 分钟"},
		{5*time.Minute + time.Millisecond, "6 分钟"},
		{2 * time.Hour, "2 小时"},
		{90 * time.Minute, "1 小时 30 分钟"},
	}
	for _, c := range cases {
		if got := formatRemaining(c.d); got != c.want {
			t.Errorf("%s: got %q, want %q", c.d, got, c.want)
		}
	}
}
