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
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// 本文件是条件组: 一个条件列表里的条件需要全部满足, 要表达"满足其中几个"或"都不满足"时, 把几个条件包进一个条件组;
// 条件组里不能再放条件组, 只能包一层

// anyOfParams 满足其中几个的参数
type anyOfParams struct {
	Conditions []Condition `json:"conditions" label:"条件" help:"最多 8 个, 不能再放条件组" binding:"required,min=1,max=8"`
	Count      int32       `json:"count" label:"至少满足几个" binding:"min=1,max=8"`
	Text       string      `json:"text" label:"展示的说明" help:"展示给玩家看, 留空时由里面的条件自动生成" binding:"max=64"`
}

// Validate 里面的条件都要合法, 至少满足的个数不能超过条件的个数
func (p *anyOfParams) Validate(tx *gorm.DB) (string, error) {
	if int(p.Count) > len(p.Conditions) {
		return NestedCountOver, nil
	}
	return validateNested(tx, p.Conditions)
}

func checkAnyOf(tx *gorm.DB, userID uint64, p *anyOfParams) (*Progress, error) {
	met, texts, err := evaluateNested(tx, userID, p.Conditions)
	if err != nil {
		return nil, err
	}
	target := int64(p.Count)
	text := p.Text
	switch {
	case text != "":
	case target == 1:
		text = strings.Join(texts, " 或 ")
	default:
		text = fmt.Sprintf("以下满足 %d 项(%d/%d): %s", target, min(met, target), target, strings.Join(texts, "; "))
	}
	return &Progress{Met: met >= target, Current: min(met, target), Target: target, Text: text}, nil
}

// noneOfParams 都不满足的参数
type noneOfParams struct {
	Conditions []Condition `json:"conditions" label:"条件" help:"最多 8 个, 不能再放条件组" binding:"required,min=1,max=8"`
	Text       string      `json:"text" label:"展示的说明" help:"展示给玩家看, 留空时由里面的条件自动生成" binding:"max=64"`
}

// Validate 里面的条件都要合法
func (p *noneOfParams) Validate(tx *gorm.DB) (string, error) {
	return validateNested(tx, p.Conditions)
}

func checkNoneOf(tx *gorm.DB, userID uint64, p *noneOfParams) (*Progress, error) {
	met, texts, err := evaluateNested(tx, userID, p.Conditions)
	if err != nil {
		return nil, err
	}
	text := p.Text
	switch {
	case text != "":
	case len(texts) == 1:
		text = "不能满足: " + texts[0]
	default:
		text = "以下都不能满足: " + strings.Join(texts, "; ")
	}
	return &Progress{Met: met == 0, Text: text}, nil
}

// validateNested 校验条件组里的条件, 返回不合法的原因; 里面不能再有条件组
func validateNested(tx *gorm.DB, conditions []Condition) (string, error) {
	for i := range conditions {
		if def, ok := definitions[conditions[i].Type]; ok && def.group {
			return fmt.Sprintf(NestedInvalid, i+1, NestedTooDeep), nil
		}
		reason, err := validateOne(tx, &conditions[i])
		if err != nil {
			return "", err
		}
		if reason != "" {
			return fmt.Sprintf(NestedInvalid, i+1, reason), nil
		}
	}
	return "", nil
}

// evaluateNested 检查条件组里的每个条件, 返回满足的个数和每个条件的描述
func evaluateNested(tx *gorm.DB, userID uint64, conditions []Condition) (int64, []string, error) {
	progress, err := Evaluate(tx, userID, conditions)
	if err != nil {
		return 0, nil, err
	}
	var met int64
	texts := make([]string, len(progress))
	for i, p := range progress {
		if p.Met {
			met++
		}
		texts[i] = p.Text
	}
	return met, texts, nil
}

// Rewrite 逐个改写条件参数中的字符串: fn 得到参数的输入方式(可选值来源的名称, 如 scope)和现在的值, 返回新的值; 条件组里的条件一起改写
//
// 用于配置包中被引用的配置改名导入时, 把条件里对它的引用改为新代码, 新增的条件类型不用再登记改名的处理;
// 类型不存在或参数解析不了的条件不变。
func Rewrite(conditions []Condition, fn func(input, value string) string) {
	rewrite(conditions, fn)
}

// rewrite 见 Rewrite, 返回有没有改动; 没有改动的条件保持原来的写法
func rewrite(conditions []Condition, fn func(input, value string) string) bool {
	rewritten := false
	for i := range conditions {
		def, ok := definitions[conditions[i].Type]
		if !ok {
			continue
		}
		var values map[string]json.RawMessage
		if json.Unmarshal(conditions[i].Params, &values) != nil {
			continue
		}
		changed := false
		for _, f := range def.schema.Fields() {
			var next any
			switch f.Type {
			case "string":
				var value string
				if json.Unmarshal(values[f.Key], &value) != nil {
					continue
				}
				renamed := fn(f.Input, value)
				if renamed == value {
					continue
				}
				next = renamed
			case KindConditions:
				var nested []Condition
				if json.Unmarshal(values[f.Key], &nested) != nil || !rewrite(nested, fn) {
					continue
				}
				next = nested
			default:
				continue
			}
			data, err := json.Marshal(next)
			if err != nil {
				continue
			}
			values[f.Key], changed = data, true
		}
		if !changed {
			continue
		}
		if data, err := json.Marshal(values); err == nil {
			conditions[i].Params, rewritten = data, true
		}
	}
	return rewritten
}
