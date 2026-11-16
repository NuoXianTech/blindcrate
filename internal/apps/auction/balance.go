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
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"gorm.io/gorm"
)

// 本文件是保存地图时的对局验算, 见 venue.RegisterBalancer
//
// 按推导出的拍卖参数模拟几种阵容的仓库: 人机按地图配置的风格出价, 与对局中相同; 真人按 humanModel 出价。
// 一个拍品的出价不受领先折让影响, 记下每个拍品第一名和第二名的出价, 再按不同的折让重算成交价(多拍品时见 simulate),
// 取参考阵容(4 个真人)的成交价 ÷ 货仓价值最接近目标的折让。入场费也不影响出价, 只平移盈亏,
// 按这个折让求出让参考阵容的真人盈亏等于目标的入场费, 见 solveEntryFee。

const (
	// balanceRuns 验算时每种阵容模拟的拍品数, 仓库数为它 ÷ 每局拍品数的期望
	balanceRuns = 4000
	// discountStep 领先折让的取值间隔, 基点
	discountStep = 100
	// balanceSalt 模拟对局的随机数的第二个种子
	balanceSalt = 0x5851f42d4c957f2d
)

// lineup 生成仓库的拍品, 即 *venue.Lineup(见 venueLineup), 测试时可以替换
type lineup interface {
	Count() int // 抽一个仓库的拍品数
	Pick() int  // 按权重抽一种档次
	Tiers() int // 档次的个数, 同 venue.Venue.Lots.Tiers
	Generate(tier int, valueFactor int32) (*venue.Depot, error)
	PieceOdds(tier int, valueFactor int32) []venue.PieceOdds
}

// venueLineup 把 *venue.Lineup 包成 lineup
type venueLineup struct{ l *venue.Lineup }

func (v venueLineup) Count() int { return v.l.Count() }
func (v venueLineup) Pick() int  { return v.l.Pick() }
func (v venueLineup) Tiers() int { return v.l.Tiers() }
func (v venueLineup) Generate(tier int, valueFactor int32) (*venue.Depot, error) {
	return v.l.Tier(tier).Generate(valueFactor)
}
func (v venueLineup) PieceOdds(tier int, valueFactor int32) []venue.PieceOdds {
	return v.l.Tier(tier).PieceOdds(valueFactor)
}

// humanModel 模拟的真人, 行为来自玩法改进的验算器, 参数见 settings.BalanceSettings 的模拟真人
//
//   - 估值: 知道货仓价值但看不准, 每轮的估值为价值 × (1 ± 误差 × [0.35, 1]), 每个拍品偏高或偏低;
//     误差从 SigmaStart 每轮减 SigmaStep × 5 ÷ 轮数(轮数越多每轮减得越少), 不低于 SigmaFloor
//   - 上限: min(估值 × CapFactor, 剩余资金, 预算), 低于起拍价或不高于当前出价时不再加价; 只有规划预算的真人有预算, 同人机的 budget
//   - 别人的公开出价不低于上限时放弃这个拍品
//   - 第 1 轮: JumpChance 的概率按估值的 95% 抢拍, 否则按起拍价试探
//   - 之后: 估值 × 本轮系数 × (1 ± 6%), 第 r 轮的系数为 Stair + 0.1 × (r - 1), 不超过 0.96;
//     别人的公开出价高过自己时追到它之上 5%, 并列加 4%
//   - 加轮: 在当前出价和上限之间加 30%-70%
type humanModel struct {
	SigmaStart float64
	SigmaStep  float64
	SigmaFloor float64
	JumpChance float64
	Stair      float64
	CapFactor  int64 // 基点
	PlanShare  int64 // 规划预算的比例, 基点
	BudgetPace int64 // 规划预算时的预算系数, 基点, 同人机的 BudgetPace
}

// newHumanModel 按设置生成模拟的真人
func newHumanModel(b *settings.BalanceSettings) humanModel {
	f := func(bp int) float64 { return float64(bp) / venue.BasisPoints }
	return humanModel{
		SigmaStart: f(b.HumanSigmaStart),
		SigmaStep:  f(b.HumanSigmaStep),
		SigmaFloor: f(b.HumanSigmaFloor),
		JumpChance: f(b.HumanJumpChance),
		Stair:      f(b.HumanStair),
		CapFactor:  int64(b.HumanCapFactor),
		PlanShare:  int64(b.HumanPlanShare),
		BudgetPace: int64(b.HumanBudgetPace),
	}
}

// humanView 模拟真人出价时看到的: 剩余资金、自己的出价、别人公开出价的最高价, 规划预算时还看后面拍品的期望
type humanView struct {
	carry    int64
	previous int64
	best     int64
	future   int64 // 后面拍品的期望之和, 不规划或最后一个拍品时为 -1
}

// bid 第 round 轮的出价, 0 表示不加价; sign 为 1 时这个拍品估值偏高, -1 时偏低
func (h *humanModel) bid(rule *venue.AuctionRule, round int32, value int64, sign float64, view humanView, rng *rand.Rand) int64 {
	step := h.SigmaStep * 5 / float64(max(rule.Rounds, 1))
	sigma := max(h.SigmaFloor, h.SigmaStart-step*float64(round-1))
	estimate := int64(math.Round(float64(value) * (1 + sign*sigma*(0.35+0.65*rng.Float64()))))
	carry := view.carry
	limit := min(scale(estimate, h.CapFactor), carry)
	if view.future >= 0 && estimate > 0 {
		limit = min(limit, budget(carry, estimate, view.future, h.BudgetPace))
	}
	previous := view.previous
	if limit < rule.StartPrice || limit <= previous || view.best >= limit {
		return 0
	}

	var target int64
	switch {
	case round > rule.Rounds:
		target = previous + scale(limit-previous, between(rng, 3000, 7000))
	case round == 1:
		if rng.Float64() < h.JumpChance {
			target = scale(estimate, 9500)
		}
	default:
		stair := min(0.96, h.Stair+0.1*float64(round-1))
		target = scale(estimate, int64(math.Round(stair*(1+(rng.Float64()*2-1)*0.06)*venue.BasisPoints)))
		switch {
		case view.best > previous:
			target = max(target, scale(view.best+1, venue.BasisPoints+500))
		case view.best > 0 && view.best == previous:
			target = max(target, scale(previous, 10400))
		}
	}
	target = min(max(target, rule.StartPrice), limit)
	if target > previous {
		return target
	}
	return 0
}

// squad 验算的一种阵容, 座位从 1 开始, 真人在前
type squad struct {
	code        string // 见 venue 的 Squad
	human       humanModel
	humans      int
	valueFactor int32                                // 货仓价值系数, 同 venue.Generator.Generate
	bots        []bot.Decider[Observation, Decision] // 出场的风格, 每局每个人机按 picks 的权重选一种
	picks       []int64
}

// seatSim 模拟仓库中一个座位在验算时另外记的, 资金和出价见 warehouse 的 seat
type seatSim struct {
	planner bool    // 真人是否规划预算
	sign    float64 // 真人对当前拍品的估值偏高(1)还是偏低(-1)
	profit  int64   // 仓库盈亏, 藏品按扣掉出售手续费后的回收价计, 不含入场费
}

// lotSim 一个拍品的模拟结果, 成交价由领先折让决定, 另算
type lotSim struct {
	value    int64
	topShare float64 // 最贵一件占货仓价值的比例
	round    int32   // 结束的轮次
	sold     bool
	speed    bool    // 最后一轮之前成交
	top      int64   // 第一名的出价
	second   int64   // 第二名的出价
	early    int64   // 最后一个公开轮次结束时的最高价, 没有公开轮次时为 0
	winner   int16   // 成交者的座位, 没有成交时为 0
	bidCV    float64 // 出过价的人的出价变异系数 ÷ √3(4 个人时的上限), 不到两人出过价时为 0
	appraise int64   // 人机最后的估值之和
}

// matchSim 一个仓库的模拟结果
type matchSim struct {
	lots    []lotSim
	profit  []int64 // 每个座位的仓库盈亏, 按模拟时的领先折让, 不含入场费
	carry   []int64 // 每个座位结束时的剩余资金
	planner []bool  // 每个座位是否是规划预算的真人
	hype    int32   // 结束时的热度, 见 hypeDelta
	hypes   int32   // 热度变动的次数, 即没有成交就结束的轮数
	hot     int32   // 其中变动后火热(不低于 hypeHot)的次数
	cold    int32   // 其中变动后冷清(不高于 hypeCold)的次数
}

// sold 是否至少成交了一个拍品, 是的话入场费不退
func (m *matchSim) sold() bool {
	return slices.ContainsFunc(m.lots, func(l lotSim) bool { return l.sold })
}

// balance 对局验算, 见 venue.Balancer: 4 个真人的参考阵容; 开放了排位时加上 2 个真人和 2 个人机, 开放了人机模式时加上 1 个真人和 3 个人机
func balance(tx *gorm.DB, v *venue.Venue, l *venue.Lineup, b *settings.BalanceSettings) (*venue.Balance, error) {
	economy, err := settings.Economy.Load(tx)
	if err != nil {
		return nil, err
	}
	human := newHumanModel(b)
	squads := []squad{{code: venue.SquadHumans, human: human, humans: playerCount, valueFactor: venue.BasisPoints}}
	modes := []struct {
		code        string
		enabled     bool
		humans      int
		valueFactor int32
		weights     []bot.Weight
	}{
		{venue.SquadRanked, v.RankedEnabled, rankedMinHumans, venue.BasisPoints, v.Bots.Ranked},
		{venue.SquadPve, v.PveEnabled, 1, v.Pve.ValueFactor, v.Bots.Pve},
	}
	for _, m := range modes {
		if !m.enabled {
			continue
		}
		candidates, picks, err := bot.Candidates(tx, m.weights)
		if err != nil {
			return nil, err
		}
		sq := squad{code: m.code, human: human, humans: m.humans, valueFactor: m.valueFactor, picks: picks}
		for i := range candidates {
			decide, err := bot.Prepare[Observation, Decision](&candidates[i])
			if err != nil {
				return nil, err
			}
			sq.bots = append(sq.bots, decide)
		}
		squads = append(squads, sq)
	}
	maxDiscount := int32(0)
	if b.LeadDiscountEnabled {
		maxDiscount = int32(b.MaxLeadDiscount)
	}
	// 真人盈亏的目标按一个仓库的期望总价值算, 一局一个拍品时就是一个货仓的期望价值
	seatEV := scale(v.WarehouseMean(), int64(b.TargetSeatEV))
	carryFloor := int64(venue.BasisPoints)
	if !v.Lots.Single() {
		carryFloor = int64(b.BotCarryFloor)
	}
	return simulate(v, func(seed uint64) lineup { return venueLineup{l.Seeded(seed)} }, squads, simConfig{
		sellFee: int32(economy.SellFeeRate), maxDiscount: maxDiscount, seatEV: seatEV, carryFloor: carryFloor, hypeDecay: int64(b.HypeLotDecay),
	})
}

// simConfig 验算的参数
type simConfig struct {
	sellFee     int32 // 出售手续费率, 基点
	maxDiscount int32 // 领先折让上限, 基点
	seatEV      int64 // 参考阵容中每个真人席位每局的盈亏目标
	carryFloor  int64 // 人机带入资金的下限, BotCarry 的基点
	hypeDecay   int64 // 每个拍品开场时热度向开局热度回落的比例, 基点
}

// simulate 模拟各阵容的仓库, 第一种为参考阵容; 在 [0, maxDiscount] 中求出领先折让写入 v.Auction.LeadDiscount,
// 再求出让参考阵容的真人盈亏等于 seatEV 的入场费写入 v.Auction.EntryFee, 返回各阵容按这个折让和入场费的结果
//
// 多拍品时成交价决定了后面拍品的剩余资金, 出价和折让有关: 先按折让上限的一半模拟一遍求出折让, 再按它重新模拟、重新求, 共三遍;
// 一局一个拍品时出价与折让无关, 每遍的结果相同。source 按种子生成拍品, 每种阵容用不同的种子; 相同的输入得到相同的结果。
func simulate(v *venue.Venue, source func(seed uint64) lineup, squads []squad, cfg simConfig) (*venue.Balance, error) {
	tiers := v.Lots.Tiers
	target := venue.BasisPoints - cfg.sellFee
	discount := cfg.maxDiscount / 2 / discountStep * discountStep
	var results [][]matchSim
	for range 3 {
		v.Auction.LeadDiscount = discount
		results = make([][]matchSim, len(squads))
		for i := range squads {
			seed := uint64(i + 1)
			matches, err := squads[i].run(v, tiers, source(seed), seed, cfg)
			if err != nil {
				return nil, err
			}
			results[i] = matches
		}
		discount = solveDiscount(results[0], v.Auction, target, cfg.maxDiscount)
	}

	v.Auction.LeadDiscount = discount
	v.Auction.EntryFee = solveEntryFee(&squads[0], results[0], v.Auction, cfg.sellFee, cfg.seatEV)
	b := &venue.Balance{Runs: len(results[0]), Target: target, Squads: make([]venue.SquadStats, len(squads))}
	for i := range squads {
		b.Squads[i] = squads[i].summarize(results[i], &v.Auction, cfg.sellFee)
	}
	return b, nil
}

// publicLot 一种格子大小的拍品按公开信息能推断出的: 格子大小相同的各档次按权重混合的公示概率和期望价值
type publicLot struct {
	grid     venue.Grid
	odds     []venue.PieceOdds
	expected int64
}

// publicLots 每个档次的拍品在出价时公开能看到的分布: 只知道格子大小, 档次按权重在格子大小相同的档次中推断
func publicLots(tiers []venue.Tier, source lineup, valueFactor int32) []publicLot {
	odds := make([][]venue.PieceOdds, len(tiers))
	for i := range tiers {
		odds[i] = source.PieceOdds(i, valueFactor)
	}
	lots := make([]publicLot, len(tiers))
	for i := range tiers {
		var weight float64
		for j := range tiers {
			if tiers[j].Depot.Grid == tiers[i].Depot.Grid {
				weight += float64(tiers[j].Weight)
			}
		}
		var mixed []venue.PieceOdds
		for j := range tiers {
			if tiers[j].Depot.Grid != tiers[i].Depot.Grid {
				continue
			}
			share := float64(tiers[j].Weight) / weight
			for _, o := range odds[j] {
				o.Count *= share
				mixed = append(mixed, o)
			}
		}
		lots[i] = publicLot{grid: tiers[i].Depot.Grid, odds: mixed, expected: expectedValue(mixed)}
	}
	return lots
}

// run 模拟 balanceRuns ÷ 每局拍品数的期望 个仓库, 使拍品总数约为 balanceRuns; 摆不下的仓库不计入
func (sq *squad) run(v *venue.Venue, tiers []venue.Tier, source lineup, seed uint64, cfg simConfig) ([]matchSim, error) {
	rng := rand.New(rand.NewPCG(seed, balanceSalt))
	public := publicLots(tiers, source, sq.valueFactor)
	warehouses := int(math.Ceil(balanceRuns / v.Lots.MeanCount()))
	matches := make([]matchSim, 0, warehouses)
	for range warehouses {
		count := source.Count()
		kinds := make([]int, count)
		depots := make([]*venue.Depot, count)
		fit := true
		for i := range kinds {
			kinds[i] = source.Pick()
			depot, err := source.Generate(kinds[i], sq.valueFactor)
			if err != nil {
				fit = false
				break
			}
			depots[i] = depot
		}
		if !fit {
			continue
		}
		m, err := sq.play(v, tiers, public, kinds, depots, rng, cfg)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// play 模拟一个仓库, 按对局的规则推进(见 warehouse): 每轮扫出的轮廓见 outlineRounds;
// 真人带满上限, 人机带 BotCarry × [carryFloor, 100%]; 成交价按 v.Auction 的领先折让从剩余资金扣除
func (sq *squad) play(v *venue.Venue, tiers []venue.Tier, public []publicLot, kinds []int, depots []*venue.Depot, rng *rand.Rand, cfg simConfig) (matchSim, error) {
	sims := make([]seatSim, playerCount)
	s := &warehouse{Seats: make([]seat, playerCount), Hosted: true, Hype: hypeStart, HypeDecay: cfg.hypeDecay}
	for i := range sims {
		p := &s.Seats[i]
		p.Seat = int16(i + 1)
		if i < sq.humans {
			p.Human, p.Carry = true, v.Auction.MaxCarry
			sims[i].planner = len(kinds) > 1 && rng.Int64N(venue.BasisPoints) < sq.human.PlanShare
			continue
		}
		p.Decide, p.Seed = sq.bots[loot.PickIndexWith(sq.picks, rng.Int64N)], rng.Int64()
		p.Carry = scale(v.Auction.BotCarry, between(rng, cfg.carryFloor, venue.BasisPoints))
	}

	var m matchSim
	for lot := range kinds {
		var future int64
		for _, k := range kinds[lot+1:] {
			future += public[k].expected
		}
		k := kinds[lot]
		spec := lotSpec{
			Rule:     v.Auction.ForTier(&tiers[k].Auction),
			Grid:     public[k].grid,
			Odds:     public[k].odds,
			Future:   future,
			LotsLeft: int32(len(kinds) - lot - 1),
		}
		l, err := sq.playLot(v, s, sims, spec, depots[lot], &m, rng)
		if err != nil {
			return matchSim{}, err
		}
		if l.sold {
			sims[l.winner-1].profit += scale(l.value, int64(venue.BasisPoints-cfg.sellFee)) - settlePrice(&spec.Rule, l.top, l.second)
		}
		m.lots = append(m.lots, l)
	}
	m.hype = s.Hype
	for i := range sims {
		m.profit = append(m.profit, sims[i].profit)
		m.carry = append(m.carry, s.Seats[i].Carry)
		m.planner = append(m.planner, sims[i].planner)
	}
	return m, nil
}

// playLot 模拟仓库中的一个拍品: 每轮真人按 humanModel 出价, 再结束这一轮; 热度的统计记在 m 上
func (sq *squad) playLot(v *venue.Venue, s *warehouse, sims []seatSim, spec lotSpec, depot *venue.Depot, m *matchSim, rng *rand.Rand) (lotSim, error) {
	rule := &spec.Rule
	pieces := depot.Pieces
	spec.Outlines = nil
	for i, round := range outlineRounds(len(pieces), rule, rng.Int64N) {
		if round > 0 {
			spec.Outlines = append(spec.Outlines, Footprint{X: pieces[i].X, Y: pieces[i].Y, Width: pieces[i].Width, Height: pieces[i].Height, Round: round})
		}
	}
	s.openLot(spec)
	for i := range sims {
		sims[i].sign = 1
		if s.Seats[i].Human && rng.IntN(2) == 0 {
			sims[i].sign = -1
		}
	}
	start := time.Unix(0, 0)
	s.Clock = func() time.Time {
		return start.Add(time.Duration(s.lotRound())*time.Minute + time.Duration(rng.Int64N(int64(time.Minute))))
	}

	l := lotSim{value: depot.Value}
	var result verdict
	for {
		round := s.lotRound()
		for i := range sims {
			p := &s.Seats[i]
			if !p.Human || !s.contender(p.Seat) {
				continue
			}
			view := humanView{carry: p.Carry, previous: p.Bid, best: slices.Max(append(s.others(i), 0)), future: -1}
			if sims[i].planner && spec.LotsLeft > 0 {
				view.future = spec.Future
			}
			if amount := sq.human.bid(rule, round, depot.Value, sims[i].sign, view, rng); amount > p.Bid {
				s.raise(i, amount, s.Clock())
			}
		}
		res := s.closeRound()
		if len(res.Failed) > 0 {
			return lotSim{}, res.Failed[0].Err
		}
		if result = res.Verdict; result.Outcome != "" {
			l.round = round
			break
		}
		m.hypes++
		if s.Hype >= hypeHot {
			m.hot++
		}
		if s.Hype <= hypeCold {
			m.cold++
		}
		// 前 N-1 轮结束后公开出价, 最后一轮和加轮暗标
		if round < rule.Rounds {
			l.early = 0
			for _, p := range s.Seats {
				l.early = max(l.early, p.shown)
			}
		}
	}

	if depot.Value > 0 {
		var top int64
		for _, piece := range pieces {
			if piece.SellCurrency == v.BidCurrency {
				top = max(top, piece.SellPrice)
			}
		}
		l.topShare = float64(top) / float64(depot.Value)
	}
	if result.Outcome == OutcomeSold {
		sorted := sortBidders(s.bidders())
		l.sold, l.speed, l.winner = true, l.round < rule.Rounds, result.Winner
		l.top, l.second = sorted[0].Bid, sorted[1].Bid
		if l.speed {
			l.early = l.top
		}
	}
	var positive []float64
	for _, p := range s.Seats {
		if p.Bid > 0 {
			positive = append(positive, float64(p.Bid))
		}
		if !p.Human {
			l.appraise += p.Estimate
		}
	}
	l.bidCV = variation(positive) / math.Sqrt(playerCount-1)
	return l, nil
}

// summarize 按 rule 的领先折让和入场费统计一种阵容的结果, 藏品按扣掉出售手续费后的回收价计
//
// 成交、价格、速胜、离散度、估值和胜率按拍品统计, 盈亏按仓库统计; 仓库盈亏按 rule 的折让重算, 与模拟时的折让不同时剩余资金不重算。
func (sq *squad) summarize(matches []matchSim, rule *venue.AuctionRule, sellFee int32) venue.SquadStats {
	stats := venue.SquadStats{Squad: sq.code, Humans: int32(sq.humans)}
	n := len(matches)
	if n == 0 {
		return stats
	}
	var (
		lots, sold, speed, wins, rounds int
		price, value, appraised         float64
		bidCV, topShare, early          float64
		seats, winners                  float64
		winnerSeats, rankWins, out      int
		hypes, hot, cold, hype          int
		plan, noPlan                    [2]float64 // 盈亏之和, 席位数
	)
	bots := playerCount - sq.humans
	var allValue float64
	for _, m := range matches {
		hypes, hot, cold, hype = hypes+int(m.hypes), hot+int(m.hot), cold+int(m.cold), hype+int(m.hype)
		profit := make([]int64, playerCount)
		for _, l := range m.lots {
			lots++
			rounds += int(l.round)
			bidCV += l.bidCV
			topShare += l.topShare
			appraised += float64(l.appraise)
			allValue += float64(l.value)
			if !l.sold {
				continue
			}
			sold++
			if l.speed {
				speed++
			}
			paid := settlePrice(rule, l.top, l.second)
			price += float64(paid)
			value += float64(l.value)
			if paid > 0 {
				early += float64(l.early) / float64(paid)
			}
			profit[l.winner-1] += scale(l.value, int64(venue.BasisPoints-sellFee)) - paid
			if int(l.winner) <= sq.humans {
				wins++
			}
		}

		// 模拟中的真人都是正式局, 每人都交了入场费, 全部流拍时退还
		fee := int64(0)
		if m.sold() {
			fee = rule.EntryFee
		}
		best := 0
		for i := range profit {
			if profit[i] > profit[best] {
				best = i
			}
		}
		for i := range sq.humans {
			result := profit[i] - fee
			seats += float64(result)
			if slices.ContainsFunc(m.lots, func(l lotSim) bool { return l.sold && int(l.winner) == i+1 }) {
				winners += float64(result)
				winnerSeats++
			}
			if i == best && profit[i] > 0 {
				rankWins++
			}
			if m.carry[i] < rule.StartPrice {
				out++
			}
			if len(m.lots) > 1 {
				if m.planner[i] {
					plan[0], plan[1] = plan[0]+float64(result), plan[1]+1
				} else {
					noPlan[0], noPlan[1] = noPlan[0]+float64(result), noPlan[1]+1
				}
			}
		}
	}

	bp := func(x float64) int32 { return int32(math.Round(x * venue.BasisPoints)) }
	stats.Lots = int32(math.Round(float64(lots) * 100 / float64(n)))
	stats.Sold = bp(float64(sold) / float64(lots))
	if value > 0 {
		stats.PriceValue = bp(price / value)
	}
	stats.SpeedWin = bp(float64(speed) / float64(lots))
	stats.BidCV = bp(bidCV / float64(lots))
	stats.TopShare = bp(topShare / float64(lots))
	if bots > 0 && allValue > 0 {
		stats.AppraiseBias = bp(appraised / (allValue * float64(bots)))
	}
	stats.AvgRounds = int32(math.Round(float64(rounds) * 100 / float64(lots)))
	if sold > 0 {
		stats.EarlyShare = bp(early / float64(sold))
	}
	if sq.humans > 0 {
		perSeat := float64(n * sq.humans)
		stats.WinRate = bp(float64(wins) / float64(lots*sq.humans))
		stats.SeatEV = int64(math.Round(seats / perSeat))
		if winnerSeats > 0 {
			stats.WinnerEV = int64(math.Round(winners / float64(winnerSeats)))
		}
		stats.RankWin = bp(float64(rankWins) / perSeat)
		stats.BudgetOut = bp(float64(out) / perSeat)
		if plan[1] > 0 && noPlan[1] > 0 {
			stats.PlanGain = int64(math.Round(plan[0]/plan[1] - noPlan[0]/noPlan[1]))
		}
	}
	if hypes > 0 {
		stats.HypeHot, stats.HypeCold = bp(float64(hot)/float64(hypes)), bp(float64(cold)/float64(hypes))
	}
	stats.HypeEnd = int32(math.Round(float64(hype) / float64(n)))
	return stats
}

// solveEntryFee 入场费: 让 sq 每个真人席位每局的平均盈亏等于 seatEV, 不低于 0; 成交价按 rule 的领先折让
//
// 入场费不影响出价, 只平移盈亏; 全部流拍时退还, 至少成交一个拍品的仓库才收, 所以按这种仓库的比例放大。不收入场费时盈亏已经不高于目标的为 0。
func solveEntryFee(sq *squad, matches []matchSim, rule venue.AuctionRule, sellFee int32, seatEV int64) int64 {
	rule.EntryFee = 0
	stats := sq.summarize(matches, &rule, sellFee)
	var charged int64
	for i := range matches {
		if matches[i].sold() {
			charged++
		}
	}
	if charged == 0 {
		return 0
	}
	return max((stats.SeatEV-seatEV)*int64(len(matches))/charged, 0)
}

// solveDiscount 领先折让: 在 [0, maxDiscount] 中按 discountStep 取值, 取全部拍品的成交价 ÷ 货仓价值最接近 target 的, 一样近时取小的
//
// 折让越大成交价越低; 不打折时已经不高于目标的为 0, 打到上限仍高于目标的为上限, 由闸门判定是否可用。
func solveDiscount(matches []matchSim, rule venue.AuctionRule, target, maxDiscount int32) int32 {
	best, gap := int32(0), math.Inf(1)
	for d := int32(0); d <= maxDiscount; d += discountStep {
		rule.LeadDiscount = d
		var price, value float64
		for _, m := range matches {
			for _, l := range m.lots {
				if l.sold {
					price += float64(settlePrice(&rule, l.top, l.second))
					value += float64(l.value)
				}
			}
		}
		if value == 0 {
			return 0
		}
		if g := math.Abs(price/value*venue.BasisPoints - float64(target)); g < gap {
			best, gap = d, g
		}
	}
	return best
}

// variation 总体变异系数, 标准差 ÷ 平均值; 不到两个数时为 0
func variation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	var squares float64
	for _, v := range values {
		squares += (v - mean) * (v - mean)
	}
	return math.Sqrt(squares/float64(len(values))) / mean
}
