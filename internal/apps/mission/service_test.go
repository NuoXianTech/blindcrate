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

package mission

import (
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/task/schedule"
)

// useLocation 测试期间把统计时区换成 UTC+8, 结束后恢复
func useLocation(t *testing.T) *time.Location {
	t.Helper()
	old := schedule.Location
	loc := time.FixedZone("UTC+8", 8*3600)
	schedule.Location = loc
	t.Cleanup(func() { schedule.Location = old })
	return loc
}

func TestPeriod(t *testing.T) {
	loc := useLocation(t)
	// UTC 的周日 16:30 在 UTC+8 已经是周一, 属于新的一天和新的一周
	now := time.Date(2026, 11, 1, 16, 30, 0, 0, time.UTC)
	cases := map[string]string{
		CycleDaily:  "day:2026-11-02",
		CycleWeekly: "week:2026-W45",
		CycleOnce:   stats.PeriodTotal,
	}
	for cycle, want := range cases {
		if got := Period(cycle, now); got != want {
			t.Errorf("%s: got %q, want %q", cycle, got, want)
		}
	}
	// ISO 周跨年: 2027-01-01 是周五, 属于 2026 年的第 53 周
	if got := Period(CycleWeekly, time.Date(2027, 1, 1, 12, 0, 0, 0, loc)); got != "week:2026-W53" {
		t.Errorf("iso year: got %q", got)
	}
}

func TestResetsAt(t *testing.T) {
	loc := useLocation(t)
	cases := []struct {
		name  string
		now   time.Time
		cycle string
		want  time.Time
	}{
		{"daily", time.Date(2026, 10, 30, 23, 59, 0, 0, loc), CycleDaily, time.Date(2026, 10, 31, 0, 0, 0, 0, loc)},
		{"weekly friday", time.Date(2026, 10, 30, 10, 0, 0, 0, loc), CycleWeekly, time.Date(2026, 11, 2, 0, 0, 0, 0, loc)},
		{"weekly sunday", time.Date(2026, 11, 1, 10, 0, 0, 0, loc), CycleWeekly, time.Date(2026, 11, 2, 0, 0, 0, 0, loc)},
		{"weekly monday", time.Date(2026, 11, 2, 0, 0, 0, 0, loc), CycleWeekly, time.Date(2026, 11, 9, 0, 0, 0, 0, loc)},
	}
	for _, c := range cases {
		got := ResetsAt(c.cycle, c.now)
		if got == nil || !got.Equal(c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
		// 重置时间之后就是下一个周期
		if got != nil && Period(c.cycle, *got) == Period(c.cycle, c.now) {
			t.Errorf("%s: period did not change at reset", c.name)
		}
	}
	if got := ResetsAt(CycleOnce, time.Now()); got != nil {
		t.Errorf("once: got %v", got)
	}
}

// TestValidate 只校验不需要查库的部分
func TestValidate(t *testing.T) {
	valid := func() Mission {
		return Mission{Code: "daily_matches", Name: "每日对局", Cycle: CycleDaily, Scope: stats.ScopeAll, Metric: stats.MetricMatches, Target: 3}
	}
	cases := []struct {
		name   string
		modify func(m *Mission)
		want   string
	}{
		{"code", func(m *Mission) { m.Code = "Daily" }, CodeInvalid},
		{"name", func(m *Mission) { m.Name = "" }, NameInvalid},
		{"cycle", func(m *Mission) { m.Cycle = "monthly" }, CycleInvalid},
		{"scope", func(m *Mission) { m.Scope = "mode:" }, ScopeInvalid},
		{"target zero", func(m *Mission) { m.Target = 0 }, TargetInvalid},
		{"target too large", func(m *Mission) { m.Target = MaxTarget + 1 }, TargetInvalid},
	}
	for _, c := range cases {
		m := valid()
		c.modify(&m)
		if err := Validate(nil, &m); err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}

func TestValidCycle(t *testing.T) {
	for _, c := range Cycles() {
		if !ValidCycle(c.Code) || c.Label == "" {
			t.Errorf("%s: not valid", c.Code)
		}
	}
	if ValidCycle("") {
		t.Error("empty cycle is valid")
	}
}
