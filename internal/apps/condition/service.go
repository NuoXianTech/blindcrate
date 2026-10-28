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
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"gorm.io/gorm"
)

// ParamsValidator 参数结构体可以实现的接口, 保存时做需要查库的校验, 如货币是否存在
type ParamsValidator interface {
	// Validate 返回非空的 reason 表示参数不合法, reason 展示给管理员; err 为查库等内部错误
	Validate(tx *gorm.DB) (reason string, err error)
}

// InvalidError 条件配置不合法, Error 返回的提示可以直接展示
type InvalidError struct {
	Index  int // 第几个条件, 从 1 开始; 0 表示整个列表不合法, 如条件过多
	Reason string
}

func (e *InvalidError) Error() string {
	if e.Index == 0 {
		return e.Reason
	}
	return fmt.Sprintf(ConditionInvalid, e.Index, e.Reason)
}

// ScopeNamer 返回统计范围的展示名称, 如 "排位"、"码头仓库"
type ScopeNamer func(tx *gorm.DB, scope string) (string, error)

// definition 一种条件类型, 由 Register 生成
type definition struct {
	label       string
	description string
	fields      []Field
	decode      func(raw json.RawMessage, strict bool) (any, error) // 返回参数结构体的指针
	check       func(tx *gorm.DB, userID uint64, params any) (*Progress, error)
}

var (
	// definitions 已注册的条件类型, 只在启动前写入
	definitions = map[string]*definition{}
	// typeOrder 条件类型的注册顺序, 后台按这个顺序展示
	typeOrder []string
	// scopeNamers 统计范围类别 -> 展示名称, 只在启动前写入
	scopeNamers = map[string]ScopeNamer{}
)

// Register 注册一种条件类型, 需要在启动前调用; 类型为空、过长、重复或参数结构体有不支持的字段时 panic
//
// P 为参数结构体, 字段 tag 与设置组一致(json、label、help、binding), 另可用 input 指定选择器(currency、item、scope、metric);
// P 的指针可以实现 ParamsValidator。check 只读, 不能有副作用, 返回的 Progress 不需要填 Type。
func Register[P any](conditionType, label, description string, check func(tx *gorm.DB, userID uint64, params *P) (*Progress, error)) {
	if conditionType == "" || len(conditionType) > 32 {
		panic(fmt.Sprintf("condition: invalid type %q", conditionType))
	}
	if _, ok := definitions[conditionType]; ok {
		panic(fmt.Sprintf("condition: type %s already registered", conditionType))
	}
	definitions[conditionType] = &definition{
		label:       label,
		description: description,
		fields:      fieldsOf[P](conditionType),
		decode: func(raw json.RawMessage, strict bool) (any, error) {
			params := new(P)
			if len(raw) == 0 {
				return params, nil
			}
			dec := json.NewDecoder(bytes.NewReader(raw))
			if strict {
				dec.DisallowUnknownFields()
			}
			if err := dec.Decode(params); err != nil {
				return nil, err
			}
			return params, nil
		},
		check: func(tx *gorm.DB, userID uint64, params any) (*Progress, error) {
			return check(tx, userID, params.(*P))
		},
	}
	typeOrder = append(typeOrder, conditionType)
}

// fieldsOf 根据参数结构体的 tag 生成表单描述
func fieldsOf[P any](conditionType string) []Field {
	typ := reflect.TypeFor[P]()
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("condition: params of %s must be a struct", conditionType))
	}
	fields := []Field{}
	for i := range typ.NumField() {
		sf := typ.Field(i)
		f := Field{
			Key:   strings.Split(sf.Tag.Get("json"), ",")[0],
			Label: sf.Tag.Get("label"),
			Help:  sf.Tag.Get("help"),
			Input: sf.Tag.Get("input"),
		}
		switch sf.Type.Kind() {
		case reflect.String:
			f.Type = "string"
			if f.Input == "" {
				f.Input = "text"
			}
		case reflect.Bool:
			f.Type, f.Input = "bool", "switch"
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint64:
			f.Type = "int"
			if f.Input == "" {
				f.Input = "number"
			}
		default:
			panic(fmt.Sprintf("condition: %s.%s: unsupported type %s", conditionType, sf.Name, sf.Type))
		}
		fields = append(fields, f)
	}
	return fields
}

// RegisterScopeNamer 为一类统计范围注册展示名称, 需要在启动前调用
//
// prefix 为统计范围中第一个冒号之前的部分, 如 mode、venue; 没有注册的类别直接展示统计范围本身。
func RegisterScopeNamer(prefix string, namer ScopeNamer) {
	if _, ok := scopeNamers[prefix]; ok {
		panic(fmt.Sprintf("condition: scope namer %s already registered", prefix))
	}
	scopeNamers[prefix] = namer
}

// Types 全部条件类型, 按注册顺序; 选择器 metric 的可选值为已注册的计数指标
func Types() []TypeView {
	var metricOptions []Option
	for _, m := range stats.Metrics() {
		metricOptions = append(metricOptions, Option{Value: m.Key, Label: m.Label})
	}
	views := make([]TypeView, 0, len(typeOrder))
	for _, conditionType := range typeOrder {
		def := definitions[conditionType]
		fields := slices.Clone(def.fields)
		for i := range fields {
			if fields[i].Input == "metric" {
				fields[i].Options = metricOptions
			}
		}
		views = append(views, TypeView{Type: conditionType, Label: def.label, Description: def.description, Fields: fields})
	}
	return views
}

// Validate 保存配置前校验条件列表
//
// 配置不合法时返回 *InvalidError, 提示可以直接展示; 其他错误为查库失败等内部错误。
func Validate(tx *gorm.DB, conditions []Condition) error {
	if len(conditions) > maxConditions {
		return &InvalidError{Reason: TooManyConditions}
	}
	for i := range conditions {
		reason, err := validateOne(tx, &conditions[i])
		if err != nil {
			return err
		}
		if reason != "" {
			return &InvalidError{Index: i + 1, Reason: reason}
		}
	}
	return nil
}

// validateOne 校验一个条件, 返回不合法的原因
func validateOne(tx *gorm.DB, c *Condition) (string, error) {
	def, ok := definitions[c.Type]
	if !ok {
		return TypeNotFound, nil
	}
	params, err := def.decode(c.Params, true)
	if err != nil {
		return ParamsInvalid, nil
	}
	if err := settings.ValidateStruct(params); err != nil {
		return err.Error(), nil
	}
	if validator, ok := params.(ParamsValidator); ok {
		return validator.Validate(tx)
	}
	return "", nil
}

// Evaluate 检查用户对每个条件的达成情况, 按条件的顺序返回, 用于展示进度
func Evaluate(tx *gorm.DB, userID uint64, conditions []Condition) ([]Progress, error) {
	result := make([]Progress, 0, len(conditions))
	for i := range conditions {
		progress, err := evaluateOne(tx, userID, &conditions[i])
		if err != nil {
			return nil, err
		}
		result = append(result, *progress)
	}
	return result, nil
}

// FirstUnmet 返回第一个未满足的条件, 全部满足时返回 nil, 用于入场等需要拦截的地方
func FirstUnmet(tx *gorm.DB, userID uint64, conditions []Condition) (*Progress, error) {
	for i := range conditions {
		progress, err := evaluateOne(tx, userID, &conditions[i])
		if err != nil {
			return nil, err
		}
		if !progress.Met {
			return progress, nil
		}
	}
	return nil, nil
}

// evaluateOne 检查一个条件; 配置在保存时已校验, 这里只解析参数, 已删除的参数字段忽略
func evaluateOne(tx *gorm.DB, userID uint64, c *Condition) (*Progress, error) {
	def, ok := definitions[c.Type]
	if !ok {
		return nil, fmt.Errorf("condition: unknown type %s", c.Type)
	}
	params, err := def.decode(c.Params, false)
	if err != nil {
		return nil, fmt.Errorf("condition: decode params of %s failed: %w", c.Type, err)
	}
	progress, err := def.check(tx, userID, params)
	if err != nil {
		return nil, err
	}
	progress.Type = c.Type
	return progress, nil
}

// scopeName 统计范围的展示名称, 全部对局为空字符串
func scopeName(tx *gorm.DB, scope string) (string, error) {
	if scope == stats.ScopeAll {
		return "", nil
	}
	prefix, _, _ := strings.Cut(scope, ":")
	namer, ok := scopeNamers[prefix]
	if !ok {
		return scope, nil
	}
	name, err := namer(tx, scope)
	if err != nil {
		return "", err
	}
	if name == "" {
		return scope, nil
	}
	return name, nil
}
