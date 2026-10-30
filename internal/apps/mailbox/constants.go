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

// Package mailbox 游戏内邮件: 系统和管理员发给玩家的通知, 可以带附件
//
//   - 邮件: 每个收件人一行, 附件为一组货币和物品, 领取后发放, 见 Mail 和 Claim
//   - 来源: 每封邮件有来源和来源单号, 同一来源单号对同一用户只写一封, 重复发送不会重复, 见 RegisterSource
//   - 全服邮件: 发送时只写一条记录, 玩家打开信箱时按收件条件展开成自己的邮件, 见 Broadcast 和 Expand
//   - 其他模块在自己的事务中调用 Send 写入邮件, 提交后调用 Sent.Push 通知收件人
//   - 清理: 每天删除过期很久的邮件, 并把超出保留封数的旧邮件移出信箱, 见 HandleCleanup
package mailbox

// 早期定义的邮件来源, 其他模块在自己的 init 中用 RegisterSource 登记
const (
	SourceAdmin     = "admin"     // 管理员发给指定用户, 来源单号为发送时的幂等键
	SourceBroadcast = "broadcast" // 全服邮件展开, 来源单号为全服邮件 ID
)

// BizClaim 领取附件的流水业务类型, biz_id 为邮件 ID
const BizClaim = "mail_claim"

// 推送给客户端的消息类型
const (
	MessageNew = "mailbox:new" // 有新邮件, data 为空, 客户端重新获取未读数; 全服邮件推送给全部在线用户
)

// TypeCleanup 定时清理邮件的任务类型
const TypeCleanup = "mailbox:cleanup"

// cleanupCron 清理任务默认的 cron, 可以在配置 schedule.crons 中覆盖
const cleanupCron = "20 4 * * *"

// 操作日志
const (
	TargetMail      = "mail"           // 发给指定用户时 target_id 固定为 0, 收件人见 detail.user_ids
	TargetBroadcast = "mail_broadcast" // target_id 为全服邮件 ID

	ActionSend      = "admin.mail_send"
	ActionBroadcast = "admin.mail_broadcast"
	ActionRevoke    = "admin.mail_revoke"
)

// 邮件的限制
const (
	maxSender      = 32   // 发件人的最大字符数
	maxTitle       = 64   // 标题的最大字符数
	maxContent     = 2000 // 正文的最大字符数
	MaxExpireDays  = 365  // 有效期的最大天数
	MaxRecipients  = 1000 // 一次发给指定用户的最多人数
	maxClaimAll    = 100  // 一键领取一次最多处理的邮件数
	cleanupBatch   = 1000 // 清理时每批删除的行数
	defaultSender  = "系统"
	maxSourceIDLen = 64 // 来源单号的最大长度
)
