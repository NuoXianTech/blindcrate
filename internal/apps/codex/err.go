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

package codex

const (
	CodeInvalid        = "收集组代码需为 1-32 个小写字母、数字或下划线"
	CodeTaken          = "收集组代码已存在"
	CodeImmutable      = "收集组代码不能修改"
	NameInvalid        = "收集组名称需为 1-64 个字符"
	DescriptionInvalid = "收集组说明最多 255 个字符"
	ItemsEmpty         = "收集组至少包含一种藏品"
	ItemsTooMany       = "收集组最多包含 200 种藏品"
	ItemsInvalid       = "收集组中有已删除或不是藏品的物品: %s"
	RewardEmpty        = "奖励至少包含一项货币或物品"
	RewardInvalid      = "奖励: %s"

	SetNotFound       = "收集组不存在"
	NotCompleted      = "还没有集齐收集组中的藏品"
	AlreadyClaimed    = "收集组的奖励已领取"
	RewardUnavailable = "奖励中的货币或物品已停用, 暂时无法领取"
	RewardTooLarge    = "奖励数量超出上限, 无法领取"
)
