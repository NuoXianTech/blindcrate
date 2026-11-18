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

const (
	TooManyConditions = "条件不能超过 16 个"
	ConditionInvalid  = "第 %d 个条件: %s"
	TypeNotFound      = "条件类型不存在"
	ParamsInvalid     = "条件参数格式不正确"
	ScopeInvalid      = "统计范围格式不正确"
	MetricNotFound    = "统计数据不存在"
	PeriodNotFound    = "统计周期不存在"

	NestedInvalid   = "组内第 %d 个条件: %s"
	NestedTooDeep   = "条件组里不能再放条件组"
	NestedCountOver = "至少满足的个数不能超过条件的个数"

	WindowEmpty       = "至少填写一项时间限制"
	WindowTimeInvalid = "开始时间和结束时间需为有效的时间"
	WindowTimeOrder   = "结束时间需晚于开始时间"
	WindowClockPair   = "每天开始和每天结束需要同时填写"
	WindowClockFormat = "每天的时间需为 时:分, 如 19:00, 每天结束最晚为 24:00"
	WindowClockSame   = "每天开始和每天结束不能相同"
)
