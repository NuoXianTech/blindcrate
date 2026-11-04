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
	"encoding/json"
	"errors"
	"math"
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"strings"
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

// standardWarehouses 交接包验算器的标准图: 6×6 格子, 三个档位, 不波动, 最多重抽 3 次; 抽取和摆放的规则与 venue.Generator 相同, 不查库
type standardWarehouses struct {
	rng *rand.Rand
}

func newStandardWarehouses(seed uint64) warehouses {
	return &standardWarehouses{rng: rand.New(rand.NewPCG(seed, 7))}
}

func (s *standardWarehouses) Generate(valueFactor int32) (*venue.Warehouse, error) {
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
			warehouse := &venue.Warehouse{Pieces: pieces}
			for _, p := range pieces {
				warehouse.Value += p.SellPrice
			}
			return warehouse, nil
		}
	}
	return nil, errors.New(venue.WarehouseNotFit)
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

func (s *standardWarehouses) PieceOdds(valueFactor int32) []venue.PieceOdds {
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
	source := newStandardWarehouses(1)
	var values []int64
	var total int64
	for range venue.SimulationRuns {
		if warehouse, err := source.Generate(venue.BasisPoints); err == nil {
			values = append(values, warehouse.Value)
			total += warehouse.Value
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
		Warehouse:     venue.WarehouseRule{Grid: venue.Grid{Width: 6, Height: 6}},
		Auction:       venue.AuctionRule{Rounds: 5, RoundSeconds: 20, RevealSeconds: 5, QuickWin: []int32{23000, 18975, 18630, 17825}, MaxExtraRounds: 1},
		Pve:           venue.PveRule{ValueFactor: venue.BasisPoints},
	}
	v.Simulation.Ranked = standardStats()
	v.Auction.Calibrate(v.Simulation.Ranked, settings.Balance.Default())
	return v
}

// seedDeciders 初始化脚本中的人机风格, 准备好决策, 见 TestSeedPersonas
func seedDeciders(t *testing.T, codes ...string) []bot.Decider[Observation, Decision] {
	t.Helper()
	raw, err := os.ReadFile("../../db/migrator/migrations/000001_init.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, section, _ := strings.Cut(string(raw), "INSERT INTO bot_personas")
	section, _, _ = strings.Cut(section, ";")
	snapshots := map[string]*bot.Snapshot{}
	for _, m := range seedPersonaPattern.FindAllStringSubmatch(section, -1) {
		params := map[string]any{}
		if err := json.Unmarshal([]byte(m[5]), &params); err != nil {
			t.Fatalf("%s: %v", m[1], err)
		}
		snapshots[m[1]] = &bot.Snapshot{Code: m[1], Strategy: m[4], Params: params}
	}
	deciders := make([]bot.Decider[Observation, Decision], len(codes))
	for i, code := range codes {
		if snapshots[code] == nil {
			t.Fatalf("seed persona %s not found", code)
		}
		if deciders[i], err = bot.Prepare[Observation, Decision](snapshots[code]); err != nil {
			t.Fatal(err)
		}
	}
	return deciders
}

// standardSquads 参考阵容、2 个真人和 2 个人机、1 个真人和 3 个人机, 人机为初始化脚本中的三种风格, 权重相同
func standardSquads(t *testing.T) []squad {
	deciders := seedDeciders(t, "steady", "aggressive", "novice")
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

// TestSimulateStandardMap 交接包的标准图按推导出的参数验算: 参考阵容落在结构闸门内, 有人机的阵容估值不偏; 同样的输入结果相同
func TestSimulateStandardMap(t *testing.T) {
	v := standardVenue()
	b, err := simulate(v, newStandardWarehouses, standardSquads(t), 200, maxDiscount)
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
	for _, s := range b.Squads[1:] {
		if s.AppraiseBias < 8000 || s.AppraiseBias > 13000 {
			t.Errorf("%s appraise bias %d, want [8000, 13000]", s.Squad, s.AppraiseBias)
		}
	}

	again := standardVenue()
	replay, err := simulate(again, newStandardWarehouses, standardSquads(t), 200, maxDiscount)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(b, replay) || again.Auction.LeadDiscount != v.Auction.LeadDiscount {
		t.Errorf("same input gave different results:\n%+v\n%+v", b, replay)
	}
}

// TestSolveDiscount 取成交价 ÷ 仓库价值最接近目标的折让, 流拍的局不计入
func TestSolveDiscount(t *testing.T) {
	rule := venue.AuctionRule{StartPrice: 10}
	matches := []matchSim{
		{value: 1000, sold: true, top: 1200, second: 800},
		{value: 1000, sold: true, top: 900, second: 0},
		{value: 500, sold: false},
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
	if got := solveDiscount([]matchSim{{value: 100}}, rule, 9800, maxDiscount); got != 0 {
		t.Errorf("nothing sold: got %d, want 0", got)
	}
}

func TestHumanModelBid(t *testing.T) {
	rule := &venue.AuctionRule{StartPrice: 100, Rounds: 4, QuickWin: []int32{20000, 16000, 15000}}
	model := newHumanModel(settings.Balance.Default())
	model.JumpChance = 0
	rng := rand.New(rand.NewPCG(1, 1))
	if got := model.bid(rule, 1, 10000, 1, 100000, 0, nil, rng); got != rule.StartPrice {
		t.Errorf("round 1 without jump: got %d, want the start price", got)
	}
	// 第 1 轮偏高的估值在 [11470, 14200] 内, 抢拍按估值的 95%
	model.JumpChance = 1
	if got := model.bid(rule, 1, 10000, 1, 100000, 0, nil, rng); got < 10896 || got > 13490 {
		t.Errorf("round 1 jump: got %d, want [10896, 13490]", got)
	}
	if got := model.bid(rule, 2, 10000, 1, 500, 0, nil, rng); got != 500 {
		t.Errorf("capped by carry: got %d, want 500", got)
	}
	if got := model.bid(rule, 2, 10000, -1, 100000, 20000, nil, rng); got != 0 {
		t.Errorf("already above the estimate: got %d, want 0", got)
	}
}
