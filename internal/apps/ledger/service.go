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

// Package ledger 余额和背包流水共用的业务类型登记
//
// 每种会变动余额或背包的业务在自己的模块中定义业务类型代码, 并在 init 中用 RegisterBizType 登记名称;
// wallet.Change 和 inventory.Change 只接受登记过的业务类型, 前端的流水列表和筛选从接口读取名称。
package ledger

import (
	"fmt"
	"regexp"
	"slices"
)

// 流水的种类
const (
	KindWallet    = "wallet"    // 余额流水
	KindInventory = "inventory" // 背包流水
)

// MaxBizTypeLength 业务类型的最大长度, 与流水表的 biz_type 一致
const MaxBizTypeLength = 32

// bizTypePattern 业务类型的格式
var bizTypePattern = regexp.MustCompile(`^[a-z0-9_]+$`)

// BizType 一种业务类型
type BizType struct {
	Code  string `json:"code"`
	Label string `json:"label"` // 流水中展示的名称, 如 "商城购买"
}

// bizTypes 种类 -> 已登记的业务类型, 按登记顺序, 只在启动前写入
var bizTypes = map[string][]BizType{}

// RegisterBizType 登记一种业务类型, 需要在启动前调用; 种类未知、格式不正确或重复时 panic
func RegisterBizType(kind, code, label string) {
	if kind != KindWallet && kind != KindInventory {
		panic(fmt.Sprintf("ledger: unknown kind %q", kind))
	}
	if len(code) > MaxBizTypeLength || !bizTypePattern.MatchString(code) || label == "" {
		panic(fmt.Sprintf("ledger: invalid biz type %q", code))
	}
	if Registered(kind, code) {
		panic(fmt.Sprintf("ledger: biz type %s/%s already registered", kind, code))
	}
	bizTypes[kind] = append(bizTypes[kind], BizType{Code: code, Label: label})
}

// Registered 业务类型是否已登记
func Registered(kind, code string) bool {
	return slices.ContainsFunc(bizTypes[kind], func(t BizType) bool { return t.Code == code })
}

// BizTypes 一种流水的全部业务类型, 按登记顺序
func BizTypes(kind string) []BizType {
	return slices.Clone(bizTypes[kind])
}
