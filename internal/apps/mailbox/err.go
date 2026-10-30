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

package mailbox

const (
	SenderInvalid      = "发件人最多 32 个字符"
	TitleInvalid       = "标题需为 1-64 个字符"
	ContentInvalid     = "正文最多 2000 个字符"
	AttachmentInvalid  = "附件: %s"
	ExpireInvalid      = "有效期需为 1-365 天"
	RequirementInvalid = "收件条件: %s"
	RecipientsInvalid  = "收件人需为 1-1000 个用户"
	UsersNotFound      = "用户不存在: %s"

	MailNotFound          = "邮件不存在"
	MailExpired           = "邮件已过期"
	NoAttachments         = "邮件没有附件"
	AlreadyClaimed        = "附件已领取"
	AttachmentUnavailable = "附件中的货币或物品已停用, 暂时无法领取"
	AttachmentTooLarge    = "附件数量超出上限, 无法领取"
	UnclaimedAttachments  = "附件还没有领取, 不能删除"

	BroadcastNotFound = "全服邮件不存在"
	BroadcastRevoked  = "全服邮件已撤回"
)
