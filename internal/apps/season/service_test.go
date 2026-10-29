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

package season

import (
	"strings"
	"testing"
	"time"
)

func TestStatusAt(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	s := Season{StartsAt: start, EndsAt: start.AddDate(0, 1, 0)}
	cases := []struct {
		name string
		now  time.Time
		want string
	}{
		{"before start", start.Add(-time.Second), StatusUpcoming},
		{"at start", start, StatusActive},
		{"before end", s.EndsAt.Add(-time.Second), StatusActive},
		{"at end", s.EndsAt, StatusEnded},
	}
	for _, c := range cases {
		if got := s.StatusAt(c.now); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
	settled := s.EndsAt.Add(time.Hour)
	s.SettledAt = &settled
	if got := s.StatusAt(settled); got != StatusSettled {
		t.Errorf("settled: got %s", got)
	}
}

func TestValidate(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	valid := Season{Code: "s1", Name: "第一赛季", StartsAt: start, EndsAt: start.AddDate(0, 1, 0)}
	if err := Validate(&valid); err != nil {
		t.Fatalf("valid: got %v", err)
	}
	cases := []struct {
		name   string
		modify func(s *Season)
		want   string
	}{
		{"code upper", func(s *Season) { s.Code = "S1" }, CodeInvalid},
		{"code long", func(s *Season) { s.Code = strings.Repeat("a", 25) }, CodeInvalid},
		{"name empty", func(s *Season) { s.Name = "" }, NameInvalid},
		{"name long", func(s *Season) { s.Name = strings.Repeat("赛", 65) }, NameInvalid},
		{"end equals start", func(s *Season) { s.EndsAt = s.StartsAt }, TimeInvalid},
		{"end before start", func(s *Season) { s.EndsAt = s.StartsAt.Add(-time.Hour) }, TimeInvalid},
		{"carry negative", func(s *Season) { s.RatingCarry = -1 }, CarryInvalid},
		{"carry over", func(s *Season) { s.RatingCarry = 101 }, CarryInvalid},
	}
	for _, c := range cases {
		s := valid
		c.modify(&s)
		if err := Validate(&s); err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}

// TestPeriodLength 最长的赛季代码拼上前缀后不能超过 stats.period 的长度
func TestPeriodLength(t *testing.T) {
	if got := len(Period(strings.Repeat("a", 24))); got > 32 {
		t.Errorf("period length %d exceeds 32", got)
	}
}
