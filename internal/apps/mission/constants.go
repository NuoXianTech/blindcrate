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

// Package mission 任务: 在一段时间内某个统计指标达到目标后领取奖励
//
//   - 周期: 每日任务和每周任务按 schedule.time_zone 的日期和 ISO 周重置, 成就只能领取一次, 见 Cycle 开头的常量
//   - 进度: 直接读取 stats 中对应范围、周期和指标的值, 不另外计数; 新增一种任务目标只需注册统计指标
//   - 领取: 进度达到目标且满足任务条件时领取, 同一周期内只能领取一次, 见 Claim
package mission

// 任务的周期
const (
	CycleDaily  = "daily"  // 每日任务, 进度为当天的统计
	CycleWeekly = "weekly" // 每周任务, 进度为本周的统计, 周一开始
	CycleOnce   = "once"   // 成就, 进度为累计统计, 只能领取一次
)

// cycles 全部周期, 按玩家页面的页签顺序, 见 Cycles
var cycles = []Cycle{
	{CycleDaily, "每日"},
	{CycleWeekly, "每周"},
	{CycleOnce, "成就"},
}

// BizReward 领取任务奖励的流水业务类型, biz_id 为任务 ID
const BizReward = "mission_reward"

// 操作日志
const (
	TargetMission = "mission" // target_id 为任务 ID

	ActionCreate = "admin.mission_create"
	ActionUpdate = "admin.mission_update"
	ActionDelete = "admin.mission_delete"
)

// 任务的限制
const (
	maxNameLength        = 64
	maxDescriptionLength = 255
	MaxTarget            = 1_000_000_000_000 // 目标值的上限
)
