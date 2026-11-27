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

// Package redeem 兑换码: 管理员生成兑换码, 玩家输入后领取奖励
//
//   - 批次: 一次生成的兑换码为一个批次, 共用名称、奖励、使用条件、可使用次数和到期时间, 见 Batch
//   - 兑换码: 随机生成或由管理员指定; 保存和兑换时都先规范化, 不区分大小写, 忽略空白和连字符, 见 Normalize
//   - 次数: 每个兑换码可以被 MaxUses 个用户各兑换一次, 1 为单次使用, 0 表示不限; 同一用户在一个批次中最多兑换 UserLimit 个
//   - 兑换: 检查到期时间、次数和使用条件后写兑换记录, 奖励以邮件发放, 玩家在信箱中领取附件, 见 Redeem
package redeem

// SourceRedeem 邮件来源: 兑换码的奖励, 来源单号为兑换记录 ID
const SourceRedeem = "redeem"

// mailContent 奖励邮件的正文, 标题为批次的名称
const mailContent = "兑换成功, 请领取附件中的奖励。"

// 操作日志
const (
	TargetBatch = "redeem_batch" // target_id 为批次 ID

	ActionCreate = "admin.redeem_batch_create"
	ActionUpdate = "admin.redeem_batch_update"
	ActionDelete = "admin.redeem_batch_delete"
)

// 兑换码的限制
const (
	maxNameLength = 64
	minCodeLength = 4         // 规范化后的兑换码最少的字符数
	maxCodeLength = 32        // 规范化后的兑换码最多的字符数
	maxCodeUses   = 1_000_000 // 每个兑换码可使用次数的上限
	MaxBatchCodes = 10_000    // 一个批次最多的兑换码数, 也是每人可兑换个数的上限
)

// 随机生成的兑换码
const (
	randomLength = 12
	// codeAlphabet 32 个字符, 去掉了容易看错的 0、O、1、I
	codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)
