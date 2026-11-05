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
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
)

var testRule = venue.AuctionRule{StartPrice: 100, MaxCarry: 10000, Rounds: 3, QuickWin: []int32{20000, 16000}, MaxExtraRounds: 1}

func at(second int) time.Time {
	return time.Date(2026, 1, 1, 0, 0, second, 0, time.UTC)
}

func TestJudgeQuickWin(t *testing.T) {
	cases := []struct {
		name    string
		round   int32
		bidders []bidder
		want    verdict
	}{
		{"double wins round 1", 1, []bidder{{1, 1000, at(2)}, {2, 500, at(1)}}, verdict{Outcome: OutcomeSold, Winner: 1, Price: 1000}},
		{"below ratio continues", 1, []bidder{{1, 999, at(2)}, {2, 500, at(1)}}, verdict{}},
		{"single bidder continues", 1, []bidder{{1, 1000, at(1)}, {2, 0, time.Time{}}}, verdict{}},
		{"round 2 ratio", 2, []bidder{{3, 800, at(2)}, {4, 500, at(1)}}, verdict{Outcome: OutcomeSold, Winner: 3, Price: 800}},
		{"tie continues", 2, []bidder{{1, 800, at(2)}, {2, 800, at(1)}}, verdict{}},
	}
	for _, c := range cases {
		got := judge(&testRule, c.round, 0, c.bidders)
		if got.Outcome != c.want.Outcome || got.Winner != c.want.Winner || got.Price != c.want.Price || len(got.Contenders) != 0 {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestJudgeLastRound(t *testing.T) {
	if got := judge(&testRule, 3, 0, []bidder{{1, 0, time.Time{}}, {2, 0, time.Time{}}}); got.Outcome != OutcomeUnsold {
		t.Fatalf("no bids should be unsold, got %+v", got)
	}
	if got := judge(&testRule, 3, 0, []bidder{{1, 300, at(1)}, {2, 0, time.Time{}}}); got.Outcome != OutcomeSold || got.Winner != 1 || got.Price != 300 {
		t.Fatalf("single bid should sell, got %+v", got)
	}

	tied := []bidder{{4, 500, at(3)}, {2, 500, at(1)}, {1, 400, at(0)}}
	got := judge(&testRule, 3, 0, tied)
	if got.Outcome != "" || len(got.Contenders) != 2 || got.Contenders[0] != 2 || got.Contenders[1] != 4 {
		t.Fatalf("tie should add a round for seats 2 and 4, got %+v", got)
	}
	got = judge(&testRule, 4, 1, tied)
	if got.Outcome != OutcomeSold || got.Winner != 2 || got.Price != 500 {
		t.Fatalf("tie after extra rounds should go to the earlier bid, got %+v", got)
	}
}

func TestRankSeats(t *testing.T) {
	bidders := []bidder{{1, 900, at(1)}, {2, 700, at(2)}, {3, 1100, at(3)}, {4, 0, time.Time{}}}

	ranks := rankSeats(bidders, 1, 900, 1000)
	want := map[int16]int16{1: 1, 3: 2, 2: 3, 4: 4}
	for seat, rank := range want {
		if ranks[seat] != rank {
			t.Errorf("profitable winner: seat %d rank %d, want %d (%v)", seat, ranks[seat], rank, ranks)
		}
	}

	ranks = rankSeats(bidders, 1, 900, 800)
	want = map[int16]int16{2: 1, 3: 2, 4: 3, 1: 4}
	for seat, rank := range want {
		if ranks[seat] != rank {
			t.Errorf("losing winner: seat %d rank %d, want %d (%v)", seat, ranks[seat], rank, ranks)
		}
	}
}

func TestFeedbackFor(t *testing.T) {
	bids := map[int16]int64{1: 1000, 2: 800, 3: 800, 4: 0}
	cases := []struct {
		seat int16
		want Feedback
	}{
		{1, Feedback{Round: 1, Rank: 1, Hint: HintFarAhead}},
		{2, Feedback{Round: 1, Rank: 2, Hint: HintFarBehind}},
		{4, Feedback{Round: 1, Rank: 0, Hint: HintNoBid}},
	}
	for _, c := range cases {
		if got := feedbackFor(1, c.seat, bids); got != c.want {
			t.Errorf("seat %d: got %+v, want %+v", c.seat, got, c.want)
		}
	}

	if got := feedbackFor(2, 1, map[int16]int64{1: 1000, 2: 950}); got.Hint != HintAhead {
		t.Errorf("slightly ahead: got %+v", got)
	}
	if got := feedbackFor(2, 2, map[int16]int64{1: 1000, 2: 950}); got.Hint != HintClose || got.Rank != 2 {
		t.Errorf("slightly behind: got %+v", got)
	}
	if got := feedbackFor(2, 1, map[int16]int64{1: 1000, 2: 1000}); got.Hint != HintTied || got.Rank != 1 {
		t.Errorf("tied: got %+v", got)
	}
	if got := feedbackFor(2, 1, map[int16]int64{1: 1000}); got.Hint != HintFarAhead {
		t.Errorf("only bidder: got %+v", got)
	}
}

func TestBotBidAt(t *testing.T) {
	const length = 20 * time.Second
	fixed := func(v int64) func(int64) int64 { return func(int64) int64 { return v } }
	cases := []struct {
		name   string
		length time.Duration
		now    time.Time
		roll   int64
		want   time.Time
	}{
		{"no human to compare", 0, at(5), 0, at(5)},
		{"round timed out", length, at(20), int64(7 * time.Second), at(7)},
		{"round ended early", length, at(5), int64(3 * time.Second), at(3)},
		{"not reached yet", length, at(5), int64(9 * time.Second), at(5)},
	}
	for _, c := range cases {
		if got := botBidAt(at(0), c.length, c.now, fixed(c.roll)); !got.Equal(c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestScanOrders(t *testing.T) {
	pieces := []venue.Piece{{X: 2, Y: 1}, {X: 0, Y: 0}, {X: 3, Y: 0}, {X: 0, Y: 1}}
	want := []int32{4, 1, 2, 3}
	if got := scanOrders(pieces); !slices.Equal(got, want) {
		t.Fatalf("scanOrders = %v, want %v", got, want)
	}
}

func TestJudgeSpeedFloor(t *testing.T) {
	rule := testRule
	rule.SpeedFloor = 1000
	if got := judge(&rule, 1, 0, []bidder{{1, 999, at(2)}, {2, 400, at(1)}}); got.Outcome != "" {
		t.Errorf("below the floor should continue, got %+v", got)
	}
	if got := judge(&rule, 1, 0, []bidder{{1, 1000, at(2)}, {2, 400, at(1)}}); got.Outcome != OutcomeSold || got.Winner != 1 {
		t.Errorf("at the floor should sell, got %+v", got)
	}
	// 最后一轮不看下限
	if got := judge(&rule, 3, 0, []bidder{{1, 300, at(2)}, {2, 0, time.Time{}}}); got.Outcome != OutcomeSold || got.Price != 300 {
		t.Errorf("last round ignores the floor, got %+v", got)
	}
}

func TestSettlePrice(t *testing.T) {
	rule := testRule
	rule.LeadDiscount = 2000
	cases := []struct {
		name        string
		top, second int64
		want        int64
	}{
		{"discounted lead", 1000, 500, 900},
		{"no second bidder starts from the start price", 600, 0, 500},
		{"tie pays the bid", 800, 800, 800},
		{"start price only", 100, 0, 100},
	}
	for _, c := range cases {
		if got := settlePrice(&rule, c.top, c.second); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
	if got := settlePrice(&testRule, 1000, 500); got != 1000 {
		t.Errorf("no discount pays the top bid, got %d", got)
	}
	rule.QuickWin = []int32{20000, 16000}
	if got := judge(&rule, 1, 0, []bidder{{1, 1000, at(2)}, {2, 500, at(1)}}); got.Price != 900 {
		t.Errorf("quick win price: got %+v, want 900", got)
	}
}

func TestPreviewPieces(t *testing.T) {
	// 每件抽一次, 小于比例的扫出
	if got := previewPieces(4, 3000, sequence(0, 5000, 2999, 3000)); !slices.Equal(got, []bool{true, false, true, false}) {
		t.Errorf("by chance: got %v", got)
	}
	// 一件都没抽中时随机扫出一件
	if got := previewPieces(3, 3000, sequence(9999, 9999, 9999, 1)); !slices.Equal(got, []bool{false, true, false}) {
		t.Errorf("at least one: got %v", got)
	}
	if got := previewPieces(3, 0, first); slices.Contains(got, true) || len(got) != 3 {
		t.Errorf("rate 0: got %v", got)
	}
	if got := previewPieces(0, 3000, first); len(got) != 0 {
		t.Errorf("empty warehouse: got %v", got)
	}
	// 扫出的比例接近设置的比例
	rng := rand.New(rand.NewPCG(1, 1))
	picked := 0
	for range 1000 {
		for _, p := range previewPieces(10, 3000, rng.Int64N) {
			if p {
				picked++
			}
		}
	}
	if picked < 2900 || picked > 3200 {
		t.Errorf("rate: picked %d of 10000", picked)
	}
}

func TestSplitPool(t *testing.T) {
	rule := venue.AuctionRule{PoolRate: 7500, RebateRate: 4000, RebateCap: 300}
	pool, rebate, shares := splitPool(&rule, 1200, 500, []int16{3, 1, 4})
	// 900 + min(200, 300) = 1100, 按 3:2:1 分为 550、366、183, 零头 1 给第一个
	if pool != 1100 || rebate != 200 {
		t.Fatalf("pool %d rebate %d, want 1100 and 200", pool, rebate)
	}
	want := map[int16]int64{3: 551, 1: 366, 4: 183}
	if len(shares) != len(want) {
		t.Fatalf("shares %v, want %v", shares, want)
	}
	for seat, amount := range want {
		if shares[seat] != amount {
			t.Errorf("seat %d: got %d, want %d (%v)", seat, shares[seat], amount, shares)
		}
	}

	if _, rebate, _ := splitPool(&rule, 0, 5000, []int16{1}); rebate != 300 {
		t.Errorf("rebate should be capped at 300, got %d", rebate)
	}
	if pool, _, shares := splitPool(&rule, 1200, 0, nil); pool != 900 || shares != nil {
		t.Errorf("nobody eligible: got pool %d shares %v, want 900 burned", pool, shares)
	}
	if pool, _, shares := splitPool(&venue.AuctionRule{}, 1200, 500, []int16{1}); pool != 0 || shares != nil {
		t.Errorf("legacy rule has no pool, got %d %v", pool, shares)
	}
}
