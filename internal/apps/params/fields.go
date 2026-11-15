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

// Package params 带标签的参数结构体: 设置组、条件类型、物品效果、人机策略等都用一个结构体定义参数,
// 由字段的 tag 生成后台表单的描述, 并按同一套规则解析和校验
//
// 支持的 tag:
//   - json: 参数的 key
//   - label、help: 表单中的名称和说明, 校验失败的提示也使用 label
//   - binding: 校验规则, 写法与 gin 的请求参数校验一致, 见 ValidateStruct
//   - options: 字符串的可选值("值:名称,值:名称"), 显示为下拉框
//   - input: 输入方式, 字符串的 textarea, 或前端的选择器(currency、item、quality、scope、metric、pool)
//   - unit: 整数的单位, 可用的单位见 Vocabulary
//
// 其他 tag 由使用方定义, 需要在 Vocabulary.Tags 中声明(如设置组的 ref、secret、public)。
package params

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// Field 一个参数的描述, 后台据此自动生成表单
type Field struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Help    string   `json:"help"`
	Type    string   `json:"type"`              // string / bool / int / int_list
	Input   string   `json:"input"`             // text / textarea / password / select / switch / number / list, 或选择器 currency / item / quality / scope / metric / pool
	Options []Option `json:"options,omitempty"` // 下拉框和部分选择器的可选值
	Unit    string   `json:"unit,omitempty"`    // 整数的单位: bp / second / millisecond
}

// Option 下拉框的一个可选值
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// 整数的单位
const (
	UnitBasisPoints = "bp" // 基点, 后台按百分比输入
	UnitSecond      = "second"
	UnitMillisecond = "millisecond"
)

// Vocabulary 一类参数结构体可以使用的写法, 超出的写法在 Describe 时 panic
type Vocabulary struct {
	Units    []string // 可用的单位
	IntLists bool     // 是否支持整数列表
	Tags     []string // 使用方自己处理的 tag, 见 reservedTags
}

// Plain 条件类型、物品效果、人机策略的参数: 字符串、布尔和整数, 单位只有基点
var Plain = Vocabulary{Units: []string{UnitBasisPoints}}

// reservedTags 只有部分使用方支持的 tag, 没有在 Vocabulary.Tags 中声明时不能使用, 避免写了却不生效
var reservedTags = []string{"ref", "secret", "public"}

// Describe 根据参数结构体 typ 的 tag 生成表单描述, 每个字段一项, 顺序与结构体相同
//
// name 只用于出错时的提示; typ 不是结构体, 或有不支持的字段类型、tag 时 panic, 需要在启动前调用。
func Describe(typ reflect.Type, name string, v Vocabulary) []Field {
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("params: %s must be a struct", name))
	}
	fields := make([]Field, 0, typ.NumField())
	for i := range typ.NumField() {
		sf := typ.Field(i)
		fieldName := name + "." + sf.Name
		for _, tag := range reservedTags {
			if _, ok := sf.Tag.Lookup(tag); ok && !slices.Contains(v.Tags, tag) {
				panic(fmt.Sprintf("params: %s: tag %s is not supported here", fieldName, tag))
			}
		}
		f := Field{
			Key:     strings.Split(sf.Tag.Get("json"), ",")[0],
			Label:   sf.Tag.Get("label"),
			Help:    sf.Tag.Get("help"),
			Input:   sf.Tag.Get("input"),
			Options: ParseOptions(fieldName, sf.Tag.Get("options")),
			Unit:    sf.Tag.Get("unit"),
		}
		switch kind := sf.Type.Kind(); {
		case kind == reflect.String:
			f.Type = "string"
			switch {
			case len(f.Options) > 0:
				f.Input = "select"
			case f.Input == "":
				f.Input = "text"
			}
		case kind == reflect.Bool:
			f.Type, f.Input = "bool", "switch"
		case isInt(kind):
			f.Type = "int"
			if f.Input == "" {
				f.Input = "number"
			}
		case kind == reflect.Slice && isInt(sf.Type.Elem().Kind()) && v.IntLists:
			f.Type, f.Input = "int_list", "list"
		default:
			panic(fmt.Sprintf("params: %s: unsupported type %s", fieldName, sf.Type))
		}
		switch {
		case len(f.Options) > 0 && f.Type != "string":
			panic(fmt.Sprintf("params: %s: only strings can have options", fieldName))
		case f.Unit != "" && (!slices.Contains(v.Units, f.Unit) || (f.Type != "int" && f.Type != "int_list")):
			panic(fmt.Sprintf("params: %s: unsupported unit %q", fieldName, f.Unit))
		}
		fields = append(fields, f)
	}
	return fields
}

// isInt 是否为支持的整数类型
func isInt(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint64:
		return true
	}
	return false
}

// ParseOptions 解析 options tag, 格式为 "值:名称,值:名称"; name 只用于出错时的提示, 格式不对时 panic, 需要在启动前调用
func ParseOptions(name, tag string) []Option {
	if tag == "" {
		return nil
	}
	var options []Option
	for _, part := range strings.Split(tag, ",") {
		value, label, ok := strings.Cut(part, ":")
		if !ok || value == "" || label == "" {
			panic(fmt.Sprintf("params: %s: invalid options %q", name, tag))
		}
		options = append(options, Option{Value: value, Label: label})
	}
	return options
}
