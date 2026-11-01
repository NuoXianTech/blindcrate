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
//   - 最后一轮之前: 至少两人出过价, 且第一名不低于第二名 × 本轮速胜倍率时, 第一名成交
//   - 最后一轮及加轮: 没人出过价时流拍; 最高价唯一时成交; 并列时只让并列的人加轮, 加轮用完后先出价的人成交
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
		if second.Bid > 0 && top.Bid > second.Bid && top.Bid*venue.BasisPoints >= second.Bid*quickWin {
			return verdict{Outcome: OutcomeSold, Winner: top.Seat, Price: top.Bid}
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
	return verdict{Outcome: OutcomeSold, Winner: top.Seat, Price: top.Bid}
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
