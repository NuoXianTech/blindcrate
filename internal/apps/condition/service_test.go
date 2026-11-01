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
	"reflect"
	"testing"
)

type fieldsParams struct {
	Name  string `json:"name" label:"名称"`
	Kind  string `json:"kind,omitempty" label:"类型" help:"选一种" options:"a:甲,b:乙"`
	Item  uint64 `json:"item" label:"物品" input:"item"`
	Rate  int32  `json:"rate" label:"比例" unit:"bp"`
	Count int64  `json:"count" label:"数量"`
	On    bool   `json:"on" label:"开关"`
}

func TestFieldsOf(t *testing.T) {
	want := []Field{
		{Key: "name", Label: "名称", Type: "string", Input: "text"},
		{Key: "kind", Label: "类型", Help: "选一种", Type: "string", Input: "select", Options: []Option{{Value: "a", Label: "甲"}, {Value: "b", Label: "乙"}}},
		{Key: "item", Label: "物品", Type: "int", Input: "item"},
		{Key: "rate", Label: "比例", Type: "int", Input: "number", Unit: unitBasisPoints},
		{Key: "count", Label: "数量", Type: "int", Input: "number"},
		{Key: "on", Label: "开关", Type: "bool", Input: "switch"},
	}
	if got := FieldsOf[fieldsParams]("test"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestFieldsOfPanics(t *testing.T) {
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
	cases := map[string]func(){
		"not struct":   func() { FieldsOf[int]("test") },
		"float":        func() { FieldsOf[float]("test") },
		"int options":  func() { FieldsOf[intOptions]("test") },
		"bad options":  func() { FieldsOf[badOptions]("test") },
		"string unit":  func() { FieldsOf[stringUnit]("test") },
		"unknown unit": func() { FieldsOf[unknownUnit]("test") },
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
