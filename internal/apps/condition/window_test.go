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

package condition

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/task/schedule"
)

// useZone 测试期间把按天计算的时区固定为东八区
func useZone(t *testing.T) *time.Location {
	t.Helper()
	previous := schedule.Location
	zone := time.FixedZone("UTC+8", 8*3600)
	schedule.Location = zone
	t.Cleanup(func() { schedule.Location = previous })
	return zone
}

func TestParseClock(t *testing.T) {
	for _, c := range []struct {
		in   string
		end  bool
		want int
		ok   bool
	}{
		{"00:00", false, 0, true},
		{"19:30", false, 19*60 + 30, true},
		{"23:59", true, 23*60 + 59, true},
		{"24:00", true, 24 * 60, true},
		{"24:00", false, 0, false}, // 只有结束时间可以是 24:00
		{"24:01", true, 0, false},
		{"25:00", true, 0, false},
		{"07:60", false, 0, false},
		{"7:00", false, 0, false},
		{"0700", false, 0, false},
		{"07-00", false, 0, false},
		{"ab:cd", false, 0, false},
		{"", false, 0, false},
	} {
		if got, ok := parseClock(c.in, c.end); got != c.want || ok != c.ok {
			t.Errorf("parseClock(%q, %v) = %d, %v, want %d, %v", c.in, c.end, got, ok, c.want, c.ok)
		}
	}
}

func TestTimeWindowParse(t *testing.T) {
	for _, c := range []struct {
		name string
		p    timeWindowParams
		want string
	}{
		{"empty", timeWindowParams{}, WindowEmpty},
		{"bad start", timeWindowParams{StartsAt: "2026-11-18 10:00"}, WindowTimeInvalid},
		{"bad end", timeWindowParams{EndsAt: "tomorrow"}, WindowTimeInvalid},
		{"end before start", timeWindowParams{StartsAt: "2026-11-18T10:00:00+08:00", EndsAt: "2026-11-18T10:00:00+08:00"}, WindowTimeOrder},
		{"only from", timeWindowParams{From: "19:00"}, WindowClockPair},
		{"only to", timeWindowParams{To: "23:00"}, WindowClockPair},
		{"bad clock", timeWindowParams{From: "19:00", To: "25:00"}, WindowClockFormat},
		{"from at 24", timeWindowParams{From: "24:00", To: "02:00"}, WindowClockFormat},
		{"same clock", timeWindowParams{From: "19:00", To: "19:00"}, WindowClockSame},
		{"weekdays only", timeWindowParams{Weekdays: []int{6, 7}}, ""},
		{"clock only", timeWindowParams{From: "19:00", To: "23:00"}, ""},
		{"whole day", timeWindowParams{From: "00:00", To: "24:00"}, ""},
		{"overnight", timeWindowParams{From: "22:00", To: "02:00"}, ""},
		{"start only", timeWindowParams{StartsAt: "2026-11-18T10:00:00+08:00"}, ""},
		{"all", timeWindowParams{StartsAt: "2026-11-18T10:00:00+08:00", EndsAt: "2026-11-25T10:00:00+08:00", Weekdays: []int{1}, From: "08:00", To: "09:00"}, ""},
	} {
		if _, got := c.p.parse(); got != c.want {
			t.Errorf("%s: reason = %q, want %q", c.name, got, c.want)
		}
		if got, err := c.p.Validate(nil); err != nil || got != c.want {
			t.Errorf("%s: Validate = %q, %v, want %q", c.name, got, err, c.want)
		}
	}
}

// TestTimeWindowValidate 保存时按参数的校验规则检查星期: 只能是 1-7, 不能重复
func TestTimeWindowValidate(t *testing.T) {
	window := func(params string) []Condition {
		return []Condition{{Type: TypeTimeWindow, Params: json.RawMessage(params)}}
	}
	if err := Validate(nil, window(`{"weekdays":[6,7],"from":"19:00","to":"23:00"}`)); err != nil {
		t.Errorf("valid: %v", err)
	}
	for name, params := range map[string]string{
		"weekday 0":      `{"weekdays":[0]}`,
		"weekday 8":      `{"weekdays":[8]}`,
		"duplicate":      `{"weekdays":[1,1]}`,
		"unknown param":  `{"weekdays":[1],"hours":3}`,
		"nothing filled": `{}`,
	} {
		var invalid *InvalidError
		if err := Validate(nil, window(params)); !errors.As(err, &invalid) || invalid.Index != 1 || invalid.Reason == "" {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestWindowContains(t *testing.T) {
	zone := useZone(t)
	// 2026-11-21 是周六
	at := func(day, hour, minute int) time.Time { return time.Date(2026, 11, day, hour, minute, 0, 0, zone) }
	parse := func(p timeWindowParams) *window {
		t.Helper()
		w, reason := p.parse()
		if reason != "" {
			t.Fatalf("parse: %s", reason)
		}
		return w
	}

	weekend := parse(timeWindowParams{Weekdays: []int{7, 6}, From: "19:00", To: "23:00"})
	overnight := parse(timeWindowParams{Weekdays: []int{5}, From: "22:00", To: "02:00"})
	evening := parse(timeWindowParams{From: "20:00", To: "24:00"})
	week := parse(timeWindowParams{StartsAt: "2026-11-20T00:00:00+08:00", EndsAt: "2026-11-27T00:00:00+08:00"})
	monday := parse(timeWindowParams{Weekdays: []int{1}})
	for _, c := range []struct {
		name string
		w    *window
		at   time.Time
		want bool
	}{
		{"saturday before", weekend, at(21, 18, 59), false},
		{"saturday start", weekend, at(21, 19, 0), true},
		{"saturday end is exclusive", weekend, at(21, 23, 0), false},
		{"sunday", weekend, at(22, 22, 59), true},
		{"monday", weekend, at(23, 20, 0), false},
		// 其他时区的同一时刻按 schedule.time_zone 换算: UTC 周六 11:00 是东八区周六 19:00
		{"utc instant", weekend, time.Date(2026, 11, 21, 11, 0, 0, 0, time.UTC), true},
		{"utc instant before", weekend, time.Date(2026, 11, 21, 10, 59, 0, 0, time.UTC), false},

		// 跨到第二天的时间段, 零点之后的部分属于前一天, 星期按前一天算
		{"friday night", overnight, at(20, 22, 0), true},
		{"saturday small hours belong to friday", overnight, at(21, 1, 59), true},
		{"overnight end is exclusive", overnight, at(21, 2, 0), false},
		{"friday small hours belong to thursday", overnight, at(20, 1, 0), false},
		{"saturday night", overnight, at(21, 22, 30), false},
		{"friday afternoon", overnight, at(20, 15, 0), false},

		{"until midnight", evening, at(18, 23, 59), true},
		{"after midnight", evening, at(19, 0, 0), false},
		{"before evening", evening, at(18, 19, 59), false},

		{"before start", week, at(19, 23, 59), false},
		{"at start", week, at(20, 0, 0), true},
		{"last minute", week, at(26, 23, 59), true},
		{"at end is exclusive", week, at(27, 0, 0), false},

		{"weekday only", monday, at(23, 0, 0), true},
		{"other weekday", monday, at(22, 23, 59), false},
	} {
		if got := c.w.contains(c.at); got != c.want {
			t.Errorf("%s: contains(%s) = %v, want %v", c.name, c.at.Format(time.RFC3339), got, c.want)
		}
	}
}

func TestWindowText(t *testing.T) {
	useZone(t)
	for _, c := range []struct {
		p    timeWindowParams
		want string
	}{
		{timeWindowParams{Weekdays: []int{7, 6}, From: "19:00", To: "23:00"}, "开放时间: 每周六、周日 19:00-23:00"},
		{timeWindowParams{From: "22:00", To: "02:00"}, "开放时间: 22:00-次日 02:00"},
		{timeWindowParams{From: "20:00", To: "24:00"}, "开放时间: 20:00-24:00"},
		{timeWindowParams{Weekdays: []int{1}}, "开放时间: 每周一"},
		// 展示时换算成 schedule.time_zone 的时间
		{timeWindowParams{StartsAt: "2026-11-19T16:00:00Z", EndsAt: "2026-11-27T00:00:00+08:00"}, "开放时间: 2026-11-20 00:00 至 2026-11-27 00:00"},
		{timeWindowParams{StartsAt: "2026-11-20T00:00:00+08:00"}, "开放时间: 2026-11-20 00:00 起"},
		{timeWindowParams{EndsAt: "2026-11-27T12:30:00+08:00", Weekdays: []int{3}}, "开放时间: 2026-11-27 12:30 前 每周三"},
	} {
		w, reason := c.p.parse()
		if reason != "" {
			t.Errorf("parse %+v: %s", c.p, reason)
			continue
		}
		if got := w.text(); got != c.want {
			t.Errorf("text = %q, want %q", got, c.want)
		}
	}
}

// TestCheckTimeWindow 检查时不查库, 只看现在的时间
func TestCheckTimeWindow(t *testing.T) {
	useZone(t)
	open, err := checkTimeWindow(nil, 1, &timeWindowParams{StartsAt: "2000-01-01T00:00:00Z"})
	if err != nil || !open.Met || open.Text != "开放时间: 2000-01-01 08:00 起" {
		t.Errorf("open = %+v, %v", open, err)
	}
	closed, err := checkTimeWindow(nil, 1, &timeWindowParams{EndsAt: "2000-01-01T00:00:00Z"})
	if err != nil || closed.Met {
		t.Errorf("closed = %+v, %v", closed, err)
	}
	// 保存时已校验, 读出的参数不合法时返回内部错误而不是当作满足
	if _, err := checkTimeWindow(nil, 1, &timeWindowParams{}); err == nil {
		t.Error("empty window should fail")
	}
}
