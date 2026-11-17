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
	"bytes"
	"encoding/json"
	"reflect"

	"gorm.io/gorm"
)

// Validator 参数结构体的指针可以实现的接口, 保存时做需要查库的校验, 如货币是否存在
type Validator interface {
	// Validate 返回非空的 reason 表示参数不合法, reason 展示给管理员; err 为查库等内部错误
	Validate(tx *gorm.DB) (reason string, err error)
}

// Problem 参数不合法的原因
type Problem struct {
	Malformed bool   // 格式不对: 不是对象、类型不符或有未知的参数; 此时 Reason 为空, 由使用方给出提示
	Reason    string // 不满足规则的原因, 可以直接展示给管理员
}

// Schema 一种注册的参数结构体: 表单描述, 以及解析和校验; 由 New 生成, 只读, 可以并发使用
type Schema struct {
	fields []Field
	fresh  func() any // 返回新的参数结构体的指针, 已填好默认值
}

// New 为参数结构体 P 生成 Schema, 需要在启动前调用; P 不是结构体或有 v 不支持的写法时 panic, name 只用于提示
//
// defaults 为缺少的参数取的值, 为 nil 时取零值; 默认值按值复制, P 中有切片时各次解析共用默认值的底层数组, 不要修改解析结果中的切片。
func New[P any](name string, v Vocabulary, defaults *P) *Schema {
	s := &Schema{fields: Describe(reflect.TypeFor[P](), name, v)}
	s.fresh = func() any {
		target := new(P)
		if defaults != nil {
			*target = *defaults
		}
		return target
	}
	return s
}

// Fields 表单描述, 每次返回一份新的, 可以修改; 返回给后台的用 Form
func (s *Schema) Fields() []Field {
	fields := make([]Field, len(s.fields))
	copy(fields, s.fields)
	return fields
}

// Decode 把 JSON 解析成参数结构体, 返回它的指针; raw 为空或 null 时全部取默认值
//
// strict 时不接受未知的参数, 保存前使用; 读取已保存的参数时不要 strict, 之后删除的参数字段会被忽略。
func (s *Schema) Decode(raw json.RawMessage, strict bool) (any, error) {
	target := s.fresh()
	if len(raw) == 0 {
		return target, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if strict {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(target); err != nil {
		return nil, err
	}
	return target, nil
}

// DecodeMap 同 Decode, 参数为已解析的 JSON 对象
func (s *Schema) DecodeMap(values map[string]any, strict bool) (any, error) {
	raw, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return s.Decode(raw, strict)
}

// Validate 保存前校验参数: 严格解析, 按 binding 规则校验, 参数结构体实现了 Validator 时再做查库的校验
//
// 合法时返回参数结构体的指针; 不合法时返回 *Problem; err 为查库等内部错误。
func (s *Schema) Validate(tx *gorm.DB, raw json.RawMessage) (any, *Problem, error) {
	decoded, err := s.Decode(raw, true)
	if err != nil {
		return nil, &Problem{Malformed: true}, nil
	}
	if err := ValidateStruct(decoded); err != nil {
		return nil, &Problem{Reason: err.Error()}, nil
	}
	if validator, ok := decoded.(Validator); ok {
		reason, err := validator.Validate(tx)
		if err != nil {
			return nil, nil, err
		}
		if reason != "" {
			return nil, &Problem{Reason: reason}, nil
		}
	}
	return decoded, nil, nil
}

// ValidateMap 同 Validate, 参数为已解析的 JSON 对象
func (s *Schema) ValidateMap(tx *gorm.DB, values map[string]any) (any, *Problem, error) {
	raw, err := json.Marshal(values)
	if err != nil {
		return nil, &Problem{Malformed: true}, nil
	}
	return s.Validate(tx, raw)
}

// Normalize 参数结构体转回 JSON 对象: 每个参数都有, 数字统一为 float64, 与从数据库读出的一致, 便于保存和比较
func Normalize(decoded any) (map[string]any, error) {
	raw, err := json.Marshal(decoded)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
