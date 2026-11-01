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

package bot

const (
	NameInvalid  = "昵称需为 1-32 个字符"
	NameTaken    = "昵称已存在"
	NameNotFound = "昵称不存在"
	NamesEmpty   = "至少填写一个昵称"
	NamesTooMany = "一次最多添加 200 个昵称"
	NamesInvalid = "第 %d 个昵称需为 1-32 个字符"

	CodeInvalid        = "风格代码需为 1-32 个小写字母、数字或下划线"
	CodeTaken          = "风格代码已存在, 已删除的风格的代码也不能再用"
	CodeImmutable      = "风格代码不能修改"
	PersonaNameInvalid = "风格名称需为 1-32 个字符"
	DescriptionInvalid = "风格说明最多 255 个字符"
	StrategyNotFound   = "策略不存在"
	ParamsInvalid      = "策略参数格式不正确"
	ParamsRuleInvalid  = "策略参数: %s"
	PersonaNotFound    = "风格不存在"
	PersonaInUse       = "风格正在被地图 %s 使用, 请先在地图中移除"
	PersonasDuplicated = "人机风格 %s 重复"
	PersonasMissing    = "人机风格 %s 不存在"
	PersonasBroken     = "人机风格 %s 的策略已不存在, 请先修改风格"
)
