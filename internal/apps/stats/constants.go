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

// Package stats 玩家的对局统计, 按统计范围和周期分别累加, 供条件检查、每日上限和排行榜使用
//
// 每个用户每个范围每个周期一行, 结算时在同一个事务中调用 Add 一次写入所有组合:
//   - 范围: 全部、某种模式、某张地图、某张地图的某种模式, 见 ScopeAll 和 Scope 开头的函数
//   - 周期: 累计和当天, 见 PeriodTotal 和 PeriodDay; 日期按 schedule.time_zone 计算, 与定时任务一致
//
// 练习局不写统计。
package stats

// ScopeAll 统计范围: 全部对局
const ScopeAll = "all"

// PeriodTotal 统计周期: 累计
const PeriodTotal = "total"

// maxScopeLength 统计范围的最大长度, 与 user_stats.scope 一致
const maxScopeLength = 64
