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
	"slices"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
)

func TestValidateStat(t *testing.T) {
	for _, c := range []struct {
		metric, scope, period string
		want                  string
	}{
		{stats.MetricMatches, stats.ScopeAll, "", ""}, // 周期留空为累计
		{stats.MetricWins, "venue:dock:pve", stats.PeriodTotal, ""},
		{stats.MetricMatches, "mode:ranked", stats.PeriodNameDay, ""},
		{stats.MetricMatches, "mode:ranked", stats.PeriodNameWeek, ""},
		{"no_such_metric", stats.ScopeAll, "", MetricNotFound},
		{stats.MetricMatches, "venue:", "", ScopeInvalid},
		{stats.MetricMatches, stats.ScopeAll, "month", PeriodNotFound},
		// 周期填的是名称, 不是某一天、某一周的 stats.period
		{stats.MetricMatches, stats.ScopeAll, "day:2026-11-18", PeriodNotFound},
	} {
		if got := validateStat(c.metric, c.scope, c.period); got != c.want {
			t.Errorf("validateStat(%q, %q, %q) = %q, want %q", c.metric, c.scope, c.period, got, c.want)
		}
	}
	if got := periodOf(""); got != stats.PeriodTotal {
		t.Errorf("periodOf empty = %q", got)
	}
	if got := periodOf(stats.PeriodNameDay); got != stats.PeriodNameDay {
		t.Errorf("periodOf day = %q", got)
	}
}

// TestStatConditions 统计数据和统计上限保存时的校验; 每日上限已并入统计上限, 不再是一种条件类型
func TestStatConditions(t *testing.T) {
	stat := func(conditionType, params string) []Condition {
		return []Condition{{Type: conditionType, Params: json.RawMessage(params)}}
	}
	for name, conditions := range map[string][]Condition{
		"without period": stat(TypeMinStat, `{"metric":"matches","scope":"all","count":10}`),
		"weekly":         stat(TypeMinStat, `{"metric":"wins","scope":"mode:ranked","period":"week","count":3}`),
		"daily limit":    stat(TypeStatLimit, `{"metric":"matches","scope":"venue:dock","period":"day","count":5}`),
		"total limit":    stat(TypeStatLimit, `{"metric":"matches","scope":"venue:dock","count":20}`),
	} {
		if err := Validate(nil, conditions); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, c := range map[string]struct {
		conditions []Condition
		want       string
	}{
		"unknown period": {stat(TypeStatLimit, `{"metric":"matches","scope":"all","period":"month","count":5}`), PeriodNotFound},
		"unknown metric": {stat(TypeStatLimit, `{"metric":"kills","scope":"all","count":5}`), MetricNotFound},
		"bad scope":      {stat(TypeMinStat, `{"metric":"matches","scope":"dock","count":5}`), ScopeInvalid},
		"old type":       {stat("daily_limit", `{"metric":"matches","scope":"all","count":5}`), TypeNotFound},
	} {
		var invalid *InvalidError
		if err := Validate(nil, c.conditions); !errors.As(err, &invalid) || invalid.Reason != c.want {
			t.Errorf("%s: err = %v, want %q", name, err, c.want)
		}
	}
	var invalid *InvalidError
	if err := Validate(nil, stat(TypeStatLimit, `{"metric":"matches","scope":"all","count":0}`)); !errors.As(err, &invalid) || invalid.Index != 1 {
		t.Errorf("zero count: err = %v", err)
	}
}

// TestBuiltinTypes 通用的条件类型按这个顺序展示在后台, 其他模块注册的排在后面
func TestBuiltinTypes(t *testing.T) {
	want := []string{TypeMinBalance, TypeMinStat, TypeStatLimit, TypeHasItem, TypeAccountAge, TypeEmailVerified, TypeTimeWindow, TypeAnyOf, TypeNoneOf}
	if len(typeOrder) < len(want) || !slices.Equal(typeOrder[:len(want)], want) {
		t.Errorf("types = %v, want %v first", typeOrder, want)
	}
	// 统计数据和统计上限都可以选周期, 可选值由 stats 提供; 周期可以不填, 之前保存的条件没有这个参数
	for _, conditionType := range []string{TypeMinStat, TypeStatLimit} {
		fields := definitions[conditionType].schema.Fields()
		found := false
		for _, f := range fields {
			if f.Key == "period" {
				found = f.Input == stats.SourcePeriod && !f.Required
			}
		}
		if !found {
			t.Errorf("%s: period field = %+v", conditionType, fields)
		}
	}
}
