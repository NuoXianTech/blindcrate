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

package upload

const (
	PurposeInvalid   = "图片用途不正确"
	FileRequired     = "请选择要上传的图片"
	FileTooLarge     = "图片不能超过 %d KB"
	FormatNotAllowed = "只支持 %s 格式的图片"
	FormatDisabled   = "暂不支持上传图片"
	ImageInvalid     = "图片已损坏或无法识别"
	ImageTooLarge    = "图片尺寸过大, 最多 %d 万像素"
	AvatarDisabled   = "暂不允许上传头像"
	AdminRequired    = "只有管理员可以上传该用途的图片"
)

// InvalidError 图片不符合要求, Reason 为给用户看的原因
type InvalidError struct {
	Reason string
}

func (e *InvalidError) Error() string { return e.Reason }

// ForbiddenError 没有上传该用途图片的权限, Reason 为给用户看的原因
type ForbiddenError struct {
	Reason string
}

func (e *ForbiddenError) Error() string { return e.Reason }
