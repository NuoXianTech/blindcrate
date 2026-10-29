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

package season

const (
	SeasonNotFound = "赛季不存在"
	CodeInvalid    = "赛季代码需为 1-24 个小写字母、数字或下划线"
	CodeTaken      = "赛季代码已存在"
	NameInvalid    = "赛季名称需为 1-64 个字符"
	TimeInvalid    = "结束时间需晚于开始时间和当前时间"
	CarryInvalid   = "段位分继承比例需为 0-100"
	TimeOverlapped = "与其他赛季的时间重叠"
	StartLocked    = "赛季已开始, 不能修改开始时间和段位分继承比例"
	SeasonEnded    = "赛季已结束, 只能修改名称"
	DeleteStarted  = "只能删除未开始的赛季"
	EndNotActive   = "只能提前结束进行中的赛季"
)
