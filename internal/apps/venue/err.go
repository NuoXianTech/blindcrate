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

package venue

const (
	VenueNotFound      = "地图不存在"
	VenueDisabled      = "地图已关闭"
	ModeDisabled       = "该地图的这个模式已关闭"
	CodeInvalid        = "代码需为 1-24 个小写字母、数字或下划线"
	CodeTaken          = "代码已被使用, 已删除的地图的代码也不能再用"
	CodeImmutable      = "地图代码创建后不能修改"
	NameInvalid        = "名称需为 1-64 个字符"
	EntryCostInvalid   = "入场代价的数量需大于 0"
	EntryCostTooMany   = "入场代价最多 8 项"
	RequirementInvalid = "入场条件: %s"
	RuleInvalid        = "%s: %s"
	SlotRangeInvalid   = "第 %d 档: 最多次数不能少于最少次数"
	SlotPoolInvalid    = "第 %d 档: %s"
	SlotPoolHasMoney   = "掉落池中有货币条目, 仓库只能放物品"
	SlotNotCollectible = "第 %d 档: 掉落池中的 %s 不是藏品, 仓库只抽藏品"
	SlotsEmpty         = "仓库至少要能抽出一件藏品"
	WarehouseNotFit    = "仓库摆不下抽到的藏品"
	FailureRateTooHigh = "仓库摆放失败率为 %.2f%%, 不能超过 1%%, 请加大格子或减少件数"
	MaxCarryInvalid    = "带入资金上限不能低于起拍价"
	QuickWinInvalid    = "速胜倍率需要 %d 个, 即轮数减 1"
	BotWeightsInvalid  = "人机等级的权重不能全为 0"
	ConsumableNotFound = "可用道具中的物品 %d 不存在"
	ConsumableUnusable = "可用道具中的 %s 不能在对局中使用"
)
