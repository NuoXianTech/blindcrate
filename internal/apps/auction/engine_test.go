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
	"errors"
	"math/rand/v2"
	"reflect"
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
)

// testSpec 期望价值 1000 的拍品, 起拍价 100
func testSpec(rounds int32, quickWin ...int32) lotSpec {
	return lotSpec{
		Rule: venue.AuctionRule{Rounds: rounds, QuickWin: quickWin, MaxExtraRounds: 1, StartPrice: 100},
		Odds: []venue.PieceOdds{{Quality: 1, Width: 1, Height: 1, Count: 1, Price: 1000}},
	}
}

// testWarehouse 4 个座位的仓库, 前 humans 个是真人, 每人带 5000; 提交时间按调用的先后递增
func testWarehouse(humans int, deciders ...bot.Decider[Observation, Decision]) *warehouse {
	s := &warehouse{Seats: make([]seat, playerCount), Hosted: true, Hype: hypeStart, HypeDecay: 5000}
	for i := range s.Seats {
		s.Seats[i] = seat{Seat: int16(i + 1), Human: i < humans, Carry: 5000}
		if i >= humans {
			s.Seats[i].Decide, s.Seats[i].Seed = deciders[i-humans], int64(i)
		}
	}
	tick := time.Unix(0, 0)
	s.Clock = func() time.Time {
		tick = tick.Add(time.Second)
		return tick
	}
	return s
}

// fixedBid 每轮都出同一个价的人机, 估值为出价的 2 倍
func fixedBid(amount int64) bot.Decider[Observation, Decision] {
	return func(Observation, *rand.Rand) (Decision, error) {
		return Decision{Bid: amount, Estimate: amount * 2}, nil
	}
}

// TestWarehouseTie 最后一轮并列时只让并列的人加轮, 加轮用完后先出价的人成交; 最后一轮和加轮的出价不公开, 加轮不给提示
func TestWarehouseTie(t *testing.T) {
	s := testWarehouse(4)
	s.openLot(testSpec(2, 20000))
	if s.Round != 1 || s.lotRound() != 1 || !s.humanCanBid() {
		t.Fatalf("after opening: round %d, lot round %d, human can bid %v", s.Round, s.lotRound(), s.humanCanBid())
	}

	s.raise(0, 300, s.Clock())
	s.raise(1, 300, s.Clock())
	res := s.closeRound()
	// 两人各加 300, 正常节奏为 1000 ÷ 2 = 500, 各推高 3, 再降温 15
	want := roundResult{Round: 1, Raises: 2, Delta: -9, Hint: true}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("round 1: got %+v, want %+v", res, want)
	}
	if s.Hype != hypeStart-9 || s.Round != 2 {
		t.Errorf("after round 1: hype %d, round %d", s.Hype, s.Round)
	}
	if got := s.others(2); !reflect.DeepEqual(got, []int64{300, 300, 0}) {
		t.Errorf("seat 3 sees %v after round 1, want the round 1 bids", got)
	}

	s.raise(1, 500, s.Clock())
	s.raise(0, 500, s.Clock())
	res = s.closeRound()
	if res.Verdict.Outcome != "" || !reflect.DeepEqual(res.Verdict.Contenders, []int16{1, 2}) || res.Hint {
		t.Fatalf("round 2: got %+v, want an extra round between seats 1 and 2 without a hint", res)
	}
	if s.extra != 1 || s.contender(3) || !s.contender(2) || s.lotRound() != 3 {
		t.Errorf("after round 2: extra %d, lot round %d, contenders %v", s.extra, s.lotRound(), s.contenders)
	}
	if got := s.others(2); !reflect.DeepEqual(got, []int64{300, 300, 0}) {
		t.Errorf("seat 3 sees %v after the last round, want the round 1 bids", got)
	}

	// 加轮没人再加价, 座位 2 先出到 500
	res = s.closeRound()
	if v := res.Verdict; v.Outcome != OutcomeSold || v.Winner != 2 || v.Price != 500 || res.Round != 3 {
		t.Fatalf("extra round: got %+v, want sold to seat 2 at 500 in round 3", res)
	}
	if s.Seats[1].Carry != 4500 || s.Seats[0].Carry != 5000 || s.Round != 3 {
		t.Errorf("after settling: carries %d %d, round %d", s.Seats[0].Carry, s.Seats[1].Carry, s.Round)
	}
}

// TestWarehouseBots 人机的出价不超过剩余资金, 低于起拍价或不高于当前出价时为不加价; 剩余资金不够起拍价的只更新估值;
// 决策出错的视为不加价
func TestWarehouseBots(t *testing.T) {
	failing := func(Observation, *rand.Rand) (Decision, error) { return Decision{}, errors.New("boom") }
	s := testWarehouse(1, fixedBid(9000), fixedBid(50), failing)
	s.Seats[1].Carry = 800
	s.openLot(testSpec(3, 50000, 50000))
	res := s.closeRound()
	if want := []botBid{{Seat: 2, Amount: 800}, {Seat: 3}, {Seat: 4}}; !reflect.DeepEqual(res.Bots, want) {
		t.Errorf("round 1 bots: got %+v, want %+v", res.Bots, want)
	}
	if len(res.Failed) != 1 || res.Failed[0].Seat != 4 {
		t.Errorf("round 1 failures: got %+v, want seat 4", res.Failed)
	}
	if s.Seats[1].Estimate != 18000 || s.Seats[2].Estimate != 100 || s.Seats[3].Estimate != 0 {
		t.Errorf("estimates: got %d %d %d", s.Seats[1].Estimate, s.Seats[2].Estimate, s.Seats[3].Estimate)
	}

	// 已经出到剩余资金的人机不再加价; 剩余资金不够起拍价的人机没有出价记录, 估值照常更新
	s.Seats[2].Carry, s.Seats[2].Estimate = 99, 0
	res = s.closeRound()
	if want := []botBid{{Seat: 2}, {Seat: 4}}; !reflect.DeepEqual(res.Bots, want) {
		t.Errorf("round 2 bots: got %+v, want %+v", res.Bots, want)
	}
	if s.Seats[2].Estimate != 100 {
		t.Errorf("estimate of the seat short of carry: got %d, want 100", s.Seats[2].Estimate)
	}

	// 最后一轮只有座位 2 出过价, 没有第二名时按起拍价算; 座位 1 是真人, 没出价
	res = s.closeRound()
	if v := res.Verdict; v.Outcome != OutcomeSold || v.Winner != 2 || v.Price != 800 {
		t.Fatalf("round 3: got %+v, want sold to seat 2 at 800", res.Verdict)
	}
	if s.Seats[1].Carry != 0 {
		t.Errorf("winner carry %d, want 0", s.Seats[1].Carry)
	}
}

// TestWarehouseObservation 人机看到的是公开的信息: 其他人的出价在前 N-1 轮每轮结束后公开, 最后一轮看到的还是第 N-1 轮结束时的
func TestWarehouseObservation(t *testing.T) {
	var seen []Observation
	watcher := func(obs Observation, _ *rand.Rand) (Decision, error) {
		seen = append(seen, obs)
		return Decision{}, nil
	}
	s := testWarehouse(3, watcher)
	spec := testSpec(3, 50000, 50000)
	spec.Future, spec.LotsLeft, spec.Footprints = 700, 2, []Footprint{{X: 1, Y: 2, Width: 1, Height: 1}}
	s.openLot(spec)
	for round, bids := range [][]int64{{200, 0, 0}, {300, 250, 0}, {900, 800, 700}} {
		for i, amount := range bids {
			s.raise(i, amount, s.Clock())
		}
		if res := s.closeRound(); (res.Verdict.Outcome != "") != (round == 2) {
			t.Fatalf("round %d: got %+v", round+1, res.Verdict)
		}
	}
	others := [][]int64{{0, 0, 0}, {200, 0, 0}, {300, 250, 0}}
	if len(seen) != 3 {
		t.Fatalf("decided %d times, want 3", len(seen))
	}
	for i, obs := range seen {
		if obs.Round != int32(i+1) || !reflect.DeepEqual(obs.Others, others[i]) {
			t.Errorf("round %d: observed round %d, others %v, want %v", i+1, obs.Round, obs.Others, others[i])
		}
		if obs.Carry != 5000 || obs.Future != 700 || obs.LotsLeft != 2 || len(obs.Footprints) != 1 || obs.Rules.StartPrice != 100 || obs.Seed != 3 {
			t.Errorf("round %d: observation %+v", i+1, obs)
		}
	}
}

// TestWarehouseLots 下一个拍品从上一个的最后一轮之后开始, 剩余资金延续, 出价和估值清零, 热度向开局热度回落;
// 第一名领先够多时在最后一轮之前成交, 成交价按领先折让; 没有真人可以出价时不用等
func TestWarehouseLots(t *testing.T) {
	s := testWarehouse(2, fixedBid(400), fixedBid(0))
	first := testSpec(3, 20000, 20000)
	first.Rule.LeadDiscount = 5000
	s.openLot(first)
	if s.Hype != hypeStart {
		t.Errorf("first lot opened at hype %d, want %d", s.Hype, hypeStart)
	}
	s.raise(0, 1000, s.Clock())
	res := s.closeRound()
	// 第二名 400, 领先的 600 折让一半
	if v := res.Verdict; v.Outcome != OutcomeSold || v.Winner != 1 || v.Price != 700 {
		t.Fatalf("lot 1: got %+v, want a quick win for seat 1 at 700", res.Verdict)
	}

	s.Hype = 90
	s.Seats[1].Carry = 50
	second := testSpec(1)
	second.Rule.StartPrice = 4400
	s.openLot(second)
	if s.Round != 2 || s.lotRound() != 1 || s.Hype != 70 {
		t.Errorf("lot 2 opened: round %d, lot round %d, hype %d", s.Round, s.lotRound(), s.Hype)
	}
	for _, p := range s.Seats {
		if p.Bid != 0 || p.Estimate != 0 || !p.BidAt.IsZero() || p.shown != 0 {
			t.Errorf("seat %d not reset: %+v", p.Seat, p)
		}
	}
	// 座位 1 剩 4300, 座位 2 剩 50, 都不够起拍价
	if s.Seats[0].Carry != 4300 || s.humanCanBid() {
		t.Errorf("lot 2: seat 1 carry %d, human can bid %v", s.Seats[0].Carry, s.humanCanBid())
	}
	if res := s.closeRound(); res.Verdict.Outcome != OutcomeUnsold || res.Round != 2 {
		t.Errorf("lot 2: got %+v, want unsold in round 2", res)
	}

	quiet := testWarehouse(4)
	quiet.Hosted, quiet.Round, quiet.Hype = false, 5, 0
	quiet.openLot(testSpec(2, 20000))
	quiet.raise(0, 300, quiet.Clock())
	if res := quiet.closeRound(); quiet.Hype != 0 || res.Raises != 0 || res.Delta != 0 || res.Round != 6 {
		t.Errorf("without a host: hype %d, result %+v", quiet.Hype, res)
	}
}

// TestWarehouseResume 按出价记录恢复的拍品与一直在内存中推进的结果相同
func TestWarehouseResume(t *testing.T) {
	// 跟着公开的最高价加 100, 结果取决于恢复出的公开出价
	follower := func(obs Observation, _ *rand.Rand) (Decision, error) {
		best := int64(0)
		for _, other := range obs.Others {
			best = max(best, other)
		}
		return Decision{Bid: best + 100, Estimate: best}, nil
	}
	spec := testSpec(4, 50000, 50000, 50000)
	humans := [][]int64{{200, 0}, {0, 450}, {900, 0}, {0, 1200}}

	live := testWarehouse(2, follower, follower)
	live.Round = 6
	live.openLot(spec)
	var rows []Bid
	for n, bids := range humans {
		// 每一轮都从出价记录恢复一份, 与一直推进的那份比较
		resumed := testWarehouse(2, follower, follower)
		resumed.Round, resumed.Hype = live.Round, live.Hype
		for i, p := range live.Seats {
			q := &resumed.Seats[i]
			q.Carry, q.Bid, q.BidAt, q.Estimate = p.Carry, p.Bid, p.BidAt, p.Estimate
		}
		resumed.resumeLot(spec, live.firstRound, live.extra, live.contenders, rows)
		resumed.Clock = func() time.Time { return time.Unix(int64(100+n), 0) }
		live.Clock = resumed.Clock

		for i, amount := range bids {
			if amount > 0 {
				at := time.Unix(int64(n), int64(i))
				live.raise(i, amount, at)
				resumed.raise(i, amount, at)
				rows = append(rows, Bid{Round: live.Round, Seat: int16(i + 1), Amount: amount})
			}
		}
		round := live.Round
		got, want := resumed.closeRound(), live.closeRound()
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("round %d: resumed %+v, continuous %+v", n+1, got, want)
		}
		for i := range live.Seats {
			p, q := live.Seats[i], resumed.Seats[i]
			if p.Bid != q.Bid || p.shown != q.shown || p.before != q.before || p.Carry != q.Carry || !p.BidAt.Equal(q.BidAt) {
				t.Errorf("round %d seat %d: resumed %+v, continuous %+v", n+1, p.Seat, q, p)
			}
		}
		if live.Hype != resumed.Hype || live.Round != resumed.Round {
			t.Errorf("round %d: resumed hype %d round %d, continuous hype %d round %d", n+1, resumed.Hype, resumed.Round, live.Hype, live.Round)
		}
		for _, b := range want.Bots {
			rows = append(rows, Bid{Round: round, Seat: b.Seat, Amount: b.Amount})
		}
		if (want.Verdict.Outcome != "") != (n == len(humans)-1) {
			t.Fatalf("round %d: verdict %+v", n+1, want.Verdict)
		}
	}
}
