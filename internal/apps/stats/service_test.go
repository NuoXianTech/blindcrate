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

package stats

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/params"
	"github.com/NuoXianTech/blindcrate/internal/task/schedule"
	"gorm.io/gorm"
)

// TestNamedPeriods 配置中按名称选择周期: 内置的累计、今天、本周, 以及其他模块注册的(如本赛季)
func TestNamedPeriods(t *testing.T) {
	previous := schedule.Location
	schedule.Location = time.FixedZone("UTC+8", 8*3600)
	t.Cleanup(func() { schedule.Location = previous })
	// UTC 的周日 17:30 是东八区的周一 01:30, 按天和按周都算到下一个周期
	at := time.Date(2026, 11, 22, 17, 30, 0, 0, time.UTC)

	for _, c := range []struct {
		name, key string
		ok        bool
	}{
		{PeriodTotal, "total", true},
		{PeriodNameDay, "day:2026-11-23", true},
		{PeriodNameWeek, "week:2026-W48", true},
		{"month", "", false},
		{"", "", false},
	} {
		key, ok, err := PeriodKey(nil, c.name, at)
		if err != nil || key != c.key || ok != c.ok {
			t.Errorf("PeriodKey(%q) = %q, %v, %v, want %q, %v", c.name, key, ok, err, c.key, c.ok)
		}
		if HasPeriod(c.name) != c.ok {
			t.Errorf("HasPeriod(%q) = %v, want %v", c.name, !c.ok, c.ok)
		}
	}
	for name, want := range map[string]string{PeriodTotal: "累计", PeriodNameDay: "今天", PeriodNameWeek: "本周", "month": "month"} {
		if got := PeriodLabel(name); got != want {
			t.Errorf("PeriodLabel(%q) = %q, want %q", name, got, want)
		}
	}

	// 注册的周期: 现在有这种周期时返回它的 stats.period, 没有时为空; 查库出错时原样返回
	active := true
	failure := errors.New("conn reset")
	RegisterPeriod("era", "本纪元", func(_ *gorm.DB, at time.Time) (string, error) {
		switch {
		case at.IsZero():
			return "", failure
		case active:
			return "era:1", nil
		}
		return "", nil
	})
	if !HasPeriod("era") || PeriodLabel("era") != "本纪元" {
		t.Errorf("era: has = %v, label = %q", HasPeriod("era"), PeriodLabel("era"))
	}
	if key, ok, err := PeriodKey(nil, "era", at); err != nil || !ok || key != "era:1" {
		t.Errorf("active era = %q, %v, %v", key, ok, err)
	}
	active = false
	if key, ok, err := PeriodKey(nil, "era", at); err != nil || !ok || key != "" {
		t.Errorf("inactive era = %q, %v, %v", key, ok, err)
	}
	if _, _, err := PeriodKey(nil, "era", time.Time{}); !errors.Is(err, failure) {
		t.Errorf("failing era: err = %v", err)
	}

	// 后台的下拉框: 内置的在前, 注册的按注册顺序排在后面, 默认选累计
	options, err := periodOptions(nil)
	want := []params.Option{{Value: "total", Label: "累计"}, {Value: "day", Label: "今天"}, {Value: "week", Label: "本周"}, {Value: "era", Label: "本纪元"}}
	if err != nil || len(options) < len(want) || !reflect.DeepEqual(options[:3], want[:3]) || options[len(options)-1] != want[3] {
		t.Errorf("options = %+v, %v", options, err)
	}

	fn := func(*gorm.DB, time.Time) (string, error) { return "", nil }
	for name, register := range map[string]func(){
		"duplicate":   func() { RegisterPeriod("era", "本纪元", fn) },
		"builtin":     func() { RegisterPeriod(PeriodNameDay, "今天", fn) },
		"total":       func() { RegisterPeriod(PeriodTotal, "累计", fn) },
		"empty name":  func() { RegisterPeriod("", "空", fn) },
		"upper case":  func() { RegisterPeriod("Era", "纪元", fn) },
		"digit first": func() { RegisterPeriod("1era", "纪元", fn) },
		"too long":    func() { RegisterPeriod(strings.Repeat("a", 17), "纪元", fn) },
		"no label":    func() { RegisterPeriod("epoch", "", fn) },
		"no func":     func() { RegisterPeriod("epoch", "纪元", nil) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: RegisterPeriod should panic", name)
				}
			}()
			register()
		}()
	}
	if HasPeriod("epoch") {
		t.Error("rejected period should not be registered")
	}
}
