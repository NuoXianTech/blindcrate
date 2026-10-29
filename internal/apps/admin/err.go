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

package admin

const (
	Forbidden          = "无权限"
	CannotOperateSelf  = "不能对自己执行该操作"
	BanUntilInvalid    = "封禁截止时间必须晚于当前时间"
	RemarkInvalid      = "备注需为 1-255 个字符"
	MailTestIncomplete = "请先保存 SMTP 服务器和发件人"
	MailTestFailed     = "测试邮件发送失败"
)

// 配置包
const (
	PackRequired           = "请选择配置包"
	PackTooLarge           = "配置包不能超过 %d MB"
	PackInvalid            = "配置包格式不正确"
	PackVersionUnsupported = "不支持格式版本为 %d 的配置包, 请升级后再导入"
	PackSectionInvalid     = "配置包中有未知的配置种类 %s"
	PackFileInvalid        = "配置包中的 %s 格式不正确"
	PackTooMany            = "配置包中的%s超过 %d 项"
	PackImagesTooMany      = "配置包中的图片超过 %d 张"
	PackKeyDuplicate       = "配置包中的%s %s 重复"
	PackNameInvalid        = "配置包名称需为 1-64 个字符, 作者最多 64 个字符"
	PackSectionsRequired   = "请至少选择一种配置"
	PackActionInvalid      = "处理方式不正确"
	PackRenameNotAllowed   = "%s不能改名导入"
	PackRenameInvalid      = "%s %s 的新代码为空、与原代码相同或与包中的其他配置重复"
	PackConflicts          = "有 %d 项配置无法导入, 请跳过或改名后再导入"
	PackImageMissing       = "配置包中缺少图片 %s"
	PackFieldInvalid       = "%s 不符合要求"
	PackItemDeleted        = "同 ID 的物品已删除, 不能导入"
	PackCodeDeleted        = "同代码的%s已删除, 请改名导入"
	PackCodeTaken          = "已有代码为 %s 的%s, 请换一个新代码"
	PackQualityCodeDiffers = "已有的等级 %d 品质代码为 %s, 与包中的不同"
	ImportNotFound         = "导入记录不存在"
)
