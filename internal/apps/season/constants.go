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

// Package season 赛季: 管理员配置的一段时间, 赛季排行榜等赛季数据按赛季分开统计
//
//   - 时间: 从开始时间(含)到结束时间(不含), 赛季之间不能重叠, 可以有间隔, 同一时刻最多一个进行中的赛季
//   - 状态: 由时间推算出未开始、进行中、已结束, 赛季奖励发完后为已结算, 见 Season.StatusAt
//   - 统计: 进行中的赛季在结算对局时额外写入 season:<code> 周期, 见 Period, 不需要清空数据
//   - 奖励: 按赛季中的最高段位分分档, 赛季结束后由定时任务发放, 见 Season.RewardOf 和 ranked.HandleSeasonSettle
package season

// 赛季的状态
const (
	StatusUpcoming = "upcoming" // 未开始
	StatusActive   = "active"   // 进行中
	StatusEnded    = "ended"    // 已结束, 赛季奖励还没发完
	StatusSettled  = "settled"  // 已结算, 赛季奖励已发完
)

// periodPrefix 赛季统计周期的前缀, 见 Period
const periodPrefix = "season:"

// 赛季奖励的限制
const (
	maxRewards      = 10      // 最多的档数
	maxRewardLines  = 8       // 每档最多的行数
	maxRewardRating = 1000000 // 分数线的上限, 与段位的分数线一致
)
