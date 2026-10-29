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

// Package leaderboard 排行榜: 管理员配置的榜单, 按 stats 中的某个指标或赛季的段位分排名
//
// 榜单只是一行配置(统计范围、周期、指标), 新增榜单不需要改代码:
//   - 周期: 累计、当天或赛季, 查询时换成 stats 的周期, 如 day:2026-10-28、season:s1
//   - 排名: 按数值从大到小, 同分时先达到的在前; 封禁和停用的账号不上榜, 人机不写统计
//   - 缓存: 前 N 名缓存 CacheTTL, 所有人看到同一份; 自己的名次每次单独查询
//
// 榜单上只有名次、昵称、头像和数值, 不能查看其他玩家的资料。
package leaderboard

import "time"

// MetricRating 段位分指标, 读赛季的段位分而不是 stats, 只能用于全部对局范围的赛季榜单
const MetricRating = "rating"

// ratingName 段位分指标的展示名称
const ratingName = "段位分"

// 榜单的统计周期
const (
	PeriodTotal  = "total"  // 累计
	PeriodDay    = "day"    // 当天, 按 schedule.time_zone 计算
	PeriodSeason = "season" // 当前赛季, 赛季之间的间隔中为刚结束的赛季, 还没有赛季时榜单为空
)

const (
	// CacheTTL 前 N 名的缓存时长
	CacheTTL = 60 * time.Second
	// MinSize 榜单展示的最少人数
	MinSize = 10
	// MaxSize 榜单展示的最多人数
	MaxSize = 500
	// DefaultSize 榜单默认展示的人数
	DefaultSize = 100
)

// cacheKeyPrefix 前 N 名的缓存, 后接榜单代码、修改时间和统计周期, 修改榜单后自然失效
const cacheKeyPrefix = "blindcrate:leaderboard:"
