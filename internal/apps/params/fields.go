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
//   - options: 可选值("值:名称,值:名称"); 字符串显示为下拉框, 字符串或整数的列表显示为多选
//   - input: 输入方式, 字符串的 textarea、datetime(RFC 3339 的时间), 或一种可选值来源的名称(如 currency、item), 见 RegisterSource
//   - unit: 整数的单位, 可用的单位见 Vocabulary
//
// 其他 tag 由使用方定义, 需要在 Vocabulary.Tags 中声明(如设置组的 ref、secret、public)。
//
// 可选值来源由拥有数据的模块登记(货币、物品、品质、掉落池、统计范围等), 表单描述返回给后台前用 Resolve 填入当前的可选值,
// 后台只需要通用的下拉框: 新增一种来源不用改前端。结构化的参数(如一组奖励、一组条件)由定义它的模块用 RegisterKind 登记。
package params

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// Field 一个参数的描述, 后台据此自动生成表单
type Field struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Help     string   `json:"help"`
	Type     string   `json:"type"`              // string / bool / int / int_list / string_list, 或 RegisterKind 登记的类型
	Input    string   `json:"input"`             // text / textarea / datetime / password / select / suggest / multiselect / switch / number / list, 或 RegisterKind 登记的类型; Resolve 之前还可以是可选值来源的名称
	Options  []Option `json:"options,omitempty"` // 下拉框、多选和输入建议的可选值
	Unit     string   `json:"unit,omitempty"`    // 整数的单位: bp / second / millisecond
	Default  string   `json:"default,omitempty"` // 新建时的默认值, 由可选值来源给出
	Required bool     `json:"required"`          // binding 规则中有 required: 字符串不能为空, 列表不能为空
}

// Option 下拉框的一个可选值
type Option struct {
	Value    string `json:"value"`
	Label    string `json:"label"`
	Hint     string `json:"hint,omitempty"`     // 次要的说明, 如代码
	Quality  int16  `json:"quality,omitempty"`  // 按这个品质的颜色展示, 0 为不着色
	Disabled bool   `json:"disabled,omitempty"` // 不能新选, 已经选了的照常展示, 如已下架的物品
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
			Key:      strings.Split(sf.Tag.Get("json"), ",")[0],
			Label:    sf.Tag.Get("label"),
			Help:     sf.Tag.Get("help"),
			Input:    sf.Tag.Get("input"),
			Options:  ParseOptions(fieldName, sf.Tag.Get("options")),
			Unit:     sf.Tag.Get("unit"),
			Required: slices.Contains(strings.Split(sf.Tag.Get("binding"), ","), "required"),
		}
		switch kind := sf.Type.Kind(); {
		case kinds[sf.Type] != "":
			f.Type, f.Input = kinds[sf.Type], kinds[sf.Type]
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
		case kind == reflect.Slice && isInt(sf.Type.Elem().Kind()) && len(f.Options) > 0:
			f.Type, f.Input = "int_list", "multiselect"
		case kind == reflect.Slice && sf.Type.Elem().Kind() == reflect.String && len(f.Options) > 0:
			f.Type, f.Input = "string_list", "multiselect"
		case kind == reflect.Slice && isInt(sf.Type.Elem().Kind()) && v.IntLists:
			f.Type, f.Input = "int_list", "list"
		default:
			panic(fmt.Sprintf("params: %s: unsupported type %s", fieldName, sf.Type))
		}
		switch {
		case len(f.Options) > 0 && f.Type != "string" && f.Input != "multiselect":
			panic(fmt.Sprintf("params: %s: only strings and lists can have options", fieldName))
		case f.Unit != "" && (!slices.Contains(v.Units, f.Unit) || (f.Type != "int" && f.Type != "int_list")):
			panic(fmt.Sprintf("params: %s: unsupported unit %q", fieldName, f.Unit))
		}
		fields = append(fields, f)
	}
	return fields
}

// kinds 结构化的参数类型, Go 类型 -> 名称, 只在启动前写入
var kinds = map[reflect.Type]string{}

// RegisterKind 登记一种结构化的参数类型, 如一组奖励; 需要在启动前、用到它的参数结构体注册之前调用, 名称重复时 panic
//
// T 类型的字段在表单描述中 type 和 input 都是 name, 后台需要有这个名称对应的编辑器; 解析按 JSON, 校验由参数结构体的 Validate 完成。
func RegisterKind[T any](name string) {
	for _, existing := range kinds {
		if existing == name {
			panic(fmt.Sprintf("params: kind %s already registered", name))
		}
	}
	kinds[reflect.TypeFor[T]()] = name
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
