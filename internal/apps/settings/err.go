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

package settings

const (
	GroupNotFound      = "设置组不存在"
	GroupNotImportable = "设置组不存在或不能导入"
	ValueInvalid       = "设置内容格式不正确"
	SecretKeyMissing   = "未配置 app.secret_key, 无法保存密钥类设置"
	SecretBroken       = "%s无法解密, 请重新填写或清空后再保存"
	OptionInvalid      = "%s不是可选的值"
	VersionConflict    = "设置已被其他人修改, 请刷新后再保存"
	RevisionNotFound   = "版本不存在"
	RevisionBroken     = "版本 %d 中的%s无法解密, 不能回滚"
	QueryFailed        = "获取设置失败"
	SaveFailed         = "保存设置失败"
)
