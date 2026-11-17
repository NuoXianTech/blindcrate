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
	"errors"
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
)

// standardPiece 标准图的一种藏品
type standardPiece struct {
	quality       int16
	width, height int16
	price         int64
}

// standardPieces 标准图的 7 级品质, 回收价为 120 × 品质倍率
var standardPieces = []standardPiece{
	{1, 1, 1, 120}, {2, 1, 1, 360}, {3, 1, 2, 960}, {4, 2, 2, 2640}, {5, 2, 2, 7200}, {6, 2, 3, 19200}, {7, 3, 3, 36000},
}

// standardSlot 标准图的一个档位, pool 为品质和权重
type standardSlot struct {
	min, max, chance int64
	pool             [][2]int64
}

var standardSlots = []standardSlot{
	{4, 8, 10000, [][2]int64{{1, 50}, {2, 35}, {3, 15}}},
	{1, 3, 8000, [][2]int64{{3, 40}, {4, 40}, {5, 20}}},
	{0, 1, 2500, [][2]int64{{5, 50}, {6, 35}, {7, 15}}},
}

// standardDepots 交接包验算器的标准图: 6×6 格子, 三个档位, 不波动, 最多重抽 3 次; 抽取和摆放的规则与 venue.Generator 相同, 不查库
type standardDepots struct {
	rng *rand.Rand
}

func newStandardDepots(seed uint64) *standardDepots {
	return &standardDepots{rng: rand.New(rand.NewPCG(seed, 7))}
}

// standardLineup 标准图一局一个拍品, 只有一种档次
type standardLineup struct{ *standardDepots }

func newStandardLineup(seed uint64) lineup { return standardLineup{newStandardDepots(seed)} }

func (standardLineup) Count() int { return 1 }
func (standardLineup) Pick() int  { return 0 }
func (standardLineup) Tiers() int { return 1 }
func (s standardLineup) Generate(_ int, valueFactor int32) (*venue.Depot, error) {
	return s.standardDepots.Generate(valueFactor)
}
func (s standardLineup) PieceOdds(_ int, valueFactor int32) []venue.PieceOdds {
	return s.standardDepots.PieceOdds(valueFactor)
}

func (s *standardDepots) Generate(valueFactor int32) (*venue.Depot, error) {
	for range 4 {
		var pieces []venue.Piece
		occupied := int64(0)
	draw:
		for _, slot := range standardSlots {
			if s.rng.Int64N(venue.BasisPoints) >= slot.chance {
				continue
			}
			scaled := (slot.min + s.rng.Int64N(slot.max-slot.min+1)) * int64(valueFactor)
			times := scaled / venue.BasisPoints
			if s.rng.Int64N(venue.BasisPoints) < scaled%venue.BasisPoints {
				times++
			}
			var total int64
			for _, entry := range slot.pool {
				total += entry[1]
			}
			for range times {
				n, quality := s.rng.Int64N(total), slot.pool[0][0]
				for _, entry := range slot.pool {
					if n < entry[1] {
						quality = entry[0]
						break
					}
					n -= entry[1]
				}
				p := standardPieces[quality-1]
				pieces = append(pieces, venue.Piece{ItemID: uint64(quality), Quality: p.quality, Width: p.width, Height: p.height, SellCurrency: "chip", SellPrice: p.price})
				if occupied += int64(p.width * p.height); occupied > 36 {
					break draw
				}
			}
		}
		if standardPlace(pieces) {
			depot := &venue.Depot{Pieces: pieces}
			for _, p := range pieces {
				depot.Value += p.SellPrice
			}
			return depot, nil
		}
	}
	return nil, errors.New(venue.DepotNotFit)
}

// standardPlace 按面积从大到小, 从上到下、从左到右找第一个放得下的位置
func standardPlace(pieces []venue.Piece) bool {
	if len(pieces) == 0 {
		return false
	}
	slices.SortStableFunc(pieces, func(a, b venue.Piece) int { return cmp.Compare(b.Width*b.Height, a.Width*a.Height) })
	var used [6][6]bool
	fits := func(x, y, w, h int16) bool {
		if x+w > 6 || y+h > 6 {
			return false
		}
		for dy := range h {
			for dx := range w {
				if used[y+dy][x+dx] {
					return false
				}
			}
		}
		return true
	}
	for i := range pieces {
		p := &pieces[i]
		placed := false
		for y := int16(0); y < 6 && !placed; y++ {
			for x := int16(0); x < 6 && !placed; x++ {
				if fits(x, y, p.Width, p.Height) {
					for dy := range p.Height {
						for dx := range p.Width {
							used[y+dy][x+dx] = true
						}
					}
					p.X, p.Y, placed = x, y, true
				}
			}
		}
		if !placed {
			return false
		}
	}
	return true
}

func (s *standardDepots) PieceOdds(valueFactor int32) []venue.PieceOdds {
	counts := make([]float64, len(standardPieces))
	for _, slot := range standardSlots {
		times := float64(slot.chance) / venue.BasisPoints * float64(slot.min+slot.max) / 2 * float64(valueFactor) / venue.BasisPoints
		var total int64
		for _, entry := range slot.pool {
			total += entry[1]
		}
		for _, entry := range slot.pool {
			counts[entry[0]-1] += times * float64(entry[1]) / float64(total)
		}
	}
	var odds []venue.PieceOdds
	for i, p := range standardPieces {
		if counts[i] > 0 {
			odds = append(odds, venue.PieceOdds{Quality: p.quality, Width: p.width, Height: p.height, Count: counts[i], Price: float64(p.price)})
		}
	}
	return odds
}

// standardStats 标准图排位模式的价值分布, 同 venue.Generator.Simulate
func standardStats() venue.ValueStats {
	source := newStandardDepots(1)
	var values []int64
	var total int64
	for range venue.SimulationRuns {
		if depot, err := source.Generate(venue.BasisPoints); err == nil {
			values = append(values, depot.Value)
			total += depot.Value
		}
	}
	slices.Sort(values)
	mean := float64(total) / float64(len(values))
	var squares float64
	for _, v := range values {
		squares += (float64(v) - mean) * (float64(v) - mean)
	}
	return venue.ValueStats{
		Mean: total / int64(len(values)),
		P50:  values[(len(values)-1)*50/100],
		P90:  values[(len(values)-1)*90/100],
		CV:   int32(math.Round(math.Sqrt(squares/float64(len(values))) / mean * venue.BasisPoints)),
	}
}

// standardVenue 标准图, 5 轮, 速胜倍率为交接包推荐的 1.15 × [2.00, 1.65, 1.62, 1.55]
func standardVenue() *venue.Venue {
	v := &venue.Venue{
		Code:          "standard",
		BidCurrency:   "chip",
		PveEnabled:    true,
		RankedEnabled: true,
		Lots:          venue.LotRule{Min: 1, Max: 1, Tiers: []venue.Tier{{Code: "standard", Name: "标准", Weight: 1, Depot: venue.DepotRule{Grid: venue.Grid{Width: 6, Height: 6}}}}},
		Auction:       venue.AuctionRule{Rounds: 5, RoundSeconds: 20, RevealSeconds: 5, QuickWin: []int32{23000, 18975, 18630, 17825}, MaxExtraRounds: 1},
		Pve:           venue.PveRule{ValueFactor: venue.BasisPoints},
	}
	v.Simulation.Ranked = standardStats()
	v.Simulation.Tiers = []venue.TierStats{{Code: "standard", Ranked: v.Simulation.Ranked}}
	v.Calibrate(settings.Balance.Default())
	return v
}

// referenceDeciders 参考风格, 准备好决策, 见 referencePersonas
func referenceDeciders(t *testing.T, codes ...string) []bot.Decider[Observation, Decision] {
	t.Helper()
	snapshots := map[string]*bot.Snapshot{}
	for _, p := range referencePersonas() {
		snapshot := p.Snapshot()
		snapshots[p.Code] = &snapshot
	}
	deciders := make([]bot.Decider[Observation, Decision], len(codes))
	for i, code := range codes {
		if snapshots[code] == nil {
			t.Fatalf("reference persona %s not found", code)
		}
		decider, err := bot.Prepare[Observation, Decision](snapshots[code])
		if err != nil {
			t.Fatal(err)
		}
		deciders[i] = decider
	}
	return deciders
}

// standardSquads 参考阵容、2 个真人和 2 个人机、1 个真人和 3 个人机, 人机为三种参考风格, 权重相同
func standardSquads(t *testing.T) []squad {
	deciders := referenceDeciders(t, "steady", "aggressive", "novice")
	picks := []int64{1, 1, 1}
	human := newHumanModel(settings.Balance.Default())
	return []squad{
		{code: venue.SquadHumans, human: human, humans: 4, valueFactor: venue.BasisPoints},
		{code: venue.SquadRanked, human: human, humans: 2, valueFactor: venue.BasisPoints, bots: deciders, picks: picks},
		{code: venue.SquadPve, human: human, humans: 1, valueFactor: venue.BasisPoints, bots: deciders, picks: picks},
	}
}

// maxDiscount 默认的领先折让上限
const maxDiscount = 3000

// TestSimulateStandardMap 交接包的标准图按推导出的参数验算: 参考阵容落在结构闸门内, 真人盈亏等于目标, 有人机的阵容估值不偏,
// 并另外模拟了盲狙和知情狙击; 同样的输入结果相同
func TestSimulateStandardMap(t *testing.T) {
	v := standardVenue()
	seatEV := scale(v.Simulation.Ranked.Mean, int64(settings.Balance.Default().TargetSeatEV))
	b, err := simulate(v, newStandardLineup, standardSquads(t), simConfig{sellFee: 200, maxDiscount: maxDiscount, seatEV: seatEV, carryFloor: venue.BasisPoints})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("value %+v, rule %+v", v.Simulation.Ranked, v.Auction)
	for _, s := range b.Squads {
		t.Logf("%+v", s)
	}

	if d := v.Auction.LeadDiscount; d <= 0 || d >= maxDiscount {
		t.Errorf("lead discount %d, want strictly inside (0, %d)", d, maxDiscount)
	}
	humans := b.Squads[0]
	if gap := humans.PriceValue - b.Target; gap < -300 || gap > 300 {
		t.Errorf("reference price/value %d, want %d ± 300", humans.PriceValue, b.Target)
	}
	if humans.SpeedWin < 800 || humans.SpeedWin > 4500 {
		t.Errorf("reference speed win %d, want [800, 4500]", humans.SpeedWin)
	}
	if humans.BidCV < 1200 {
		t.Errorf("reference bid cv %d, want >= 1200", humans.BidCV)
	}
	if humans.TopShare > 6000 {
		t.Errorf("reference top share %d, want <= 6000", humans.TopShare)
	}
	if humans.WinRate < 2400 || humans.WinRate > 2600 {
		t.Errorf("reference win rate %d, want about 2500", humans.WinRate)
	}
	if v.Auction.EntryFee <= 0 {
		t.Errorf("entry fee %d, want > 0", v.Auction.EntryFee)
	}
	if gap := humans.SeatEV - seatEV; gap < -1 || gap > 1 {
		t.Errorf("reference seat ev %d, want %d ± 1", humans.SeatEV, seatEV)
	}
	if humans.SnipeEV != 0 || humans.InsiderEV != 0 || humans.CheapBlocked != 0 {
		t.Errorf("reference squad should have no sniper runs: %+v", humans)
	}
	for _, s := range b.Squads[1:] {
		if s.AppraiseBias < 8000 || s.AppraiseBias > 13000 {
			t.Errorf("%s appraise bias %d, want [8000, 13000]", s.Squad, s.AppraiseBias)
		}
		// 知情狙击不会买亏, 每局最多亏掉入场费; 盲狙不看货仓, 赚得不会比它多
		if s.InsiderEV < -v.Auction.EntryFee || s.InsiderEV <= s.SnipeEV {
			t.Errorf("%s insider ev %d, blind sniper ev %d, entry fee %d", s.Squad, s.InsiderEV, s.SnipeEV, v.Auction.EntryFee)
		}
	}

	again := standardVenue()
	replay, err := simulate(again, newStandardLineup, standardSquads(t), simConfig{sellFee: 200, maxDiscount: maxDiscount, seatEV: seatEV, carryFloor: venue.BasisPoints})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(b, replay) || again.Auction.LeadDiscount != v.Auction.LeadDiscount || again.Auction.EntryFee != v.Auction.EntryFee {
		t.Errorf("same input gave different results:\n%+v\n%+v", b, replay)
	}
}

// TestSolveDiscount 取成交价 ÷ 货仓价值最接近目标的折让, 流拍的局不计入
func TestSolveDiscount(t *testing.T) {
	rule := venue.AuctionRule{StartPrice: 10}
	matches := []matchSim{
		{lots: []lotSim{{value: 1000, sold: true, top: 1200, second: 800}, {value: 1000, sold: true, top: 900, second: 0}}},
		{lots: []lotSim{{value: 500, sold: false}}},
	}
	// 不打折时为 (1200 + 900) / 2000 = 10500; 折让 11% 时为 (800 + 356 + 10 + 792) / 2000 = 9790, 最接近 9800
	if got := solveDiscount(matches, rule, 9800, maxDiscount); got != 1100 {
		t.Errorf("got %d, want 1100", got)
	}
	if got := solveDiscount(matches, rule, 11000, maxDiscount); got != 0 {
		t.Errorf("target above undiscounted: got %d, want 0", got)
	}
	if got := solveDiscount(matches, rule, 1000, maxDiscount); got != maxDiscount {
		t.Errorf("target below the floor: got %d, want %d", got, maxDiscount)
	}
	if got := solveDiscount(matches, rule, 9800, 0); got != 0 {
		t.Errorf("discount turned off: got %d, want 0", got)
	}
	if got := solveDiscount([]matchSim{{lots: []lotSim{{value: 100}}}}, rule, 9800, maxDiscount); got != 0 {
		t.Errorf("nothing sold: got %d, want 0", got)
	}
}

// TestSolveEntryFee 入场费让每个真人席位每局的平均盈亏等于目标, 流拍的局退还入场费
func TestSolveEntryFee(t *testing.T) {
	sq := &squad{code: venue.SquadRanked, humans: 2}
	rule := venue.AuctionRule{StartPrice: 10, EntryFee: 999}
	matches := []matchSim{
		{lots: []lotSim{{value: 1000, sold: true, top: 900, second: 800, winner: 1}}, carry: make([]int64, 4), planner: make([]bool, 4)},
		{lots: []lotSim{{value: 500, sold: false}}, carry: make([]int64, 4), planner: make([]bool, 4)},
	}
	// 不收入场费时成交者赚 980 - 900 = 80, 平均到 2 局 × 2 个真人为 20; 只有一半的局收入场费,
	// 要平均每人亏 10 需要 (20 + 10) ÷ 50% = 60
	if got := solveEntryFee(sq, matches, rule, 200, -10); got != 60 {
		t.Errorf("got %d, want 60", got)
	}
	rule.EntryFee = 60
	if got := sq.summarize(matches, &rule, 200).SeatEV; got != -10 {
		t.Errorf("seat ev with the solved fee %d, want -10", got)
	}
	if got := solveEntryFee(sq, matches, rule, 200, 30); got != 0 {
		t.Errorf("target above the gross: got %d, want 0", got)
	}
	if got := solveEntryFee(sq, matches[1:], rule, 200, -10); got != 0 {
		t.Errorf("nothing sold: got %d, want 0", got)
	}
}

// TestSniperBid 狙击只在最后一轮和加轮出价: 盲狙比别人已公开的最高价高一成, 只受剩余资金限制; 知情狙击不超过卖掉货仓能收回的钱
func TestSniperBid(t *testing.T) {
	rule := &venue.AuctionRule{StartPrice: 100, Rounds: 4}
	blind := &sniper{Factor: snipeFactor}
	insider := &sniper{Factor: insiderFactor, Keep: 9800}
	cases := []struct {
		name   string
		sniper *sniper
		round  int32
		value  int64
		view   humanView
		want   int64
	}{
		{"最后一轮之前不出价", blind, 3, 5000, humanView{carry: 10000, best: 1000}, 0},
		{"最后一轮比公开的最高价高一成", blind, 4, 5000, humanView{carry: 10000, best: 1000}, 1100},
		{"不看货仓值多少", blind, 4, 500, humanView{carry: 10000, best: 1000}, 1100},
		{"没人出过价时按起拍价", blind, 4, 5000, humanView{carry: 10000}, 110},
		{"不超过剩余资金", blind, 4, 5000, humanView{carry: 1050, best: 1000}, 1050},
		{"资金不到起拍价", blind, 4, 5000, humanView{carry: 99}, 0},
		{"加轮再加 5%", blind, 5, 5000, humanView{carry: 10000, previous: 1100, best: 1100}, 1155},
		{"加轮时资金用完", blind, 5, 5000, humanView{carry: 1100, previous: 1100, best: 1100}, 0},
		{"知情时多加一些", insider, 4, 5000, humanView{carry: 10000, best: 1000}, 1250},
		{"知情时不超过能收回的钱", insider, 4, 1200, humanView{carry: 10000, best: 1000}, 1176},
		{"知情时加到能收回的钱为止", insider, 5, 1200, humanView{carry: 10000, previous: 1176, best: 1176}, 0},
		{"知情时不值起拍价的不拍", insider, 4, 100, humanView{carry: 10000}, 0},
	}
	for _, c := range cases {
		if got := c.sniper.bid(rule, c.round, c.value, c.view); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
	// 出价很低时加轮至少加 1
	if got := blind.bid(&venue.AuctionRule{StartPrice: 1, Rounds: 4}, 5, 5000, humanView{carry: 10000, previous: 10, best: 10}); got != 11 {
		t.Errorf("small raise: got %d, want 11", got)
	}
}

// TestSeatEV 一个座位每局的平均盈亏与 summarize 的口径相同: 拍下的货仓按扣掉出售手续费后的回收价减成交价, 至少成交一个拍品的仓库扣入场费
func TestSeatEV(t *testing.T) {
	rule := venue.AuctionRule{StartPrice: 10, EntryFee: 60}
	seats := func() ([]int64, []bool) { return make([]int64, playerCount), make([]bool, playerCount) }
	matches := []matchSim{
		{lots: []lotSim{{value: 1000, sold: true, top: 900, second: 800, winner: 1}, {value: 2000, sold: true, top: 500, second: 400, winner: 2}}},
		{lots: []lotSim{{value: 500}}},
		{lots: []lotSim{{value: 800, sold: true, top: 700, winner: 3}}},
	}
	for i := range matches {
		matches[i].carry, matches[i].planner = seats()
	}
	// 1 号拍下一个赚 980 - 900 = 80, 三局里两局有成交, 各扣 60
	if got := seatEV(matches, 1, &rule, 200); got != -13 {
		t.Errorf("seat 1: got %d, want -13", got)
	}
	// 2 号拍下一个赚 1960 - 500 = 1460
	if got := seatEV(matches, 2, &rule, 200); got != 447 {
		t.Errorf("seat 2: got %d, want 447", got)
	}
	// 什么都没拍到也要交入场费
	if got := seatEV(matches, 4, &rule, 200); got != -40 {
		t.Errorf("seat 4: got %d, want -40", got)
	}
	sq := &squad{code: venue.SquadPve, humans: 1}
	if got := sq.summarize(matches, &rule, 200).SeatEV; got != -13 {
		t.Errorf("summarize: got %d, want -13", got)
	}
	// 成交价按领先折让: 折让一半时 1 号按 850 成交
	rule.LeadDiscount = 5000
	if got := seatEV(matches, 1, &rule, 200); got != 3 {
		t.Errorf("with the lead discount: got %d, want 3", got)
	}
	if got := seatEV(nil, 1, &rule, 200); got != 0 {
		t.Errorf("no matches: got %d, want 0", got)
	}
}

// TestCheapBlocked 只看每局第一个拍品里价值不到公示期望六成的, 人机的最高出价不低于卖掉货仓能收回的钱时算拍不到
func TestCheapBlocked(t *testing.T) {
	matches := []matchSim{
		{lots: []lotSim{{value: 500, expected: 1000, bots: 490}}}, // 能收回 490, 拍不到
		{lots: []lotSim{{value: 500, expected: 1000, bots: 489}}},
		{lots: []lotSim{{value: 599, expected: 1000}, {value: 100, expected: 1000, bots: 900}}}, // 第二个拍品不看
		{lots: []lotSim{{value: 600, expected: 1000, bots: 9999}}},                              // 不算便宜
		{lots: []lotSim{{value: 0, expected: 1000, bots: 100}}},                                 // 空货仓不看
		{},
	}
	if got := cheapBlocked(matches, 200); got != 3333 {
		t.Errorf("got %d, want 3333", got)
	}
	if got := cheapBlocked(matches[3:], 200); got != 0 {
		t.Errorf("no cheap depots: got %d, want 0", got)
	}
}

func TestHumanModelBid(t *testing.T) {
	rule := &venue.AuctionRule{StartPrice: 100, Rounds: 4, QuickWin: []int32{20000, 16000, 15000}}
	model := newHumanModel(settings.Balance.Default())
	model.JumpChance = 0
	rng := rand.New(rand.NewPCG(1, 1))
	if got := model.bid(rule, 1, 10000, 1, humanView{carry: 100000, future: -1}, rng); got != rule.StartPrice {
		t.Errorf("round 1 without jump: got %d, want the start price", got)
	}
	// 第 1 轮偏高的估值在 [11470, 14200] 内, 抢拍按估值的 95%
	model.JumpChance = 1
	if got := model.bid(rule, 1, 10000, 1, humanView{carry: 100000, future: -1}, rng); got < 10896 || got > 13490 {
		t.Errorf("round 1 jump: got %d, want [10896, 13490]", got)
	}
	if got := model.bid(rule, 2, 10000, 1, humanView{carry: 500, future: -1}, rng); got != 500 {
		t.Errorf("capped by carry: got %d, want 500", got)
	}
	if got := model.bid(rule, 2, 10000, -1, humanView{carry: 100000, previous: 20000, future: -1}, rng); got != 0 {
		t.Errorf("already above the estimate: got %d, want 0", got)
	}
	if got := model.bid(rule, 2, 10000, 1, humanView{carry: 100000, best: 100000, future: -1}, rng); got != 0 {
		t.Errorf("outbid beyond the limit: got %d, want 0", got)
	}
	// 规划预算: 剩余资金 1000, 后面拍品的期望是这个拍品的 9 倍左右, 偏高的估值不超过 1.42 倍, 最多花 1000 × 1.42 ÷ 10.42 × 预算系数
	if got := model.bid(rule, 2, 10000, 1, humanView{carry: 1000, future: 90000}, rng); got > 1000*142/1042*model.BudgetPace/venue.BasisPoints {
		t.Errorf("planner budget: got %d, want at most %d", got, 1000*142/1042*model.BudgetPace/venue.BasisPoints)
	}
}
