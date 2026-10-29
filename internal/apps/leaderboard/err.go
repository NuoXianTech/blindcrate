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

package leaderboard

const (
	BoardNotFound      = "排行榜不存在"
	CodeInvalid        = "代码需为 1-64 个小写字母、数字或下划线"
	CodeTaken          = "代码已存在"
	NameInvalid        = "名称需为 1-64 个字符"
	MetricInvalid      = "统计数据不存在"
	ScopeInvalid       = "统计范围格式不正确"
	PeriodInvalid      = "统计周期不合法"
	SizeInvalid        = "上榜人数需为 10-500"
	MinMatchesInvalid  = "上榜最少局数不能为负数"
	RatingBoardInvalid = "段位分榜单的统计范围需为全部对局, 统计周期需为赛季"
)
