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
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"gorm.io/gorm"
)

// 本文件是保存地图时的对局验算, 见 venue.RegisterBalancer
//
// 按推导出的拍卖参数模拟几种阵容的对局: 人机按地图配置的风格出价, 与对局中相同; 真人按 humanModel 出价。
// 模拟的出价不受领先折让影响, 所以只模拟一遍, 记下每局第一名和第二名的出价, 再按不同的折让重算成交价,
// 取参考阵容(4 个真人)的成交价 ÷ 仓库价值最接近目标的折让。

const (
	// balanceRuns 验算时每种阵容模拟的局数
	balanceRuns = 4000
	// discountStep 领先折让的取值间隔, 基点
	discountStep = 100
	// balanceSalt 模拟对局的随机数的第二个种子
	balanceSalt = 0x5851f42d4c957f2d
)

// warehouses 生成仓库, 即 *venue.Generator, 测试时可以替换
type warehouses interface {
	Generate(valueFactor int32) (*venue.Warehouse, error)
	PieceOdds(valueFactor int32) []venue.PieceOdds
}

// humanModel 模拟的真人, 行为来自玩法改进的验算器, 参数见 settings.BalanceSettings 的模拟真人
//
//   - 估值: 知道仓库价值但看不准, 每轮的估值为价值 × (1 ± 误差 × [0.35, 1]), 整局偏高或偏低;
//     误差从 SigmaStart 每轮减 SigmaStep × 5 ÷ 轮数(轮数越多每轮减得越少), 不低于 SigmaFloor
//   - 上限: min(估值 × CapFactor, 带入资金), 低于起拍价或不高于当前出价时不再加价
//   - 第 1 轮: JumpChance 的概率按估值的 95% 抢拍, 否则按起拍价试探
//   - 之后: 估值 × 本轮系数 × (1 ± 6%), 第 r 轮的系数为 Stair + 0.1 × (r - 1), 不超过 0.96;
//     落后较多且够得着时追到反超所需的最低价之上 5%, 小幅落后加 12%, 并列加 4%
//   - 加轮: 在当前出价和上限之间加 30%-70%
type humanModel struct {
	SigmaStart float64
	SigmaStep  float64
	SigmaFloor float64
	JumpChance float64
	Stair      float64
	CapFactor  int64 // 基点
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
	}
}

// bid 第 round 轮的出价, 0 表示不加价; sign 为 1 时整局估值偏高, -1 时偏低
func (h *humanModel) bid(rule *venue.AuctionRule, round int32, value int64, sign float64, carry, previous int64, feedback []Feedback, rng *rand.Rand) int64 {
	step := h.SigmaStep * 5 / float64(max(rule.Rounds, 1))
	sigma := max(h.SigmaFloor, h.SigmaStart-step*float64(round-1))
	estimate := int64(math.Round(float64(value) * (1 + sign*sigma*(0.35+0.65*rng.Float64()))))
	limit := min(scale(estimate, h.CapFactor), carry)
	if limit < rule.StartPrice || limit <= previous {
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
		if previous > 0 && len(feedback) > 0 {
			switch feedback[len(feedback)-1].Hint {
			case HintFarBehind:
				if need := ceilDiv(previous*hintFarRatio, venue.BasisPoints); need <= limit {
					target = max(target, scale(need, venue.BasisPoints+500))
				}
			case HintClose:
				target = max(target, scale(previous, 11200))
			case HintTied:
				target = max(target, scale(previous, 10400))
			}
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
	valueFactor int32                                // 仓库价值系数, 同 venue.Generator.Generate
	bots        []bot.Decider[Observation, Decision] // 出场的风格, 每局每个人机按 picks 的权重选一种
	picks       []int64
}

// seatSim 模拟对局中的一个座位
type seatSim struct {
	seat     int16
	human    bool
	sign     float64 // 真人整局估值偏高(1)还是偏低(-1)
	decide   bot.Decider[Observation, Decision]
	seed     int64
	bid      int64
	bidAt    time.Time
	feedback []Feedback
	estimate int64 // 人机最后的估值
}

// matchSim 一局模拟对局的结果, 成交价由领先折让决定, 另算
type matchSim struct {
	value    int64
	topShare float64 // 最贵一件占仓库价值的比例
	round    int32   // 结束的轮次
	sold     bool
	speed    bool    // 最后一轮之前成交
	top      int64   // 第一名的出价
	second   int64   // 第二名的出价
	winner   int16   // 成交者的座位, 没有成交时为 0
	eligible []int16 // 可以分奖池的座位, 按名次排好
	bidCV    float64 // 出过价的人的出价变异系数 ÷ √3(4 个人时的上限), 不到两人出过价时为 0
	appraise int64   // 人机最后的估值之和
}

// balance 对局验算, 见 venue.Balancer: 4 个真人的参考阵容; 开放了排位时加上 2 个真人和 2 个人机, 开放了人机模式时加上 1 个真人和 3 个人机
func balance(tx *gorm.DB, v *venue.Venue, g *venue.Generator, b *settings.BalanceSettings) (*venue.Balance, error) {
	economy, _, err := settings.Economy.Load(tx)
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
	return simulate(v, func(seed uint64) warehouses { return g.Seeded(seed) }, squads, int32(economy.SellFeeRate), maxDiscount)
}

// simulate 模拟各阵容的对局, 第一种为参考阵容; 在 [0, maxDiscount] 中求出领先折让写入 v.Auction.LeadDiscount, 返回各阵容按这个折让的结果
//
// source 按种子生成仓库, 每种阵容用不同的种子; 相同的输入得到相同的结果。
func simulate(v *venue.Venue, source func(seed uint64) warehouses, squads []squad, sellFee, maxDiscount int32) (*venue.Balance, error) {
	results := make([][]matchSim, len(squads))
	for i := range squads {
		seed := uint64(i + 1)
		matches, err := squads[i].run(v, source(seed), seed)
		if err != nil {
			return nil, err
		}
		results[i] = matches
	}

	target := venue.BasisPoints - sellFee
	v.Auction.LeadDiscount = solveDiscount(results[0], v.Auction, target, maxDiscount)
	b := &venue.Balance{Runs: balanceRuns, Target: target, Squads: make([]venue.SquadStats, len(squads))}
	for i := range squads {
		b.Squads[i] = squads[i].summarize(results[i], &v.Auction, sellFee)
	}
	return b, nil
}

// run 模拟 balanceRuns 局, 摆不下的局不计入
func (sq *squad) run(v *venue.Venue, source warehouses, seed uint64) ([]matchSim, error) {
	rng := rand.New(rand.NewPCG(seed, balanceSalt))
	odds := source.PieceOdds(sq.valueFactor)
	matches := make([]matchSim, 0, balanceRuns)
	for range balanceRuns {
		warehouse, err := source.Generate(sq.valueFactor)
		if err != nil {
			continue
		}
		m, err := sq.play(v, warehouse, odds, rng)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// play 模拟一局, 规则与对局中相同: 公开顺序见 revealOrders, 判定见 judge, 反馈见 feedbackFor; 真人带满上限, 人机带 BotCarry
func (sq *squad) play(v *venue.Venue, warehouse *venue.Warehouse, odds []venue.PieceOdds, rng *rand.Rand) (matchSim, error) {
	rule := &v.Auction
	pieces := warehouse.Pieces
	reveal := revealOrders(pieces, v.BidCurrency, rule.Rounds, rule.Finale, rng.Perm)
	prints := make([]Footprint, len(pieces))
	for i, piece := range pieces {
		prints[i] = Footprint{X: piece.X, Y: piece.Y, Width: piece.Width, Height: piece.Height}
	}

	seats := make([]seatSim, playerCount)
	for i := range seats {
		s := &seats[i]
		s.seat, s.feedback = int16(i+1), []Feedback{}
		if i < sq.humans {
			s.human, s.sign = true, 1
			if rng.IntN(2) == 0 {
				s.sign = -1
			}
			continue
		}
		s.decide, s.seed = sq.bots[pickIndex(sq.picks, rng)], rng.Int64()
	}

	var (
		start      = time.Unix(0, 0)
		bidders    = make([]bidder, len(seats))
		contenders []int16
		extra      int32
		result     verdict
		round      int32
	)
	for round = 1; ; round++ {
		// 出价的观察是这一轮的格子, 只在两轮之间改
		for i := range pieces {
			if reveal[i] == round {
				prints[i].Quality = pieces[i].Quality
			}
		}
		for i := range seats {
			s := &seats[i]
			if len(contenders) > 0 && !slices.Contains(contenders, s.seat) {
				continue
			}
			var amount int64
			if s.human {
				amount = sq.human.bid(rule, round, warehouse.Value, s.sign, rule.MaxCarry, s.bid, s.feedback, rng)
			} else {
				obs := Observation{
					Round:      round,
					Rules:      *rule,
					Grid:       v.Warehouse.Grid,
					Footprints: prints,
					Odds:       odds,
					Carry:      rule.BotCarry,
					Bid:        s.bid,
					Feedback:   s.feedback,
					Seed:       s.seed,
				}
				decision, err := s.decide(obs, rand.New(rand.NewPCG(uint64(s.seed), uint64(round))))
				if err != nil {
					return matchSim{}, err
				}
				s.estimate = decision.Estimate
				if amount = min(decision.Bid, rule.BotCarry); amount < rule.StartPrice {
					amount = 0
				}
			}
			if amount > s.bid {
				s.bid, s.bidAt = amount, start.Add(time.Duration(round)*time.Minute+time.Duration(rng.Int64N(int64(time.Minute))))
			}
		}

		for i, s := range seats {
			bidders[i] = bidder{Seat: s.seat, Bid: s.bid, BidAt: s.bidAt}
		}
		if result = judge(rule, round, extra, bidders); result.Outcome != "" {
			break
		}
		bids := make(map[int16]int64, len(seats))
		for _, s := range seats {
			bids[s.seat] = s.bid
		}
		for i := range seats {
			seats[i].feedback = append(seats[i].feedback, feedbackFor(round, seats[i].seat, bids))
		}
		if round >= rule.Rounds {
			extra++
		}
		contenders = result.Contenders
	}

	m := matchSim{value: warehouse.Value, round: round}
	if warehouse.Value > 0 {
		var top int64
		for _, piece := range pieces {
			if piece.SellCurrency == v.BidCurrency {
				top = max(top, piece.SellPrice)
			}
		}
		m.topShare = float64(top) / float64(warehouse.Value)
	}
	if result.Outcome == OutcomeSold {
		sorted := sortBidders(bidders)
		m.sold, m.speed, m.winner = true, round < rule.Rounds, result.Winner
		m.top, m.second = sorted[0].Bid, sorted[1].Bid
		ranks := rankSeats(bidders, result.Winner, result.Price, warehouse.Value)
		for _, s := range seats {
			if s.human && s.seat != result.Winner && s.bid > 0 {
				m.eligible = append(m.eligible, s.seat)
			}
		}
		slices.SortFunc(m.eligible, func(x, y int16) int { return cmp.Compare(ranks[x], ranks[y]) })
	}
	var positive []float64
	for _, s := range seats {
		if s.bid > 0 {
			positive = append(positive, float64(s.bid))
		}
		if !s.human {
			m.appraise += s.estimate
		}
	}
	m.bidCV = variation(positive) / math.Sqrt(playerCount-1)
	return m, nil
}

// summarize 按 rule 的领先折让统计一种阵容的结果, 藏品按扣掉出售手续费后的回收价计
func (sq *squad) summarize(matches []matchSim, rule *venue.AuctionRule, sellFee int32) venue.SquadStats {
	stats := venue.SquadStats{Squad: sq.code, Humans: int32(sq.humans)}
	n := len(matches)
	if n == 0 {
		return stats
	}
	var (
		sold, speed, wins, rounds int
		price, value, appraised   float64
		bidCV, topShare           float64
		seats, winners, losers    float64
		shares                    float64
	)
	bots := playerCount - sq.humans
	for _, m := range matches {
		rounds += int(m.round)
		bidCV += m.bidCV
		topShare += m.topShare
		appraised += float64(m.appraise)
		if !m.sold {
			continue
		}
		sold++
		if m.speed {
			speed++
		}
		paid := settlePrice(rule, m.top, m.second)
		price += float64(paid)
		value += float64(m.value)

		// 与 state.splitPool 相同, 模拟中的真人都是正式局
		var overpay int64
		humanWon := int(m.winner) <= sq.humans
		if humanWon {
			overpay = max(paid-m.value, 0)
		}
		_, _, split := splitPool(rule, rule.EntryFee*int64(sq.humans), overpay, m.eligible)
		for seat := int16(1); int(seat) <= sq.humans; seat++ {
			var result int64
			if seat == m.winner {
				result = scale(m.value, int64(venue.BasisPoints-sellFee)) - paid - rule.EntryFee
				winners += float64(result)
				wins++
			} else {
				result = split[seat] - rule.EntryFee
				losers += float64(result)
				shares += float64(split[seat])
			}
			seats += float64(result)
		}
	}

	var allValue float64
	for _, m := range matches {
		allValue += float64(m.value)
	}
	bp := func(x float64) int32 { return int32(math.Round(x * venue.BasisPoints)) }
	stats.Sold = bp(float64(sold) / float64(n))
	if value > 0 {
		stats.PriceValue = bp(price / value)
	}
	stats.SpeedWin = bp(float64(speed) / float64(n))
	stats.BidCV = bp(bidCV / float64(n))
	stats.TopShare = bp(topShare / float64(n))
	if bots > 0 && allValue > 0 {
		stats.AppraiseBias = bp(appraised / (allValue * float64(bots)))
	}
	stats.AvgRounds = int32(math.Round(float64(rounds) * 100 / float64(n)))
	if sq.humans > 0 {
		perSeat := float64(n * sq.humans)
		stats.WinRate = bp(float64(wins) / perSeat)
		stats.SeatEV = int64(math.Round(seats / perSeat))
		if wins > 0 {
			stats.WinnerEV = int64(math.Round(winners / float64(wins)))
		}
		if lost := perSeat - float64(wins); lost > 0 {
			stats.LoserEV = int64(math.Round(losers / lost))
		}
	}
	stats.Share = int64(math.Round(shares / float64(n)))
	return stats
}

// solveDiscount 领先折让: 在 [0, maxDiscount] 中按 discountStep 取值, 取成交价 ÷ 仓库价值最接近 target 的, 一样近时取小的
//
// 折让越大成交价越低; 不打折时已经不高于目标的为 0, 打到上限仍高于目标的为上限, 由闸门判定是否可用。
func solveDiscount(matches []matchSim, rule venue.AuctionRule, target, maxDiscount int32) int32 {
	best, gap := int32(0), math.Inf(1)
	for d := int32(0); d <= maxDiscount; d += discountStep {
		rule.LeadDiscount = d
		var price, value float64
		for _, m := range matches {
			if m.sold {
				price += float64(settlePrice(&rule, m.top, m.second))
				value += float64(m.value)
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

// pickIndex 按权重随机选一个下标, 同 loot.PickIndex, 随机数从 rng 取
func pickIndex(weights []int64, rng *rand.Rand) int {
	var total int64
	for _, w := range weights {
		total += w
	}
	n := rng.Int64N(total)
	for i, w := range weights {
		if n < w {
			return i
		}
		n -= w
	}
	return len(weights) - 1
}
