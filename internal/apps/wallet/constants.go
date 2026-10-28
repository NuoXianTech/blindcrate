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

// 货币代码
const (
	CurrencyChip = "chip" // 筹码, 默认的基础货币, 见 settings.EconomySettings
	CurrencyGold = "gold" // 金条, 珍贵货币
)

// RefCurrency 设置中引用货币的字段, 可选值为全部启用的货币, 见 settings.RegisterRef
const RefCurrency = "currency"

// 业务类型, 每种会变动余额的业务一个
const (
	BizAdminAdjust = "admin_adjust"  // 管理员调账
	BizItemSell    = "item_sell"     // 出售物品入账, biz_id 为出售单 ID
	BizItemSellFee = "item_sell_fee" // 出售物品的手续费, 入账后扣除, biz_id 为出售单 ID
	BizSignupGrant = "signup_grant"  // 新用户注册时发放的新手资金, biz_id 为用户 ID

	BizMatchEntryFee     = "match_entry_fee"     // 对局的入场代价, biz_id 为对局 ID
	BizMatchCarryFreeze  = "match_carry_freeze"  // 开局时冻结带入资金, biz_id 为对局 ID
	BizMatchCarryRelease = "match_carry_release" // 结算或取消时解冻剩余的带入资金, biz_id 为对局 ID
	BizMatchBidSettle    = "match_bid_settle"    // 成交时从冻结的带入资金中扣除成交价, biz_id 为对局 ID
	BizMatchRefund       = "match_refund"        // 流拍或取消时退还入场代价, biz_id 为对局 ID
	BizMatchReward       = "match_reward"        // 成交者获得的藏品, 已下架的藏品按回收价折成货币, biz_id 为对局 ID
)

// 推送给客户端的消息类型
const (
	MessageBalanceChanged = "wallet:balance_changed" // 余额变动, data 为 BalanceChangedMessage
)

// MaxAmount 单笔变动金额上限, 防止异常请求一次写入过大金额
const MaxAmount int64 = 1_000_000_000_000
