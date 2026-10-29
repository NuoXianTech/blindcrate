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

package effect

import "github.com/NuoXianTech/blindcrate/internal/apps/condition"

// TypeView 后台看到的效果类型
type TypeView struct {
	Code        string            `json:"code"`
	Scene       string            `json:"scene"`      // 场景, 见 Scene 开头的常量
	SceneName   string            `json:"scene_name"` // 场景的展示名称, 如 "对局中"
	Label       string            `json:"label"`
	Description string            `json:"description"`
	Fields      []condition.Field `json:"fields"` // 参数的表单描述, 与条件类型相同
}

// UseRequest 使用一个物品
type UseRequest struct {
	UserID         uint64
	ItemID         uint64
	BizID          string   // 业务单号, 由场景决定, 如对局 ID
	IdempotencyKey string   // 每次使用一个, 需带场景前缀保证全局唯一
	Remark         string   // 流水备注
	Scopes         []string // 统计范围, 如 stats.MatchScopes 的结果
}

// RefundRequest 退还用掉的物品
type RefundRequest struct {
	UserID         uint64
	Items          map[uint64]int64 // 物品 ID -> 数量
	BizID          string
	IdempotencyKey string
	Remark         string
}
