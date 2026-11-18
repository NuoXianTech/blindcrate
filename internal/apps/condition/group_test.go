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
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// 这里用到的条件都不查库: 时间窗口只看现在的时间, 统计数据保存时只校验格式

const (
	// openWindow 总是满足的时间窗口, closedWindow 总是不满足的
	openWindow   = `{"type":"time_window","params":{"starts_at":"2000-01-01T00:00:00Z"}}`
	closedWindow = `{"type":"time_window","params":{"ends_at":"2000-01-01T00:00:00Z"}}`
	openText     = "开放时间: 2000-01-01 08:00 起"
	closedText   = "开放时间: 2000-01-01 08:00 前"
)

// group 一个条件组, rest 为 conditions 之外的参数, 如 `"count":1`
func group(conditionType, rest string, conditions ...string) Condition {
	params := `{"conditions":[` + strings.Join(conditions, ",") + `]`
	if rest != "" {
		params += "," + rest
	}
	return Condition{Type: conditionType, Params: json.RawMessage(params + "}")}
}

func TestGroupValidate(t *testing.T) {
	nested := `{"type":"any_of","params":{"conditions":[` + openWindow + `],"count":1}}`
	many := strings.Split(strings.Repeat(openWindow+"|", 9), "|")[:9]
	for _, c := range []struct {
		name string
		cond Condition
		want string // 为空表示合法, "*" 表示只要不合法
	}{
		{"any of", group(TypeAnyOf, `"count":1`, openWindow, closedWindow), ""},
		{"all of them", group(TypeAnyOf, `"count":2`, openWindow, closedWindow), ""},
		{"custom text", group(TypeAnyOf, `"count":1,"text":"周末或者老玩家"`, openWindow), ""},
		{"none of", group(TypeNoneOf, "", closedWindow), ""},
		{"count over", group(TypeAnyOf, `"count":2`, openWindow), NestedCountOver},
		{"count missing", group(TypeAnyOf, "", openWindow), "*"},
		{"no conditions", group(TypeAnyOf, `"count":1`), "*"},
		{"too many", group(TypeNoneOf, "", many...), "*"},
		{"text too long", group(TypeNoneOf, `"text":"`+strings.Repeat("长", 65)+`"`, openWindow), "*"},
		{"nested group", group(TypeAnyOf, `"count":1`, openWindow, nested), fmt.Sprintf(NestedInvalid, 2, NestedTooDeep)},
		{"nested in none of", group(TypeNoneOf, "", nested), fmt.Sprintf(NestedInvalid, 1, NestedTooDeep)},
		{"unknown type inside", group(TypeNoneOf, "", `{"type":"daily_limit","params":{}}`), fmt.Sprintf(NestedInvalid, 1, TypeNotFound)},
		{"invalid inside", group(TypeAnyOf, `"count":1`, openWindow, `{"type":"time_window","params":{}}`), fmt.Sprintf(NestedInvalid, 2, WindowEmpty)},
		{"malformed inside", group(TypeAnyOf, `"count":1`, `{"type":"account_age","params":{"days":"7"}}`), fmt.Sprintf(NestedInvalid, 1, ParamsInvalid)},
	} {
		err := Validate(nil, []Condition{{Type: TypeEmailVerified}, c.cond})
		var invalid *InvalidError
		switch {
		case c.want == "":
			if err != nil {
				t.Errorf("%s: %v", c.name, err)
			}
		case !errors.As(err, &invalid) || invalid.Index != 2 || (c.want != "*" && invalid.Reason != c.want):
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
	}

	// 只有参数里有一组条件的类型是条件组
	for conditionType, want := range map[string]bool{TypeAnyOf: true, TypeNoneOf: true, TypeMinStat: false, TypeTimeWindow: false} {
		if got := definitions[conditionType].group; got != want {
			t.Errorf("%s group = %v, want %v", conditionType, got, want)
		}
	}
}

func TestGroupEvaluate(t *testing.T) {
	useZone(t)
	for _, c := range []struct {
		name            string
		cond            Condition
		met             bool
		current, target int64
		text            string
	}{
		{"one of two", group(TypeAnyOf, `"count":1`, closedWindow, openWindow), true, 1, 1, closedText + " 或 " + openText},
		{"none of two", group(TypeAnyOf, `"count":1`, closedWindow, closedWindow), false, 0, 1, closedText + " 或 " + closedText},
		{"two of two", group(TypeAnyOf, `"count":2`, closedWindow, openWindow), false, 1, 2, "以下满足 2 项(1/2): " + closedText + "; " + openText},
		// 进度不超过需要的个数
		{"more than needed", group(TypeAnyOf, `"count":2`, openWindow, openWindow, openWindow), true, 2, 2, "以下满足 2 项(2/2): " + openText + "; " + openText + "; " + openText},
		{"custom text", group(TypeAnyOf, `"count":1,"text":"周末或者老玩家"`, closedWindow, openWindow), true, 1, 1, "周末或者老玩家"},
		{"none met", group(TypeNoneOf, "", closedWindow), true, 0, 0, "不能满足: " + closedText},
		{"some met", group(TypeNoneOf, "", closedWindow, openWindow), false, 0, 0, "以下都不能满足: " + closedText + "; " + openText},
		{"none of custom text", group(TypeNoneOf, `"text":"活动结束后不再开放"`, openWindow), false, 0, 0, "活动结束后不再开放"},
	} {
		progress, err := Evaluate(nil, 1, []Condition{c.cond})
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		want := Progress{Type: c.cond.Type, Met: c.met, Current: c.current, Target: c.target, Text: c.text}
		if !reflect.DeepEqual(progress[0], want) {
			t.Errorf("%s: progress = %+v, want %+v", c.name, progress[0], want)
		}
	}

	// 列表里的条件仍然需要全部满足, 条件组算一个条件
	list := []Condition{group(TypeAnyOf, `"count":1`, closedWindow, openWindow), group(TypeNoneOf, "", openWindow)}
	unmet, err := FirstUnmet(nil, 1, list)
	if err != nil || unmet == nil || unmet.Type != TypeNoneOf {
		t.Errorf("first unmet = %+v, %v", unmet, err)
	}
	if unmet, err := FirstUnmet(nil, 1, list[:1]); err != nil || unmet != nil {
		t.Errorf("all met: %+v, %v", unmet, err)
	}
}

func TestRewrite(t *testing.T) {
	// 从数据库读出的写法: 键按 JSONB 的顺序, 冒号和逗号后有空格
	const untouched = `{"count": 3, "scope": "mode:ranked", "metric": "matches"}`
	conditions := []Condition{
		{Type: TypeMinStat, Params: json.RawMessage(`{"count": 3, "scope": "venue:dock:pve", "metric": "wins", "period": "week"}`)},
		{Type: TypeMinStat, Params: json.RawMessage(untouched)},
		group(TypeAnyOf, `"count":1,"text":"venue:dock"`,
			`{"type":"stat_limit","params":{"metric":"matches","scope":"venue:dock","period":"day","count":5}}`,
			`{"type":"account_age","params":{"days":7}}`),
		group(TypeNoneOf, "", `{"type":"account_age","params":{"days":7}}`),
		{Type: "no_such_type", Params: json.RawMessage(`{"scope":"venue:dock"}`)},
		{Type: TypeMinStat, Params: json.RawMessage(`[1]`)},
		{Type: TypeMinStat, Params: json.RawMessage(`{"scope":7}`)},
	}
	before := make([]string, len(conditions))
	for i := range conditions {
		before[i] = string(conditions[i].Params)
	}

	seen := map[string][]string{}
	Rewrite(conditions, func(input, value string) string {
		seen[input] = append(seen[input], value)
		if input == "scope" {
			return strings.Replace(value, "venue:dock", "venue:pier", 1)
		}
		return value
	})

	// 每个字符串参数都带着它的输入方式; 展示的说明是普通文本, 内容像统计范围也不会被当成引用
	want := map[string][]string{
		"metric": {"wins", "matches", "matches"},
		"scope":  {"venue:dock:pve", "mode:ranked", "venue:dock"},
		"period": {"week", "day"},
		"text":   {"venue:dock"},
	}
	if !reflect.DeepEqual(seen, want) {
		t.Errorf("seen = %v, want %v", seen, want)
	}

	for i, wantParams := range []string{
		`{"count":3,"metric":"wins","period":"week","scope":"venue:pier:pve"}`,
		untouched, // 没有改动的条件保持原来的写法
		`{"conditions":[{"type":"stat_limit","params":{"count":5,"metric":"matches","period":"day","scope":"venue:pier"}},{"type":"account_age","params":{"days":7}}],"count":1,"text":"venue:dock"}`,
		before[3],
		before[4], // 类型不存在或参数解析不了的不变
		before[5],
		before[6],
	} {
		if got := string(conditions[i].Params); got != wantParams {
			t.Errorf("conditions[%d] = %s, want %s", i, got, wantParams)
		}
	}

	// 改写后的条件仍然合法
	if err := Validate(nil, conditions[:4]); err != nil {
		t.Errorf("rewritten conditions: %v", err)
	}
}
