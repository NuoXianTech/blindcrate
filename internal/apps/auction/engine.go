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
	"math/rand/v2"
	"slices"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
)

// 本文件是仓库的推进规则, 不读写数据库: 对局(service.go)把结果写库, 保存地图时的验算(balance.go)只统计结果,
// 两边按同一份规则推进。判定和成交价见 rules.go, 热度的变动见 hypeDelta

// seat 仓库中的一个座位
type seat struct {
	Seat     int16
	Human    bool
	Carry    int64                              // 剩余资金, 当前拍品出价的上限, 成交后扣除成交价
	Bid      int64                              // 当前拍品的出价, 0 表示还没出过价
	BidAt    time.Time                          // 当前出价的提交时间
	Estimate int64                              // 人机对当前拍品的最新估值, 真人为 0
	Seed     int64                              // 人机在这一局的随机种子
	Decide   bot.Decider[Observation, Decision] // 人机的决策, 真人为 nil

	shown  int64 // 已公开的出价, 拍品的前 N-1 轮每轮结束后公开
	before int64 // 本轮开始时的出价
}

// lotSpec 一个拍品开拍时就确定的信息
type lotSpec struct {
	Rule       venue.AuctionRule // 这个拍品的拍卖参数
	Grid       venue.Grid
	Odds       []venue.PieceOdds // 按格子大小推断的公示分布
	Footprints []Footprint       // 第 1 轮扫出轮廓的藏品占的格子
	Future     int64             // 之后的拍品按格子大小推断的期望价值之和
	LotsLeft   int32             // 之后还有几个拍品
}

// warehouse 一个仓库的局面: 座位、热度和当前拍品的进度
//
// 开拍一个拍品见 openLot, 从数据库恢复拍到一半的拍品见 resumeLot; 真人的出价由调用方写入(见 raise), 之后用 closeRound 结束这一轮。
type warehouse struct {
	Seats     []seat           // 按座位号排序
	Hosted    bool             // 是否记录热度, 没有主持人的对局不记录
	Hype      int32            // 当前热度, 跨拍品延续
	HypeDecay int64            // 每个拍品开场时热度向开局热度回落的比例, 基点
	Round     int32            // 当前轮次, 在整个仓库内连续编号, 还没开拍时为 0
	Clock     func() time.Time // 人机加价时的提交时间, 只用于并列时比较先后

	lot        lotSpec
	firstRound int32   // 当前拍品开拍时的轮次
	extra      int32   // 当前拍品已加的轮数
	contenders []int16 // 加轮时可以出价的座位, 为空表示所有人
}

// botFailure 一个人机这一轮决策出错, 视为不加价
type botFailure struct {
	Seat int16
	Err  error
}

// botBid 一个人机这一轮的出价, 0 表示不加价
type botBid struct {
	Seat   int16
	Amount int64
}

// roundResult 结束一轮的结果
type roundResult struct {
	Round   int32        // 结束的轮次
	Bots    []botBid     // 这一轮可以出价的人机的出价, 按座位
	Failed  []botFailure // 决策出错的人机
	Verdict verdict      // Outcome 不为空时当前拍品结束, 之后的字段都为零值
	Raises  int32        // 这一轮加价的座位数, 只有记录热度时有
	Delta   int32        // 热度的变动, 同上
	Hint    bool         // 下一轮还在规定的轮数内, 主持人给出提示; 加轮不给
}

// publicBefore 仓库进行到第 round 轮时, 从第 firstRound 轮开拍的拍品上这个轮次之前的出价已公开:
// 拍品的前 N-1 轮(N 为规则的轮数)每轮结束后公开, 最后一轮和加轮在结算后公开
func publicBefore(round, firstRound, rounds int32) int32 {
	return min(round, firstRound+rounds-1)
}

// coolHype 拍品开场时热度按 decay(基点)向开局热度回落
func coolHype(hype int32, decay int64) int32 {
	return hypeStart + int32(int64(hype-hypeStart)*(venue.BasisPoints-decay)/venue.BasisPoints)
}

// openLot 开拍下一个拍品: 从上一个拍品的最后一轮之后开始, 清空每个人的出价和估值; 不是第一个拍品时热度回落
func (s *warehouse) openLot(spec lotSpec) {
	if s.Hosted && s.Round > 0 {
		s.Hype = coolHype(s.Hype, s.HypeDecay)
	}
	for i := range s.Seats {
		p := &s.Seats[i]
		p.Bid, p.BidAt, p.Estimate, p.shown, p.before = 0, time.Time{}, 0, 0, 0
	}
	s.Round++
	s.lot, s.firstRound, s.extra, s.contenders = spec, s.Round, 0, nil
}

// resumeLot 恢复拍到一半的拍品: Round 和各座位的出价由调用方按数据库填好, bids 为这个拍品的全部出价记录
func (s *warehouse) resumeLot(spec lotSpec, firstRound, extra int32, contenders []int16, bids []Bid) {
	s.lot, s.firstRound, s.extra, s.contenders = spec, firstRound, extra, contenders
	public := publicBefore(s.Round, firstRound, spec.Rule.Rounds)
	for i := range s.Seats {
		p := &s.Seats[i]
		p.shown, p.before = 0, 0
		for _, b := range bids {
			if b.Seat != p.Seat {
				continue
			}
			if b.Round < s.Round {
				p.before = max(p.before, b.Amount)
			}
			if b.Round < public {
				p.shown = max(p.shown, b.Amount)
			}
		}
	}
}

// lotRound 当前轮次是当前拍品的第几轮
func (s *warehouse) lotRound() int32 {
	return s.Round - s.firstRound + 1
}

// contender 座位在本轮是否有资格出价, 加轮时只有并列最高的人有
func (s *warehouse) contender(seat int16) bool {
	return len(s.contenders) == 0 || slices.Contains(s.contenders, seat)
}

// canBid p 在本轮是否可以出价: 有资格, 且剩余资金够当前拍品的起拍价
func (s *warehouse) canBid(p *seat) bool {
	return s.contender(p.Seat) && p.Carry >= s.lot.Rule.StartPrice
}

// humanCanBid 本轮是否有真人可以出价, 没有时不用等待, 直接结束这一轮
func (s *warehouse) humanCanBid() bool {
	for i := range s.Seats {
		if p := &s.Seats[i]; p.Human && s.canBid(p) {
			return true
		}
	}
	return false
}

// others 第 i 个座位看到的其他座位已公开的出价, 按座位
func (s *warehouse) others(i int) []int64 {
	others := make([]int64, 0, len(s.Seats)-1)
	for j := range s.Seats {
		if j != i {
			others = append(others, s.Seats[j].shown)
		}
	}
	return others
}

// bidders 判定用的出价者
func (s *warehouse) bidders() []bidder {
	bidders := make([]bidder, len(s.Seats))
	for i, p := range s.Seats {
		bidders[i] = bidder{Seat: p.Seat, Bid: p.Bid, BidAt: p.BidAt}
	}
	return bidders
}

// raise 第 i 个座位加价到 amount, 不高于当前出价时不变
func (s *warehouse) raise(i int, amount int64, at time.Time) {
	if p := &s.Seats[i]; amount > p.Bid {
		p.Bid, p.BidAt = amount, at
	}
}

// closeRound 结束当前轮: 有资格出价的人机按风格决策并出价, 再判定; 成交时从成交者的剩余资金中扣除成交价,
// 没有结束时记下加轮和下一轮可以出价的座位, 更新热度, 公开这一轮的出价, 进入下一轮
//
// 人机看到的与真人在这一轮出价时相同, 见 Observation; 每轮的随机数由种子和仓库内的轮次决定, 各拍品互不相同。
// 剩余资金不够起拍价的人机只更新估值, 不出价; 出价超过剩余资金时按剩余资金, 低于起拍价或不高于当前出价时为不加价。
func (s *warehouse) closeRound() roundResult {
	rule := &s.lot.Rule
	round := s.lotRound()
	res := roundResult{Round: s.Round}
	for i := range s.Seats {
		p := &s.Seats[i]
		if p.Human || !s.contender(p.Seat) {
			continue
		}
		obs := Observation{
			Round:      round,
			Rules:      *rule,
			Grid:       s.lot.Grid,
			Footprints: s.lot.Footprints,
			Odds:       s.lot.Odds,
			Carry:      p.Carry,
			Bid:        p.Bid,
			Others:     s.others(i),
			Future:     s.lot.Future,
			LotsLeft:   s.lot.LotsLeft,
			Seed:       p.Seed,
		}
		var amount int64
		decision, err := p.Decide(obs, rand.New(rand.NewPCG(uint64(p.Seed), uint64(s.Round))))
		if err != nil {
			res.Failed = append(res.Failed, botFailure{Seat: p.Seat, Err: err})
		} else {
			p.Estimate = decision.Estimate
			if amount = min(decision.Bid, p.Carry); amount < rule.StartPrice || amount <= p.Bid {
				amount = 0
			}
		}
		if !s.canBid(p) {
			continue
		}
		res.Bots = append(res.Bots, botBid{Seat: p.Seat, Amount: amount})
		if amount > 0 {
			s.raise(i, amount, s.Clock())
		}
	}

	res.Verdict = judge(rule, round, s.extra, s.bidders())
	if res.Verdict.Outcome != "" {
		if res.Verdict.Outcome == OutcomeSold {
			for i := range s.Seats {
				if p := &s.Seats[i]; p.Seat == res.Verdict.Winner {
					p.Carry -= res.Verdict.Price
				}
			}
		}
		return res
	}

	if round >= rule.Rounds {
		s.extra++
	}
	s.contenders = res.Verdict.Contenders
	if s.Hosted {
		var raises []int64
		for _, p := range s.Seats {
			if p.Bid > p.before {
				raises = append(raises, p.Bid-p.before)
			}
		}
		next := min(max(s.Hype+hypeDelta(raises, expectedValue(s.lot.Odds), rule.Rounds), 0), hypeMax)
		res.Raises, res.Delta = int32(len(raises)), next-s.Hype
		s.Hype = next
	}
	res.Hint = round < rule.Rounds
	for i := range s.Seats {
		p := &s.Seats[i]
		if round < rule.Rounds {
			p.shown = p.Bid
		}
		p.before = p.Bid
	}
	s.Round++
	return res
}
