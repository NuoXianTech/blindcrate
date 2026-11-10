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
//   - 准入: 检查地图和入场条件, 扣入场费和入场代价, 冻结 min(可用余额, 带入资金上限) 作为带入资金; 人机模式当天净收益达到上限时改为练习局
//   - 开局: 快照地图配置, 生成仓库, 每件藏品按首轮扫描比例的概率扫出轮廓(至少一件); 扫描顺序按位置从上到下、从左到右,
//     扫描时长按品质快照; 人机按地图配置的权重选出风格
//   - 出价(bidding): 只公开仓库大小和第 1 轮扫出的轮廓(占的格子), 件数、品质、名称和回收价直到开箱都不公开, 只能靠道具私下看到;
//     所有人暗标出价, 出价只增不减, 0 表示不加价; 真人都提交后提前结束
//   - 判定: 前几轮第一名不低于第二名的速胜倍率且不低于速胜下限时成交; 最后一轮最高价成交, 并列时只让并列的人加轮, 加轮用完后先出价的人成交;
//     一直没人出价时流拍。成交价为第二名的出价加上领先部分扣掉领先折让, 见 settlePrice
//   - 开箱(reveal): 进入时结算成交价、藏品、带入资金、名次和统计, 公开全部藏品和出价, 并释放对局, 玩家可以离开去排下一局;
//     客户端按扫描顺序逐件播放扫描, 仓库价值和盈亏随扫描揭晓, 全部扫描完再停留地图的开箱后停留时长, 到期后对局结束
//
// 入场费和入场代价成交时销毁, 流拍和取消时退还。金额和比例由保存地图时的验算推导(见 balance.go 和 venue.RegisterBalancer),
// 入场费按玩法平衡设置的真人盈亏目标求出, 用来回收货币。
//
// 人机(见 bot.go): 每轮结束时按风格的策略决策, 只用同座位的真人看得到的信息估值和出价, 不知道仓库价值和其他人的出价;
// 估值用开局时按公示概率算出的藏品分布, 出价时间在本轮内随机。
//
// 练习局不扣入场费和入场代价、不冻结资金、不发藏品, 也不写统计, 结算页的盈亏按正式局计算供参考。
// 玩家通过 ws 消息 auction:bid 出价, 断线重连后用 GET /auctions/{match_id} 恢复。
//
// 道具: 出价阶段可以通过 ws 消息 auction:use 使用效果属于对局场景的物品(见 effect 模块和 effects.go), 每用一个扣一个, 看到的藏品同时扫出轮廓;
// 次数和可用的道具由地图的道具规则决定, 练习局不能用; 得到的情报只给使用者看, 开箱后公开每个座位用了哪些道具。
// 对局被取消时退还用掉的道具, 流拍不退。
//
// 主持人(见 host.go, 规则见 venue.HostRule): 开局时选出性格(开箱后才公开)并公开初始热度, 每个规定的轮次开始时公开一条关于仓库的提示,
// 有一定概率说错, 开箱后公开真假; 热度在每轮结束时按加价的人数和幅度变化, 变动只追加不修改。人机不看提示。
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
	OutcomeUnsold    = "unsold"    // 流拍, 退还入场费和入场代价
	OutcomeCancelled = "cancelled" // 管理员取消, 退还入场费和入场代价
)

// 主持人提示的类型
const (
	HostHintCount = "count" // 品质不低于 Quality 的藏品共有 Count 件
	HostHintBest  = "best"  // 仓库里最好的藏品是 Quality
	HostHintPiece = "piece" // 位置为 (X, Y) 的藏品是 Quality, 只指第 1 轮扫出轮廓的
	HostHintValue = "value" // 仓库价值高于(Above)或不高于期望价值
)

// ws 消息类型
const (
	MessageBid     = "auction:bid"     // 客户端出价, data 为 BidRequest, 回复的 data 为 AuctionView
	MessageUse     = "auction:use"     // 客户端使用道具, data 为 UseRequest, 回复的 data 为 AuctionView
	MessageUpdated = "auction:updated" // 开局、每轮结束和开箱时推送给对局中的真人, data 为 AuctionView
)

const (
	// playerCount 每局的玩家数
	playerCount = 4
	// rankedMinHumans 排位模式开局至少需要的真人数
	rankedMinHumans = 2
	// rankedBotFillAfter 排位模式真人不够时, 最早加入的人等满这个时间后用人机补齐
	rankedBotFillAfter = 15 * time.Second
	// hintFarRatio 人机看公开出价时, 领先或落后达到这个倍数算"较多", 基点
	hintFarRatio = 12000
)

// 主持人和热度
const (
	// maxLieRate 提示说错的概率上限, 基点
	maxLieRate = 2500
	// hypeStart 开局时的热度
	hypeStart = 50
	// hypeMax 热度上限, 下限为 0
	hypeMax = 100
	// hypeHot 热度不低于它时煽动的主持人更敢说错
	hypeHot = 70
	// hypeCold 热度不高于它时狡黠的主持人更敢说错
	hypeCold = 30
	// hypePacePoints 按正常节奏(期望价值 ÷ 轮数)加价推高的热度, 按加价的比例缩放, 不低于 1, 不超过 hypeRaiseMax
	hypePacePoints = 5
	hypeRaiseMax   = 12
	// hypeCooling 有人加价的轮次热度先降这么多, 约等于 4 个人都按正常节奏加价推高的; 没人加价时降 hypeIdle
	hypeCooling = 15
	hypeIdle    = 15
)
