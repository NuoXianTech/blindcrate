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

// Package condition 通用条件: 地图入场、商城购买、活动等需要"满足什么才能做"的地方共用一套条件
//
// 条件只做检查, 没有副作用; 需要扣除的代价(入场费、门票)用 reward.Bundle 配置, 不在这里。
//   - 配置: 条件列表 []Condition 存进各模块的 JSONB 字段, 同一个列表中的条件需要全部满足; 保存前调用 Validate
//   - 展示: Evaluate 返回每个条件的达成情况和进度, 如 "完成 10 局排位(3/10)"
//   - 拦截: FirstUnmet 返回第一个未满足的条件, 由调用方决定如何拒绝, 如 match.Reject(p.Text)
//
// 新增一种条件类型: 定义参数结构体, 在 checkers.go 的 init 中调用 Register, 后台的条件编辑器根据 Types 自动生成表单。
// 统计范围的展示名称(如 "排位"、"码头仓库")由定义模式和地图的模块通过 RegisterScopeNamer 提供。
package condition

// 条件类型
const (
	TypeMinBalance    = "min_balance"    // 可用余额不少于
	TypeMinStat       = "min_stat"       // 累计的统计数据不少于, 如对局数、胜利次数
	TypeHasItem       = "has_item"       // 持有物品不少于, 不消耗
	TypeAccountAge    = "account_age"    // 注册天数不少于
	TypeEmailVerified = "email_verified" // 已验证邮箱
	TypeDailyLimit    = "daily_limit"    // 今天的统计数据少于
)

// maxConditions 一个条件列表最多的条件数
const maxConditions = 16
