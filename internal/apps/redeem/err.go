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

package redeem

const (
	NameInvalid = "批次名称需为 1-64 个字符"
	CodeInvalid = "兑换码需为 4-32 个字母、数字或汉字"
	CodeTaken   = "兑换码已存在"

	IdempotencyConflict = "幂等键已被其他请求使用"
	CountInvalid        = "生成数量需为 1-10000"
	MaxUsesInvalid      = "每个兑换码的可使用次数需为 0-1000000"
	UserLimitInvalid    = "每人可兑换的个数需为 0-10000"
	ExpiresInvalid      = "到期时间需晚于当前时间"
	RewardEmpty         = "奖励至少包含一项货币或物品"
	RewardInvalid       = "奖励: %s"
	RequirementInvalid  = "使用条件: %s"

	BatchNotFound     = "兑换码批次不存在"
	CodeNotFound      = "兑换码不存在或已失效"
	CodeExpired       = "兑换码已过期"
	CodeUsedUp        = "兑换码已被兑换完"
	AlreadyRedeemed   = "你已经兑换过这个兑换码"
	UserLimitReached  = "这一批兑换码每人最多兑换 %d 个"
	RequirementUnmet  = "未满足使用条件: %s"
	RewardUnavailable = "奖励中的货币或物品已停用, 暂时无法兑换"
	RewardTooLarge    = "奖励数量超出上限, 无法兑换"
)
