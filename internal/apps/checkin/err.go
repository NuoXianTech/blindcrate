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

package checkin

const (
	DaysInvalid      = "签到天数需为 1-31 天"
	DayInvalid       = "签到的天数需为 1-31"
	DayGap           = "第 %d 天之前还有没配置的天数"
	DayRewardInvalid = "第 %d 天的奖励: %s"
	DayRewardEmpty   = "第 %d 天的奖励至少包含一项货币或物品"

	CheckinDisabled   = "签到暂未开放"
	DaysEmpty         = "签到还没有配置奖励"
	AlreadyCheckedIn  = "今天已经签到过了"
	RewardUnavailable = "奖励中的货币或物品已停用, 暂时无法签到"
	RewardTooLarge    = "奖励数量超出上限, 无法签到"
)
