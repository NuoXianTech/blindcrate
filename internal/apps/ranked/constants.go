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

// Package ranked 段位分: 排位按名次加减的分数, 每个赛季分开计算
//
//   - 赛季: 进行中的赛季按赛季 ID 记分, 赛季之外的排位记在赛季 0, 见 seasonIDOf
//   - 初始分: 第一次在赛季中结算排位时创建, 有上一个赛季的段位分时按赛季的继承比例向基础分靠拢, 见 initialRating
//   - 加减: 成交的排位局按名次加减, 人机占名次但不计分, 流拍不计分; 扣到 0 为止, 见 Settle
//   - 条件: 注册条件类型 min_rating、max_rating 和 no_afk_ban, 地图可以按段位分限制入场, 也可以把挂机禁赛的人挡在其他地方
//   - 赛季奖励: 赛季结束后按赛季中的最高段位分以邮件发放, 玩家在信箱中领取, 见 HandleSeasonSettle
//   - 挂机: 一局中所有轮次都没有提交出价记为一次挂机, 写入统计指标 afk; 24 小时内次数达到设置时禁止排位, 见 RecordAfk、AfkBanReason
package ranked

import "time"

// 条件类型
const (
	TypeMinRating = "min_rating" // 当前赛季的段位分不少于
	TypeMaxRating = "max_rating" // 当前赛季的段位分不高于
	TypeNoAfkBan  = "no_afk_ban" // 现在没有因挂机被禁赛
)

// MetricAfk 统计指标: 排位挂机次数, 结算时累加
const MetricAfk = "afk"

// 发放赛季奖励
const (
	// SourceSeason 邮件来源: 赛季奖励, 来源单号为赛季代码
	SourceSeason = "season"
	// seasonMailTitle 赛季奖励邮件的标题
	seasonMailTitle = "赛季奖励"
	// seasonMailContent 赛季奖励邮件的正文, 参数为赛季名称、最高段位分和达到的分数线
	seasonMailContent = "%s 已结束。你在本赛季的最高段位分为 %d, 达到 %d 分档的赛季奖励, 请领取附件。"
	// settleDelay 赛季结束多久之后发放, 等结束前开始结算的对局写完段位分
	settleDelay = time.Minute
	// settleBatch 每批发放的人数
	settleBatch = 200
)

// 挂机禁赛
const (
	// afkWindow 统计挂机次数的时间窗口
	afkWindow = 24 * time.Hour
	// afkKeyPrefix 每个用户一个有序集合, 成员为挂机的对局 ID, 分数为记录时间的毫秒时间戳
	afkKeyPrefix = "blindcrate:ranked:afk:"
	// afkBanKeyPrefix 禁赛中的用户, 过期时间即禁赛结束的时间
	afkBanKeyPrefix = "blindcrate:ranked:afk_ban:"
)

// 操作日志的对象类型, 见 audit.Record
const (
	TargetRankedTier = "ranked_tier"
	TargetSuspect    = "suspect" // target_id 为较小的用户 ID, 另一个见 detail.user_b
)

// 操作日志的操作类型, 格式为 "模块.动作", 在 init 中登记
const (
	ActionRankedTierCreate = "admin.ranked_tier_create"
	ActionRankedTierUpdate = "admin.ranked_tier_update"
	ActionRankedTierDelete = "admin.ranked_tier_delete"
	ActionSuspectReview    = "admin.suspect_review"
)
