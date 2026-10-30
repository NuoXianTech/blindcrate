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

// Package checkin 每日签到: 每天签到一次, 按连续签到的天数领取对应一天的奖励
//
//   - 奖励: 后台按天配置一个周期的奖励, 天数就是周期的长度, 连续签到超过周期时从第 1 天重新开始
//   - 连续: 前一天签到过时连续天数加 1, 否则从 1 开始; 不提供补签
//   - 日期: 按 schedule.time_zone 的日期, 签到记录的日期与统计的按天周期相同, 过了统计的保留天数后一起清理
//   - 统计: 每次签到给指标 checkins 加 1, 任务可以用它配置"本周签到 5 天"
package checkin

// BizCheckin 签到奖励的流水业务类型, biz_id 为签到的日期周期, 如 day:2026-10-30
const BizCheckin = "checkin"

// MetricCheckins 统计指标: 签到天数
const MetricCheckins = "checkins"

// 操作日志
const (
	TargetCheckinDays = "checkin_days" // target_id 为 0, 签到奖励整体保存

	ActionUpdateDays = "admin.checkin_days_update"
)

// MaxDays 一个签到周期最多的天数
const MaxDays = 31
