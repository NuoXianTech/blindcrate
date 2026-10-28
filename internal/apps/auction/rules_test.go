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

func TestPickBotLevel(t *testing.T) {
	weights := []int32{0, 3, 1}
	for n, want := range map[int64]int16{0: 2, 2: 2, 3: 3} {
		if got := pickBotLevel(weights, func(int64) int64 { return n }); got != want {
			t.Errorf("n=%d: got level %d, want %d", n, got, want)
		}
	}
}

func TestBotBid(t *testing.T) {
	bots := venue.BotRule{RoundFactors: []int32{5000, 8000}, Jitter: 0, MaxFactor: 9000, BluffChance: 10000, BluffRounds: 1, BluffFactor: 2000}
	zero := func(int64) int64 { return 0 }

	if got := botBid(&bots, &testRule, 1, 1000, 0, 10000, zero); got != 200 {
		t.Errorf("bluff round: got %d, want 200", got)
	}
	if got := botBid(&bots, &testRule, 2, 1000, 0, 10000, zero); got != 800 {
		t.Errorf("round 2: got %d, want 800", got)
	}
	if got := botBid(&bots, &testRule, 3, 1000, 0, 10000, zero); got != 800 {
		t.Errorf("round 3 should reuse the last factor: got %d, want 800", got)
	}
	if got := botBid(&bots, &testRule, 2, 1000, 0, 600, zero); got != 600 {
		t.Errorf("capped by carry: got %d, want 600", got)
	}
	if got := botBid(&bots, &testRule, 2, 1000, 850, 10000, zero); got != 850 {
		t.Errorf("never below previous: got %d, want 850", got)
	}
	if got := botBid(&bots, &testRule, 2, 100, 0, 10000, zero); got != 0 {
		t.Errorf("limit below start price should pass: got %d, want 0", got)
	}
}
