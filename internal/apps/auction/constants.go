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

// Package auction 仓库暗拍: 每张地图的人机和排位两种玩法, 玩法代码为 pve:<地图代码> 和 ranked:<地图代码>
//
// 每局 4 人, 人机模式 1 个真人和 3 个人机, 排位模式至少 2 个真人, 等满 15 秒后用人机补齐:
//   - 准入: 检查地图和入场条件, 扣入场代价, 冻结 min(可用余额, 带入资金上限) 作为带入资金; 人机模式当天净收益达到上限时改为练习局
//   - 开局: 快照地图配置, 生成仓库, 藏品的公开顺序随机, 扫描顺序按位置从上到下、从左到右, 扫描时长按品质快照; 人机按等级得到带误差的估值
//   - 出价(bidding): 全部藏品占的格子一开始就公开, 不公开名称和回收价; 所有人暗标出价, 出价只增不减, 0 表示不加价; 真人都提交后提前结束, 第 N 轮公开前 N 件藏品的品质
//   - 判定: 前几轮第一名不低于第二名的速胜倍率时成交; 最后一轮最高价成交, 并列时只让并列的人加轮, 加轮用完后先出价的人成交; 一直没人出价时流拍
//   - 开箱(reveal): 进入时结算成交价、藏品、带入资金、名次和统计, 公开全部藏品和出价, 并释放对局, 玩家可以离开去排下一局;
//     客户端按扫描顺序逐件播放扫描, 全部扫描完再停留地图的开箱后停留时长, 到期后对局结束
//
// 练习局不扣入场代价、不冻结资金、不发藏品, 也不写统计, 结算页的盈亏按正式局计算供参考。
// 玩家通过 ws 消息 auction:bid 出价, 断线重连后用 GET /auctions/{match_id} 恢复。
package auction

import "time"

// 阶段
const (
	PhaseBidding = "bidding" // 出价, Round 为第几轮, 加轮时继续递增
	PhaseReveal  = "reveal"  // 开箱, 已结算
)

// MetricBidWon 统计指标: 成交次数, 成交者结算时累加
const MetricBidWon = "bid_won"

// 结算结果
const (
	OutcomeSold      = "sold"      // 成交
	OutcomeUnsold    = "unsold"    // 流拍, 退还入场代价
	OutcomeCancelled = "cancelled" // 管理员取消, 退还入场代价
)

// 每轮反馈的提示, 只比较自己和其他人的最高出价
const (
	HintNoBid     = "no_bid"     // 自己还没出过价
	HintFarAhead  = "far_ahead"  // 领先较多
	HintAhead     = "ahead"      // 小幅领先
	HintTied      = "tied"       // 与其他人并列最高
	HintClose     = "close"      // 小幅落后
	HintFarBehind = "far_behind" // 落后较多
)

// ws 消息类型
const (
	MessageBid     = "auction:bid"     // 客户端出价, data 为 BidRequest, 回复的 data 为 AuctionView
	MessageUpdated = "auction:updated" // 开局、每轮结束和开箱时推送给对局中的真人, data 为 AuctionView
)

const (
	// playerCount 每局的玩家数
	playerCount = 4
	// rankedMinHumans 排位模式开局至少需要的真人数
	rankedMinHumans = 2
	// rankedBotFillAfter 排位模式真人不够时, 最早加入的人等满这个时间后用人机补齐
	rankedBotFillAfter = 15 * time.Second
	// hintFarRatio 领先或落后达到这个倍数时提示"较多", 基点
	hintFarRatio = 12000
)
