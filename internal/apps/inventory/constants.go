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

package inventory

// 业务类型, 每种会变动背包的业务一个
const (
	BizAdminAdjust = "admin_adjust" // 管理员发放或扣除
	BizItemSell    = "item_sell"    // 出售物品, biz_id 为出售单 ID

	BizMatchEntryFee = "match_entry_fee" // 对局的入场代价, biz_id 为对局 ID
	BizMatchRefund   = "match_refund"    // 流拍或取消时退还入场代价, biz_id 为对局 ID
	BizMatchReward   = "match_reward"    // 成交者获得的藏品, biz_id 为对局 ID

	BizSeasonReward = "season_reward" // 赛季结束后按最高段位分发放的奖励, biz_id 为赛季代码
)

// 推送给客户端的消息类型
const (
	MessageItemsChanged = "inventory:items_changed" // 背包物品数量变动, data 为 ItemsChangedMessage
)

// MaxQuantity 单笔变动数量上限, 防止异常请求一次写入过多物品
const MaxQuantity int64 = 1_000_000
