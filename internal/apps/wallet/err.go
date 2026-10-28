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

package wallet

const (
	CurrencyNotFound    = "货币不存在"
	CurrencyDisabled    = "货币已停用"
	CurrencyCodeInvalid = "货币代码需为 1-32 个小写字母、数字或下划线"
	CurrencyNameInvalid = "货币名称需为 1-32 个字符"
	CurrencyTaken       = "货币代码已存在"
	CurrencyInUse       = "货币已被余额、物品、地图或掉落池使用, 不能修改小数位"
	CurrencyIsBase      = "基础货币不能停用, 请先在系统设置中更换基础货币"
	AmountInvalid       = "金额不合法"
	InsufficientBalance = "余额不足"
	InsufficientFrozen  = "冻结余额不足"
	IdempotencyConflict = "幂等键已被其他请求使用"
	DuplicateRequest    = "请求正在处理, 请勿重复提交"
)
