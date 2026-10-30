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

package mission

const (
	CodeInvalid        = "任务代码需为 1-32 个小写字母、数字或下划线"
	CodeTaken          = "任务代码已存在"
	CodeImmutable      = "任务代码不能修改"
	CycleImmutable     = "任务的周期创建后不能修改, 请新建任务"
	NameInvalid        = "任务名称需为 1-64 个字符"
	DescriptionInvalid = "任务说明最多 255 个字符"
	CycleInvalid       = "任务周期需为每日、每周或成就"
	ScopeInvalid       = "统计范围格式不正确"
	MetricInvalid      = "统计指标不存在"
	TargetInvalid      = "目标需为 1-1000000000000"
	RewardEmpty        = "奖励至少包含一项货币或物品"
	RewardInvalid      = "奖励: %s"
	RequirementInvalid = "任务条件: %s"

	MissionNotFound   = "任务不存在"
	RequirementUnmet  = "未满足任务条件: %s"
	NotCompleted      = "任务还没有完成"
	AlreadyClaimed    = "本周期的奖励已领取"
	RewardUnavailable = "奖励中的货币或物品已停用, 暂时无法领取"
	RewardTooLarge    = "奖励数量超出上限, 无法领取"
)
