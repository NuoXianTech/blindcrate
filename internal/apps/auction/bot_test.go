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
	"encoding/json"
	"math"
	"math/rand/v2"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
)

// testOdds 四种藏品, 全部藏品的平均价为 2900 / 6.5
var testOdds = []venue.PieceOdds{
	{Quality: 1, Width: 1, Height: 1, Count: 4, Price: 50},
	{Quality: 1, Width: 2, Height: 1, Count: 1, Price: 200},
	{Quality: 3, Width: 1, Height: 1, Count: 1, Price: 1000},
	{Quality: 3, Width: 1, Height: 2, Count: 0.5, Price: 3000},
}

func TestAppraise(t *testing.T) {
	average := 2900 / 6.5
	cases := []struct {
		name       string
		perception string
		footprints []Footprint
		rate       int32
		want       float64
	}{
		{"size", PerceptionSize, []Footprint{{Width: 1, Height: 1}}, 10000, 240},
		{"size by shape", PerceptionSize, []Footprint{{Width: 2, Height: 1}, {Width: 1, Height: 2}}, 10000, 3200},
		{"size unknown shape", PerceptionSize, []Footprint{{Width: 3, Height: 3}}, 10000, average},
		{"area merges shapes", PerceptionArea, []Footprint{{Width: 1, Height: 2}}, 10000, 1700 / 1.5},
		{"quality ignores size", PerceptionQuality, []Footprint{{Width: 1, Height: 2}}, 10000, average},
		{"nothing previewed", PerceptionSize, nil, 0, 2900},
		{"unseen at the expected value", PerceptionSize, []Footprint{{Width: 1, Height: 1}}, 3000, 240 + 2900*0.7},
	}
	for _, c := range cases {
		if got := appraise(testOdds, c.footprints, c.perception, c.rate); math.Abs(got-c.want) > 1e-6 {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	if got := appraise(nil, []Footprint{{Width: 1, Height: 1}}, PerceptionSize, 3000); got != 0 {
		t.Errorf("no odds: got %v, want 0", got)
	}
}

// flatObservation 10 件 1×1 的藏品, 全部扫出轮廓, 每件估价 1000, 没有估值偏差时估值为 10000; others 为其他座位已公开的出价
func flatObservation(round int32, bid int64, others ...int64) Observation {
	obs := Observation{
		Round:      round,
		Rules:      testRule,
		Footprints: make([]Footprint, 10),
		Odds:       []venue.PieceOdds{{Quality: 1, Width: 1, Height: 1, Count: 10, Price: 1000}},
		Carry:      testRule.MaxCarry,
		Bid:        bid,
		Others:     append([]int64{}, others...),
		Seed:       1,
	}
	obs.Rules.PreviewRate = venue.BasisPoints
	for i := range obs.Footprints {
		obs.Footprints[i] = Footprint{X: int16(i), Width: 1, Height: 1}
	}
	return obs
}

// exactParams 没有随机成分的参数: 估值 10000 时上限为 9000, 第 1-3 轮按 4500、6250、8000 出价
func exactParams() basicParams {
	return basicParams{Perception: PerceptionSize, OpenFactor: 4500, CloseFactor: 8000, MaxFactor: 9000, JumpFactor: 9500, Chase: 1500, FollowRate: venue.BasisPoints, FoldMargin: 1000}
}

func TestDecideBasic(t *testing.T) {
	cases := []struct {
		name      string
		obs       Observation
		modify    func(p *basicParams, obs *Observation)
		low, high int64 // 出价的范围, 0 表示不加价
	}{
		{"round 1 opens", flatObservation(1, 0), nil, 4500, 4500},
		{"round 2 interpolates", flatObservation(2, 4500, 4000), nil, 6250, 6250},
		{"last round closes", flatObservation(3, 6250, 6000), nil, 8000, 8000},
		{"single round closes", flatObservation(1, 0), func(_ *basicParams, obs *Observation) {
			obs.Rules.Rounds, obs.Rules.QuickWin = 1, nil
		}, 8000, 8000},
		{"jitter", flatObservation(1, 0), func(p *basicParams, _ *Observation) { p.Jitter = 1000 }, 4050, 4950},
		{"start price floor", flatObservation(1, 0), func(_ *basicParams, obs *Observation) { obs.Rules.StartPrice = 5000 }, 5000, 5000},
		{"carry caps", flatObservation(3, 4000, 3900), func(_ *basicParams, obs *Observation) { obs.Carry = 5000 }, 5000, 5000},
		{"limit below start price", flatObservation(1, 0), func(_ *basicParams, obs *Observation) { obs.Carry = 50 }, 0, 0},
		{"limit reached", flatObservation(3, 9000, 8900), nil, 0, 0},
		{"far ahead adds half", flatObservation(2, 4500, 3000), nil, 5375, 5375},
		{"alone adds half", flatObservation(2, 4500, 0, 0, 0), nil, 5375, 5375},
		{"far ahead never lowers", flatObservation(3, 8500, 5000), nil, 0, 0},
		// 最高价 6000 是自己的 1.2 倍, 追到 6001 之上 15%
		{"far behind chases", flatObservation(2, 5000, 6000, 1000), nil, 6901, 6901},
		{"no follow", flatObservation(2, 5000, 6000), func(p *basicParams, _ *Observation) { p.FollowRate = 0 }, 6250, 6250},
		// 最高价够到上限 9000 时放弃
		{"behind the limit gives up", flatObservation(2, 8000, 9000), nil, 0, 0},
		// 上限放到 1.5 倍, 最高价超过估值 10% 时放弃
		{"fold above the estimate", flatObservation(2, 8000, 11001), func(p *basicParams, _ *Observation) { p.MaxFactor, p.CloseFactor = 15000, 15000 }, 0, 0},
		{"close raises", flatObservation(2, 7000, 7500), nil, 7726, 8251},
		{"tied raises", flatObservation(2, 7000, 7000), nil, 7210, 7700},
		{"extra round", flatObservation(4, 6000, 6000), nil, 6900, 8100},
		{"jump", flatObservation(1, 0), func(p *basicParams, _ *Observation) { p.JumpChance, p.JumpFactor = venue.BasisPoints, 8800 }, 8800, 8800},
		{"jump only in round 1", flatObservation(2, 4500, 4000), func(p *basicParams, _ *Observation) { p.JumpChance = venue.BasisPoints }, 6250, 6250},
		// 上限 9000 ÷ 速胜倍率 2 倍
		{"guard round 1", flatObservation(1, 0), func(p *basicParams, _ *Observation) { p.Guard, p.OpenFactor = true, 3000 }, 4500, 4500},
		// 上限 9000 ÷ 速胜倍率 1.6 倍
		{"guard round 2", flatObservation(2, 4500, 4000), func(p *basicParams, _ *Observation) { p.Guard, p.OpenFactor = true, 3000 }, 5625, 5625},
		{"no guard in last round", flatObservation(3, 5625, 5000), func(p *basicParams, _ *Observation) {
			p.Guard, p.OpenFactor, p.CloseFactor = true, 3000, 6000
		}, 6000, 6000},
		// 剩余资金 10000, 后面拍品的期望也是 10000, 这个拍品最多花一半
		{"budget", flatObservation(3, 4000, 3900), func(p *basicParams, obs *Observation) {
			p.BudgetPace, obs.Carry, obs.Future, obs.LotsLeft = venue.BasisPoints, 10000, 10000, 1
		}, 5000, 5000},
		{"budget pace", flatObservation(3, 4000, 3900), func(p *basicParams, obs *Observation) {
			p.BudgetPace, obs.Carry, obs.Future, obs.LotsLeft = 15000, 10000, 10000, 1
		}, 7500, 7500},
		{"last lot spends all", flatObservation(3, 4000, 3900), func(p *basicParams, obs *Observation) {
			p.BudgetPace, obs.Carry, obs.Future, obs.LotsLeft = venue.BasisPoints, 10000, 0, 0
		}, 8000, 8000},
	}
	for _, c := range cases {
		for seed := range uint64(100) {
			p, obs := exactParams(), c.obs
			if c.modify != nil {
				c.modify(&p, &obs)
			}
			got := decideBasic(obs, &p, rand.New(rand.NewPCG(seed, uint64(obs.Round))))
			if got.Bid < c.low || got.Bid > c.high || got.Estimate != 10000 {
				t.Errorf("%s: seed %d got %+v, want bid in [%d, %d] and estimate 10000", c.name, seed, got, c.low, c.high)
				break
			}
		}
	}
}

// TestDecideBasicBias 估值偏差由人机的种子决定, 整局不变, 与每轮的随机数无关
func TestDecideBasicBias(t *testing.T) {
	p := exactParams()
	p.Bias = 1000
	seen := map[int64]bool{}
	for seed := range int64(200) {
		obs := flatObservation(1, 0)
		obs.Seed = seed
		first := decideBasic(obs, &p, rand.New(rand.NewPCG(1, 1)))
		if first.Estimate < 9000 || first.Estimate > 11000 {
			t.Fatalf("seed %d: estimate %d out of 10000 ± 10%%", seed, first.Estimate)
		}
		obs.Round, obs.Bid = 3, first.Bid
		obs.Others = []int64{1}
		if later := decideBasic(obs, &p, rand.New(rand.NewPCG(7, 3))); later.Estimate != first.Estimate {
			t.Fatalf("seed %d: estimate changed from %d to %d", seed, first.Estimate, later.Estimate)
		}
		seen[first.Estimate] = true
	}
	if len(seen) < 100 {
		t.Errorf("only %d distinct estimates over 200 seeds", len(seen))
	}
}

// TestDecideSnapshot 按风格快照调用基础策略: 缺少的参数取默认值, 相同的随机数得到相同的决策
func TestDecideSnapshot(t *testing.T) {
	snapshot := &bot.Snapshot{Code: "test", Strategy: StrategyBasic, Params: map[string]any{
		"bias": 0, "open_factor": 4000, "close_factor": 8000, "max_factor": 9000, "jitter": 500, "jump_chance": 0, "guard": false,
	}}
	obs := flatObservation(1, 0)
	first, err := bot.Decide[Observation, Decision](snapshot, obs, rand.New(rand.NewPCG(1, 1)))
	if err != nil {
		t.Fatal(err)
	}
	if first.Estimate != 10000 || first.Bid < 3800 || first.Bid > 4200 {
		t.Fatalf("got %+v, want estimate 10000 and bid 4000 ± 5%%", first)
	}
	again, err := bot.Decide[Observation, Decision](snapshot, obs, rand.New(rand.NewPCG(1, 1)))
	if err != nil || again != first {
		t.Fatalf("same rng: got %+v, %v, want %+v", again, err, first)
	}
}

func TestBasicParamsValidate(t *testing.T) {
	cases := []struct {
		name    string
		factors [3]int32 // 首轮、末轮、上限系数
		want    string
	}{
		{"increasing", [3]int32{4500, 8000, 9000}, ""},
		{"equal", [3]int32{8000, 8000, 8000}, ""},
		{"open above close", [3]int32{8500, 8000, 9000}, BotFactorsInvalid},
		{"close above max", [3]int32{4500, 9500, 9000}, BotFactorsInvalid},
	}
	for _, c := range cases {
		p := basicParams{OpenFactor: c.factors[0], CloseFactor: c.factors[1], MaxFactor: c.factors[2]}
		if got, err := p.Validate(nil); err != nil || got != c.want {
			t.Errorf("%s: got %q, %v, want %q", c.name, got, err, c.want)
		}
	}
}

// seedPersonaPattern 初始化脚本中的一个风格: 代码、名称、说明、策略、参数
var seedPersonaPattern = regexp.MustCompile(`\('([a-z0-9_]+)', '([^']*)', '([^']*)', '([a-z0-9_]+)',\s*'(\{[^']*\})'\)`)

// TestSeedPersonas 初始化脚本中的风格和基础策略的默认参数都能通过保存风格时的校验
func TestSeedPersonas(t *testing.T) {
	raw, err := os.ReadFile("../../db/migrator/migrations/000001_init.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, section, _ := strings.Cut(string(raw), "INSERT INTO bot_personas")
	section, _, _ = strings.Cut(section, ";")
	var personas []bot.Persona
	for _, m := range seedPersonaPattern.FindAllStringSubmatch(section, -1) {
		params := map[string]any{}
		if err := json.Unmarshal([]byte(m[5]), &params); err != nil {
			t.Fatalf("%s: %v", m[1], err)
		}
		personas = append(personas, bot.Persona{Code: m[1], Name: m[2], Description: m[3], Strategy: m[4], Params: params})
	}
	if len(personas) == 0 {
		t.Fatal("no seed personas found")
	}
	for _, s := range bot.Strategies() {
		if s.Code == StrategyBasic {
			personas = append(personas, bot.Persona{Code: "defaults", Name: "默认参数", Strategy: s.Code, Params: s.Defaults})
		}
	}
	if personas[len(personas)-1].Code != "defaults" {
		t.Fatalf("strategy %s not registered", StrategyBasic)
	}
	for _, p := range personas {
		if err := bot.Validate(nil, &p); err != nil {
			t.Errorf("%s: %v", p.Code, err)
		}
	}
}
