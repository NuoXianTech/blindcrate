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

package loot

const (
	PoolNotFound       = "掉落池不存在"
	PoolDisabled       = "掉落池已停用"
	PoolEmpty          = "掉落池中没有可抽取的物品"
	PoolCodeInvalid    = "代码需为 1-64 个小写字母、数字或下划线"
	PoolCodeTaken      = "代码已存在"
	PoolNameInvalid    = "名称需为 1-64 个字符"
	EntryTargetInvalid = "每个条目需要选择一种物品或货币"
	EntryRangeInvalid  = "最小数量不能大于最大数量"
	EntryTooMany       = "掉落池条目过多"
	RollTimesInvalid   = "抽取次数不合法"
)
