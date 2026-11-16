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
	"fmt"

	"gorm.io/gorm"
)

// Source 一种可选值来源, 如全部启用的货币; 参数字段用 input tag 引用它的名称
type Source struct {
	// Options 当前的可选值
	Options func(tx *gorm.DB) ([]Option, error)
	// Free 可以填可选值之外的值, 可选值只作为输入建议, 如统计范围
	Free bool
	// Default 新建时的默认值; 为空且 First 为 true 时取第一个可选值, 都没有时需要管理员自己选
	Default string
	First   bool
}

// sources 已登记的可选值来源, 按名称, 只在启动前写入
var sources = map[string]Source{}

// RegisterSource 登记一种可选值来源, 由拥有数据的模块在 init 中调用; 重复登记时 panic
func RegisterSource(name string, source Source) {
	if _, ok := sources[name]; ok {
		panic(fmt.Sprintf("params: source %s already registered", name))
	}
	sources[name] = source
}

// SourceOptions 一种来源当前的可选值, 来源没有登记时返回错误
func SourceOptions(tx *gorm.DB, name string) ([]Option, error) {
	source, ok := sources[name]
	if !ok {
		return nil, fmt.Errorf("params: source %s not registered", name)
	}
	return source.Options(tx)
}

// inputs 不需要来源的输入方式
var inputs = map[string]bool{
	"text": true, "textarea": true, "datetime": true, "password": true, "select": true,
	"multiselect": true, "switch": true, "number": true, "list": true,
}

// Resolve 把表单描述中引用可选值来源的字段换成通用的下拉框(select)或带建议的输入框(suggest), 并填入当前的可选值和默认值
//
// 直接修改 fields, 需要传入 Schema.Fields 返回的副本; 来源没有登记时返回错误。没有引用来源的字段不查库, tx 可以为 nil。
func Resolve(tx *gorm.DB, fields []Field) error {
	for i := range fields {
		f := &fields[i]
		if inputs[f.Input] || f.Input == f.Type {
			continue
		}
		source, ok := sources[f.Input]
		if !ok {
			return fmt.Errorf("params: field %s: source %s not registered", f.Key, f.Input)
		}
		options, err := source.Options(tx)
		if err != nil {
			return err
		}
		f.Options = options
		f.Default = source.Default
		if f.Default == "" && source.First && len(options) > 0 {
			f.Default = options[0].Value
		}
		if source.Free {
			f.Input = "suggest"
		} else {
			f.Input = "select"
		}
	}
	return nil
}

// Form 表单描述, 引用可选值来源的字段已填入当前的可选值, 见 Resolve
func (s *Schema) Form(tx *gorm.DB) ([]Field, error) {
	fields := s.Fields()
	if err := Resolve(tx, fields); err != nil {
		return nil, err
	}
	return fields, nil
}
