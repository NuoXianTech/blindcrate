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

package item

const (
	ItemNotFound    = "物品不存在"
	ItemDisabled    = "物品已下架"
	ItemNameInvalid = "物品名称需为 1-64 个字符"
	ItemNameTaken   = "物品名称已存在"

	CategoryNotFound = "物品类别不存在"

	QualityNotFound      = "品质不存在"
	QualityCodeInvalid   = "品质代码需为 1-32 个小写字母、数字或下划线"
	QualityNameInvalid   = "品质名称需为 1-32 个字符"
	QualityColorInvalid  = "颜色需为 #RRGGBB 格式"
	QualityEffectInvalid = "品质特效不存在"
	QualityTaken         = "品质等级或代码已存在"
	QualityInUse         = "还有物品使用该品质, 不能删除"
)
