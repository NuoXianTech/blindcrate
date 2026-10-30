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

package checkin

import (
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/task/schedule"
)

func TestPeriods(t *testing.T) {
	old := schedule.Location
	loc := time.FixedZone("UTC+8", 8*3600)
	schedule.Location = loc
	t.Cleanup(func() { schedule.Location = old })

	cases := []struct {
		name             string
		now              time.Time
		today, yesterday string
	}{
		{"utc evening", time.Date(2026, 10, 30, 16, 30, 0, 0, time.UTC), "day:2026-10-31", "day:2026-10-30"},
		{"month start", time.Date(2026, 11, 1, 0, 0, 0, 0, loc), "day:2026-11-01", "day:2026-10-31"},
		{"year start", time.Date(2027, 1, 1, 9, 0, 0, 0, loc), "day:2027-01-01", "day:2026-12-31"},
	}
	for _, c := range cases {
		today, yesterday := periods(c.now)
		if today != c.today || yesterday != c.yesterday {
			t.Errorf("%s: got %q %q, want %q %q", c.name, today, yesterday, c.today, c.yesterday)
		}
	}
}

func TestCycleDay(t *testing.T) {
	cases := []struct {
		streak int32
		days   int
		want   int16
	}{
		{1, 7, 1},
		{7, 7, 7},
		{8, 7, 1},
		{15, 7, 1},
		{20, 7, 6},
		{3, 1, 1},
	}
	for _, c := range cases {
		if got := cycleDay(c.streak, c.days); got != c.want {
			t.Errorf("streak %d of %d days: got %d, want %d", c.streak, c.days, got, c.want)
		}
	}
}

// TestValidateDay 只校验不需要查库的部分
func TestValidateDay(t *testing.T) {
	for _, day := range []int16{0, MaxDays + 1} {
		if err := ValidateDay(nil, &Day{Day: day}); err == nil || err.Error() != DayInvalid {
			t.Errorf("day %d: got %v", day, err)
		}
	}
}
