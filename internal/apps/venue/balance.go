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

package venue

import (
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"gorm.io/gorm"
)

// Balancer 对局验算, 由拍卖模块注册: 按 v 推导出的拍卖参数模拟几种阵容的对局, 开启领先折让时求出折让写入 v.Auction.LeadDiscount,
// 再求出入场费写入 v.Auction.EntryFee, 返回各阵容的结果, 闸门由 Validate 判定; 只能用 l.Seeded 的副本生成拍品, 相同的配置得到相同的结果
type Balancer func(tx *gorm.DB, v *Venue, l *Lineup, b *settings.BalanceSettings) (*Balance, error)

// balancer 注册的对局验算, 为空时不验算, 如只测试本模块时
var balancer Balancer

// RegisterBalancer 注册对局验算, 需要在启动前调用
func RegisterBalancer(fn Balancer) {
	balancer = fn
}

// Calibrate 按价值分布推导金额和比例, 系数和开关见 settings.BalanceSettings; 领先折让和入场费由对局验算求出, 这里先置 0
//
// 两种模式共用一套金额, 按排位模式的价值分布推导: 人机模式按仓库价值系数缩小仓库, 金额不跟着缩小, 是人机模式的防刷手段之一。
func (r *AuctionRule) Calibrate(stats ValueStats, b *settings.BalanceSettings) {
	r.StartPrice = max(ofRate(stats.Mean, b.StartPriceRate), 1)
	r.MaxCarry = max(ofRate(stats.Mean, b.CarryOfMean), ofRate(stats.P90, b.CarryOfP90), r.StartPrice)
	r.BotCarry = min(max(ofRate(stats.Mean, b.BotCarryRate), r.StartPrice), r.MaxCarry)
	r.EntryFee, r.SpeedFloor, r.LeadDiscount = 0, 0, 0
	if b.SpeedFloorEnabled {
		r.SpeedFloor = ofRate(stats.Mean, b.SpeedFloorRate)
	}
	r.PreviewRate = int32(b.PreviewRate)
	r.IntelPrice = 0
	if b.IntelRate > 0 {
		r.IntelPrice = max(ofRate(stats.Mean, b.IntelRate), 1)
	}
}

// Calibrate 推导地图的金额: 没有配置档次时同 AuctionRule.Calibrate; 配置了档次时起拍价、速胜下限和情报价格按格子大小推导,
// 带入资金上限按拍场期望总价值 × 覆盖率, 起拍价取各档次中最低的
//
// 拍品的起拍价等金额出价时公开, 档次不公开, 所以不能按档次的期望价值算: 格子大小相同的档次按权重混合期望价值, 用同一套金额。
func (v *Venue) Calibrate(b *settings.BalanceSettings) {
	r := &v.Auction
	r.Calibrate(v.Simulation.Ranked, b)
	if !v.Lots.Enabled() {
		return
	}
	tiers := v.Lots.Tiers
	for i := range tiers {
		var weight, total float64
		for j := range tiers {
			if tiers[j].Warehouse.Grid == tiers[i].Warehouse.Grid {
				weight += float64(tiers[j].Weight)
				total += float64(tiers[j].Weight) * float64(v.Simulation.Tiers[j].Ranked.Mean)
			}
		}
		var t AuctionRule
		t.Calibrate(ValueStats{Mean: int64(total / weight)}, b)
		tiers[i].Auction = TierAuction{Mean: v.Simulation.Tiers[i].Ranked.Mean, StartPrice: t.StartPrice, SpeedFloor: t.SpeedFloor, IntelPrice: t.IntelPrice}
	}
	r.StartPrice = v.Lots.Tiers[0].Auction.StartPrice
	for _, t := range v.Lots.Tiers {
		r.StartPrice = min(r.StartPrice, t.Auction.StartPrice)
	}
	coverage := int(v.Lots.Coverage)
	if coverage == 0 {
		coverage = b.CarryCoverage
	}
	total := v.SessionMean()
	r.MaxCarry = max(ofRate(total, coverage), r.StartPrice)
	r.BotCarry = max(ofRate(r.MaxCarry, b.BotCarryShare), r.StartPrice)
}

// ofRate v × 比例, 比例为基点
func ofRate(v int64, rate int) int64 {
	return v * int64(rate) / BasisPoints
}

// checkNames 闸门的名称, 用于未通过时的提示
var checkNames = map[string]string{
	CheckRounds:       "轮数",
	CheckGridWidth:    "仓库宽度",
	CheckGridHeight:   "仓库高度",
	CheckTopShare:     "最贵一件的占比",
	CheckPriceValue:   "成交价与价值之比",
	CheckBidCV:        "出价离散度",
	CheckSpeedWin:     "速胜率",
	CheckAppraiseBias: "人机估值偏差",
	CheckSeatEV:       "真人盈亏",
	CheckWinRate:      "真人胜率",
}

// squadNames 阵容的名称, 用于未通过时的提示
var squadNames = map[string]string{
	SquadHumans: "4 个真人",
	SquadRanked: "排位补人机",
	SquadPve:    "人机模式",
}

// bound 闸门的上下限
func bound(n int64) *int64 {
	return &n
}

// newCheck 生成一项闸门并判定, min 和 max 为空表示不限
func newCheck(code, squad string, value int64, min, max *int64, unit string) Check {
	return Check{
		Code:   code,
		Squad:  squad,
		Value:  value,
		Min:    min,
		Max:    max,
		Unit:   unit,
		Passed: (min == nil || value >= *min) && (max == nil || value <= *max),
	}
}

// judge 按地图配置和各阵容的结果判定闸门, 安全区间见 settings.BalanceSettings
//
// 结构类的闸门只看 4 个真人的参考阵容: 含人机时成交价和速胜由人机的策略决定, 不能说明规则本身;
// 有人机的阵容只看人机估值、真人盈亏和真人胜率。人机模式只能练习时不看盈亏。
//
// 多拍品时带入资金有限, 第二名常常追不起, 成交价天然低于价值, 由入场费收回, 不看成交价与价值之比, 系统收支由真人盈亏把关;
// 真人花完钱后剩下的拍品只能给人机, 按拍品的胜率说明不了手感, 真人胜率改为拍场第 1 名(盈亏最高且赚钱)的比例。
func (b *Balance) judge(v *Venue, rule *settings.BalanceSettings) {
	n := func(x int) *int64 { return bound(int64(x)) }
	b.Checks = []Check{newCheck(CheckRounds, "", int64(v.Auction.Rounds), n(rule.MinRounds), n(rule.MaxRounds), UnitCount)}
	// 多个档次时按最小和最大的边长判定
	tiers := v.LotTiers()
	low, high := tiers[0].Warehouse.Grid, tiers[0].Warehouse.Grid
	for _, t := range tiers[1:] {
		g := t.Warehouse.Grid
		low.Width, low.Height = min(low.Width, g.Width), min(low.Height, g.Height)
		high.Width, high.Height = max(high.Width, g.Width), max(high.Height, g.Height)
	}
	for _, side := range []struct {
		code      string
		low, high int16
	}{{CheckGridWidth, low.Width, high.Width}, {CheckGridHeight, low.Height, high.Height}} {
		c := newCheck(side.code, "", int64(side.low), n(rule.MinGridSide), n(rule.MaxGridSide), UnitCount)
		if c.Passed && side.high != side.low {
			c = newCheck(side.code, "", int64(side.high), n(rule.MinGridSide), n(rule.MaxGridSide), UnitCount)
		}
		b.Checks = append(b.Checks, c)
	}
	minSeat := bound(ofRate(v.SessionMean(), rule.MinSeatEV))
	lots := v.Lots.Enabled()
	winRates := map[string]int{SquadRanked: rule.MinRankedWinRate, SquadPve: rule.MinPveWinRate}
	for _, s := range b.Squads {
		if s.Squad == SquadHumans {
			target := int(b.Target)
			b.Checks = append(b.Checks, newCheck(CheckTopShare, s.Squad, int64(s.TopShare), nil, n(rule.MaxTopShare), UnitBP))
			if !lots {
				b.Checks = append(b.Checks, newCheck(CheckPriceValue, s.Squad, int64(s.PriceValue), n(target-rule.PriceValueTolerance), n(target+rule.PriceValueTolerance), UnitBP))
			}
			b.Checks = append(b.Checks,
				newCheck(CheckBidCV, s.Squad, int64(s.BidCV), n(rule.MinBidCV), nil, UnitBP),
				newCheck(CheckSpeedWin, s.Squad, int64(s.SpeedWin), n(rule.MinSpeedWin), n(rule.MaxSpeedWin), UnitBP),
			)
		} else {
			b.Checks = append(b.Checks, newCheck(CheckAppraiseBias, s.Squad, int64(s.AppraiseBias), n(rule.MinAppraiseBias), n(rule.MaxAppraiseBias), UnitBP))
			if minWin := winRates[s.Squad]; minWin > 0 {
				win := s.WinRate
				if lots {
					win = s.RankWin
				}
				b.Checks = append(b.Checks, newCheck(CheckWinRate, s.Squad, int64(win), n(minWin), nil, UnitBP))
			}
		}
		if s.Squad != SquadPve || !v.Pve.PracticeOnly {
			b.Checks = append(b.Checks, newCheck(CheckSeatEV, s.Squad, s.SeatEV, minSeat, nil, UnitAmount))
		}
	}
	b.Passed = true
	for _, c := range b.Checks {
		b.Passed = b.Passed && c.Passed
	}
}

// failures 未通过的闸门, 如 "轮数、速胜率(4 个真人)"
func (b *Balance) failures() string {
	var names []string
	for _, c := range b.Checks {
		if c.Passed {
			continue
		}
		name := checkNames[c.Code]
		if c.Squad != "" {
			name += "(" + squadNames[c.Squad] + ")"
		}
		names = append(names, name)
	}
	return strings.Join(names, "、")
}
