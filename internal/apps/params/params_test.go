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

package params

import (
	"encoding/json"
	"reflect"
	"testing"

	"gorm.io/gorm"
)

type fieldsParams struct {
	Name  string `json:"name" label:"名称"`
	Kind  string `json:"kind,omitempty" label:"类型" help:"选一种" options:"a:甲,b:乙"`
	Item  uint64 `json:"item" label:"物品" input:"item"`
	Rate  int32  `json:"rate" label:"比例" unit:"bp"`
	Count int64  `json:"count" label:"数量"`
	On    bool   `json:"on" label:"开关"`
	Note  string `json:"note" label:"备注" input:"textarea"`
}

func describe[P any](v Vocabulary) []Field {
	return Describe(reflect.TypeFor[P](), "test", v)
}

func TestDescribe(t *testing.T) {
	want := []Field{
		{Key: "name", Label: "名称", Type: "string", Input: "text"},
		{Key: "kind", Label: "类型", Help: "选一种", Type: "string", Input: "select", Options: []Option{{Value: "a", Label: "甲"}, {Value: "b", Label: "乙"}}},
		{Key: "item", Label: "物品", Type: "int", Input: "item"},
		{Key: "rate", Label: "比例", Type: "int", Input: "number", Unit: UnitBasisPoints},
		{Key: "count", Label: "数量", Type: "int", Input: "number"},
		{Key: "on", Label: "开关", Type: "bool", Input: "switch"},
		{Key: "note", Label: "备注", Type: "string", Input: "textarea"},
	}
	if got := describe[fieldsParams](Plain); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	// 使用方声明了的写法: 整数列表、其他单位和自己处理的 tag
	type wide struct {
		Days   []int32 `json:"days" label:"天数" unit:"second"`
		Wait   int     `json:"wait" label:"等待" unit:"millisecond"`
		Token  string  `json:"token" label:"密钥" secret:"true"`
		Target string  `json:"target" label:"货币" ref:"currency" public:"true"`
	}
	vocabulary := Vocabulary{Units: []string{UnitSecond, UnitMillisecond}, IntLists: true, Tags: []string{"ref", "secret", "public"}}
	want = []Field{
		{Key: "days", Label: "天数", Type: "int_list", Input: "list", Unit: UnitSecond},
		{Key: "wait", Label: "等待", Type: "int", Input: "number", Unit: UnitMillisecond},
		{Key: "token", Label: "密钥", Type: "string", Input: "text"},
		{Key: "target", Label: "货币", Type: "string", Input: "text"},
	}
	if got := describe[wide](vocabulary); !reflect.DeepEqual(got, want) {
		t.Errorf("wide: got %+v, want %+v", got, want)
	}
}

func TestDescribePanics(t *testing.T) {
	type float struct {
		Ratio float64 `json:"ratio"`
	}
	type intOptions struct {
		Level int32 `json:"level" options:"1:一,2:二"`
	}
	type badOptions struct {
		Kind string `json:"kind" options:"a:甲,b"`
	}
	type stringUnit struct {
		Rate string `json:"rate" unit:"bp"`
	}
	type unknownUnit struct {
		Rate int32 `json:"rate" unit:"percent"`
	}
	type otherUnit struct {
		Wait int32 `json:"wait" unit:"second"`
	}
	type list struct {
		Days []int32 `json:"days"`
	}
	type ref struct {
		Currency string `json:"currency" ref:"currency"`
	}
	type secret struct {
		Token string `json:"token" secret:"true"`
	}
	cases := map[string]func(){
		"not struct":    func() { describe[int](Plain) },
		"float":         func() { describe[float](Plain) },
		"int options":   func() { describe[intOptions](Plain) },
		"bad options":   func() { describe[badOptions](Plain) },
		"string unit":   func() { describe[stringUnit](Plain) },
		"unknown unit":  func() { describe[unknownUnit](Plain) },
		"other unit":    func() { describe[otherUnit](Plain) },
		"int list":      func() { describe[list](Plain) },
		"ref tag":       func() { describe[ref](Plain) },
		"secret tag":    func() { describe[secret](Plain) },
		"string slices": func() { describe[struct{ Names []string }](Vocabulary{IntLists: true}) },
	}
	for name, fn := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: expected panic", name)
				}
			}()
			fn()
		}()
	}
}

type schemaParams struct {
	Currency string `json:"currency" label:"货币" binding:"required,max=8"`
	Amount   int64  `json:"amount" label:"金额" binding:"min=1"`
	Strict   bool   `json:"strict" label:"严格"`
}

// Validate 货币为 none 时不合法, 为 boom 时查库失败
func (p *schemaParams) Validate(*gorm.DB) (string, error) {
	switch p.Currency {
	case "none":
		return "货币不存在", nil
	case "boom":
		return "", gorm.ErrInvalidDB
	}
	return "", nil
}

func TestSchema(t *testing.T) {
	plain := New[schemaParams]("test", Plain, nil)
	withDefaults := New("test", Plain, &schemaParams{Currency: "chip", Amount: 5, Strict: true})

	decoded, problem, err := plain.Validate(nil, json.RawMessage(`{"amount": 3, "currency":"gold"}`))
	if err != nil || problem != nil || *decoded.(*schemaParams) != (schemaParams{Currency: "gold", Amount: 3}) {
		t.Fatalf("valid: got %+v, %+v, %v", decoded, problem, err)
	}
	if got, err := Normalize(decoded); err != nil || !reflect.DeepEqual(got, map[string]any{"currency": "gold", "amount": float64(3), "strict": false}) {
		t.Errorf("normalize: got %v, %v", got, err)
	}

	problems := map[string]struct {
		raw  string
		want Problem
	}{
		"unknown key":  {`{"currency":"gold","amount":3,"other":1}`, Problem{Malformed: true}},
		"wrong type":   {`{"currency":1,"amount":3}`, Problem{Malformed: true}},
		"not object":   {`[1]`, Problem{Malformed: true}},
		"binding rule": {`{"currency":"gold","amount":0}`, Problem{Reason: "金额最小只能为1"}},
		"missing":      {``, Problem{Reason: "货币为必填字段"}},
		"validator":    {`{"currency":"none","amount":3}`, Problem{Reason: "货币不存在"}},
	}
	for name, c := range problems {
		decoded, problem, err := plain.Validate(nil, json.RawMessage(c.raw))
		if err != nil || decoded != nil || problem == nil || *problem != c.want {
			t.Errorf("%s: got %+v, %+v, %v, want %+v", name, decoded, problem, err, c.want)
		}
	}
	if _, problem, err := plain.Validate(nil, json.RawMessage(`{"currency":"boom","amount":3}`)); err == nil || problem != nil {
		t.Errorf("internal error: got %+v, %v", problem, err)
	}

	// 缺少的参数取默认值, 每次解析互不影响
	first, _, _ := withDefaults.ValidateMap(nil, map[string]any{"amount": 9})
	second, problem, err := withDefaults.ValidateMap(nil, map[string]any{})
	if err != nil || problem != nil {
		t.Fatalf("defaults: got %+v, %v", problem, err)
	}
	if *first.(*schemaParams) != (schemaParams{Currency: "chip", Amount: 9, Strict: true}) || *second.(*schemaParams) != (schemaParams{Currency: "chip", Amount: 5, Strict: true}) {
		t.Errorf("defaults: got %+v and %+v", first, second)
	}

	// 读取已保存的参数时忽略已删除的参数字段, 保存前不接受
	if decoded, err := plain.DecodeMap(map[string]any{"currency": "gold", "removed": true}, false); err != nil || decoded.(*schemaParams).Currency != "gold" {
		t.Errorf("lenient: got %+v, %v", decoded, err)
	}
	if _, err := plain.DecodeMap(map[string]any{"currency": "gold", "removed": true}, true); err == nil {
		t.Errorf("strict: expected an error")
	}
	if decoded, err := plain.DecodeMap(nil, true); err != nil || *decoded.(*schemaParams) != (schemaParams{}) {
		t.Errorf("nil map: got %+v, %v", decoded, err)
	}

	// 表单描述每次返回一份新的
	fields := plain.Fields()
	fields[0].Label = "changed"
	if plain.Fields()[0].Label != "货币" {
		t.Errorf("fields are shared")
	}
}
