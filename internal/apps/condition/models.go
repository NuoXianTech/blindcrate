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

import "encoding/json"

// Condition 一个条件, 存进各模块的 JSONB 配置
type Condition struct {
	Type   string          `json:"type"`                        // 条件类型, 见 constants.go
	Params json.RawMessage `json:"params" swaggertype:"object"` // 参数, 结构由条件类型决定
}

// Progress 用户对一个条件的达成情况
type Progress struct {
	Type    string `json:"type"`
	Met     bool   `json:"met"`
	Current int64  `json:"current"` // 当前值, 如已玩局数; 没有进度的条件为 0
	Target  int64  `json:"target"`  // 目标值; 没有进度的条件为 0
	Text    string `json:"text"`    // 展示给玩家的描述, 如 "完成 10 局排位(3/10)"
}

// Field 条件参数的描述, 后台据此自动生成表单
type Field struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Help    string   `json:"help"`
	Type    string   `json:"type"`              // string / bool / int
	Input   string   `json:"input"`             // text / switch / number, 或选择器 currency / item / scope / metric
	Options []Option `json:"options,omitempty"` // 选择器 metric 的可选值
}

// Option 选择器的一个可选值
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// TypeView 后台看到的条件类型
type TypeView struct {
	Type        string  `json:"type"`
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Fields      []Field `json:"fields"`
}
