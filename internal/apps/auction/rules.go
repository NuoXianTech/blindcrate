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

package auction

import (
	"cmp"
	"slices"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
)

// 本文件是不读写数据库的规则函数, 随机数由调用方传入

// bidder 判定时的一个出价者
type bidder struct {
	Seat  int16
	Bid   int64     // 当前出价, 0 表示没出过价
	BidAt time.Time // 当前出价的提交时间
}

// verdict 一轮的判定结果
type verdict struct {
	Outcome    string  // 为空表示继续下一轮, 否则为 OutcomeSold 或 OutcomeUnsold
	Winner     int16   // 成交者的座位
	Price      int64   // 成交价
	Contenders []int16 // 继续时下一轮可以出价的座位, 为空表示所有人
}

// sortBidders 按出价从高到低, 出价相同时先出价的在前, 再按座位号
func sortBidders(bidders []bidder) []bidder {
	sorted := slices.Clone(bidders)
	slices.SortFunc(sorted, func(a, b bidder) int {
		if c := cmp.Compare(b.Bid, a.Bid); c != 0 {
			return c
		}
		if c := a.BidAt.Compare(b.BidAt); c != 0 {
			return c
		}
		return cmp.Compare(a.Seat, b.Seat)
	})
	return sorted
}

// judge 判定第 round 轮, extraRounds 为之前已加的轮数
//
//   - 最后一轮之前: 至少两人出过价, 且第一名不低于第二名 × 本轮速胜倍率、不低于速胜下限时, 第一名成交
//   - 最后一轮及加轮: 没人出过价时流拍; 最高价唯一时成交; 并列时只让并列的人加轮, 加轮用完后先出价的人成交
//
// 成交价见 settlePrice。
func judge(rule *venue.AuctionRule, round, extraRounds int32, bidders []bidder) verdict {
	sorted := sortBidders(bidders)
	var top, second bidder
	if len(sorted) > 0 {
		top = sorted[0]
	}
	if len(sorted) > 1 {
		second = sorted[1]
	}

	if round < rule.Rounds {
		quickWin := int64(rule.QuickWin[round-1])
		if second.Bid > 0 && top.Bid > second.Bid && top.Bid*venue.BasisPoints >= second.Bid*quickWin && top.Bid >= rule.SpeedFloor {
			return verdict{Outcome: OutcomeSold, Winner: top.Seat, Price: settlePrice(rule, top.Bid, second.Bid)}
		}
		return verdict{}
	}

	if top.Bid == 0 {
		return verdict{Outcome: OutcomeUnsold}
	}
	var tied []int16
	for _, b := range sorted {
		if b.Bid == top.Bid {
			tied = append(tied, b.Seat)
		}
	}
	if len(tied) > 1 && extraRounds < rule.MaxExtraRounds {
		slices.Sort(tied)
		return verdict{Contenders: tied}
	}
	return verdict{Outcome: OutcomeSold, Winner: top.Seat, Price: settlePrice(rule, top.Bid, second.Bid)}
}

// settlePrice 成交价: 第二名的出价加上第一名领先部分 × (1 - 领先折让); 没有第二名时按起拍价算, 并列时就是最高价
//
// 只按最高价成交时赢家总是买贵, 按第二名的出价成交时赢家白捡; 折让由保存地图时的验算求出, 让成交价平均落在仓库价值 × (1 - 出售手续费率)。
func settlePrice(rule *venue.AuctionRule, top, second int64) int64 {
	base := max(second, rule.StartPrice)
	if top <= base {
		return top
	}
	return base + (top-base)*int64(venue.BasisPoints-rule.LeadDiscount)/venue.BasisPoints
}

// previewPieces 第 1 轮扫出轮廓的藏品, 返回每件是否扫出: 每件按 rate(基点)的概率独立抽取, rate 大于 0 时至少扫出一件
//
// 按概率而不是按比例取整, 否则从扫出的件数就能反推出总件数。randN 返回 [0, n) 的随机数。
func previewPieces(n int, rate int32, randN func(n int64) int64) []bool {
	picked := make([]bool, n)
	if n == 0 || rate <= 0 {
		return picked
	}
	some := false
	for i := range picked {
		if randN(venue.BasisPoints) < int64(rate) {
			picked[i], some = true, true
		}
	}
	if !some {
		picked[randN(int64(n))] = true
	}
	return picked
}

// shareWeights 奖池按名次分配的权重, 名次更后的都按最后一个
var shareWeights = []int64{3, 2, 1}

// splitPool 成交后的奖池和分配, 不读写数据库
//
// fees 为正式局真人缴纳的入场费之和, 按进奖池比例进入奖池; overpay 为正式局真人成交时成交价超出仓库价值的部分, 其他情况为 0,
// 按比例进入奖池且不超过上限。奖池分给 eligible(正式局出过价的真人输家, 按名次排好), 权重见 shareWeights, 除不尽的零头给名次最前的人;
// 没有人可分时奖池销毁, 返回的 shares 为空。
func splitPool(rule *venue.AuctionRule, fees, overpay int64, eligible []int16) (pool, rebate int64, shares map[int16]int64) {
	if overpay > 0 {
		rebate = min(overpay*int64(rule.RebateRate)/venue.BasisPoints, rule.RebateCap)
	}
	pool = fees*int64(rule.PoolRate)/venue.BasisPoints + rebate
	if pool <= 0 || len(eligible) == 0 {
		return pool, rebate, nil
	}
	weights := make([]int64, len(eligible))
	var total int64
	for i := range eligible {
		weights[i] = shareWeights[min(i, len(shareWeights)-1)]
		total += weights[i]
	}
	shares = make(map[int16]int64, len(eligible))
	paid := int64(0)
	for i, seat := range eligible {
		shares[seat] = pool * weights[i] / total
		paid += shares[seat]
	}
	shares[eligible[0]] += pool - paid
	return pool, rebate, shares
}

// rankSeats 成交后的名次, 从 1 开始且不重复
//
// 盈利(仓库价值 > 成交价)的成交者第 1; 其他出过价的人按出价与仓库价值的差距从小到大, 差距相同时先出价的在前;
// 之后是没出过价的人; 亏本的成交者最后。
func rankSeats(bidders []bidder, winner int16, price, value int64) map[int16]int16 {
	var (
		winnerBidder bidder
		bid, noBid   []bidder
	)
	for _, b := range sortBidders(bidders) {
		switch {
		case b.Seat == winner:
			winnerBidder = b
		case b.Bid > 0:
			bid = append(bid, b)
		default:
			noBid = append(noBid, b)
		}
	}
	slices.SortStableFunc(bid, func(a, b bidder) int {
		return cmp.Compare(absInt64(a.Bid-value), absInt64(b.Bid-value))
	})

	order := make([]bidder, 0, len(bidders))
	if value > price {
		order = append(order, winnerBidder)
	}
	order = append(order, bid...)
	order = append(order, noBid...)
	if value <= price {
		order = append(order, winnerBidder)
	}
	ranks := make(map[int16]int16, len(order))
	for i, b := range order {
		ranks[b.Seat] = int16(i + 1)
	}
	return ranks
}

// feedbackFor 第 round 轮结束后 seat 看到的反馈, bids 为座位 -> 当前出价
func feedbackFor(round int32, seat int16, bids map[int16]int64) Feedback {
	own := bids[seat]
	feedback := Feedback{Round: round, Hint: HintNoBid}
	if own == 0 {
		return feedback
	}
	var best int64
	feedback.Rank = 1
	for other, amount := range bids {
		if other == seat {
			continue
		}
		best = max(best, amount)
		if amount > own {
			feedback.Rank++
		}
	}
	switch {
	case own > best && (best == 0 || own*venue.BasisPoints >= best*hintFarRatio):
		feedback.Hint = HintFarAhead
	case own > best:
		feedback.Hint = HintAhead
	case own == best:
		feedback.Hint = HintTied
	case best*venue.BasisPoints >= own*hintFarRatio:
		feedback.Hint = HintFarBehind
	default:
		feedback.Hint = HintClose
	}
	return feedback
}

// botBidAt 人机的出价时间, 只用于并列时比较先后: 在从 start 开始、时长为 length 的一轮内均匀随机, 时间还没到时取 now; length 为 0 时为 now
//
// 真人都提交后本轮提前结束, 不能只在 start 到 now 之间随机, 否则人机总排在最后提交的真人前面。
// 这样真人和人机的时间同分布, 真人越早出价越可能排在人机前面。
func botBidAt(start time.Time, length time.Duration, now time.Time, randN func(int64) int64) time.Time {
	if length <= 0 {
		return now
	}
	if at := start.Add(time.Duration(randN(int64(length) + 1))); at.Before(now) {
		return at
	}
	return now
}

// scanOrders 开箱时的扫描顺序, 按位置从上到下、从左到右, 返回每件藏品的顺序, 从 1 开始
func scanOrders(pieces []venue.Piece) []int32 {
	indexes := make([]int, len(pieces))
	for i := range indexes {
		indexes[i] = i
	}
	slices.SortFunc(indexes, func(a, b int) int {
		return cmp.Or(cmp.Compare(pieces[a].Y, pieces[b].Y), cmp.Compare(pieces[a].X, pieces[b].X))
	})
	orders := make([]int32, len(pieces))
	for order, i := range indexes {
		orders[i] = int32(order + 1)
	}
	return orders
}

// amountIn 一组变动中某种货币的金额
func amountIn(bundle reward.Bundle, currency string) int64 {
	var amount int64
	for _, line := range bundle.Currencies {
		if line.Currency == currency {
			amount += line.Amount
		}
	}
	return amount
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
