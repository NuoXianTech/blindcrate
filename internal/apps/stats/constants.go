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

// Package stats 玩家的统计数据, 按统计范围、周期和指标分别累加, 供条件检查、每日上限和排行榜使用
//
// 每个用户每个范围每个周期每个指标一行, 结算时在同一个事务中调用 Add 一次写入所有组合:
//   - 范围: 全部、某种模式、某张地图、某张地图的某种模式, 见 ScopeAll 和 Scope 开头的函数
//   - 周期: 累计、当天和当周, 见 PeriodTotal、PeriodDay 和 PeriodWeek; 日期按 schedule.time_zone 计算, 与定时任务一致; 赛季等周期见 RegisterPeriod
//   - 指标: 对局数、胜利次数等计数指标用 RegisterMetric 注册, 可以在条件中使用; 净收益按货币分开, 见 MetricProfit
//
// 新增一种统计数据只需注册指标并在结算时写入, 不需要改表; 练习局不写统计。
package stats

// ScopeAll 统计范围: 全部对局
const ScopeAll = "all"

// PeriodTotal 统计周期: 累计
const PeriodTotal = "total"

// 按天和按周统计周期的前缀, 见 PeriodDay 和 PeriodWeek
const (
	periodDayPrefix  = "day:"
	periodWeekPrefix = "week:"
)

// profitPrefix 净收益指标的前缀, 见 MetricProfit
const profitPrefix = "profit:"

// 通用的计数指标, 所有玩法结算时都会写入
const (
	MetricMatches = "matches" // 完成的对局数
	MetricWins    = "wins"    // 胜利次数, 名次为第 1 的对局数
)

// RecentDays 个人数据中按天展示的天数, 含今天; 不能超过 settings.StatsSettings.DayRetention 的下限
const RecentDays = 7

const (
	// maxScopeLength 统计范围的最大长度, 与 stats.scope 一致
	maxScopeLength = 64
	// maxPeriodLength 统计周期的最大长度, 与 stats.period 一致
	maxPeriodLength = 32
	// maxMetricLength 指标的最大长度, 与 stats.metric 一致
	maxMetricLength = 64
	// cleanupBatch 清理按天和按周统计及登记的表时每批删除的行数
	cleanupBatch = 5000
)
