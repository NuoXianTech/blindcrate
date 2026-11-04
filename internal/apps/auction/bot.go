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

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"gorm.io/gorm"
)

// StrategyBasic 暗拍的基础策略, 见 decideBasic
const StrategyBasic = "auction_basic"

// 基础策略的估值方式, 越粗越容易看错
const (
	PerceptionSize    = "size"    // 按尺寸和品质
	PerceptionArea    = "area"    // 按面积和品质, 不分横竖和形状
	PerceptionQuality = "quality" // 只按品质, 不看尺寸
)

// Observation 人机每轮决策时看到的信息, 与同座位的真人在这一轮出价时看到的相同:
// 地图的拍卖规则和公示概率, 藏品占的格子和已公开的品质, 自己的带入资金、出价和之前每轮的反馈;
// 不包含仓库价值、藏品名称和其他人的出价。策略不能修改观察。
type Observation struct {
	Round      int32             `json:"round"` // 第几轮, 大于规则的轮数时为加轮, 此时自己与其他可以出价的人并列最高
	Rules      venue.AuctionRule `json:"rules"`
	Grid       venue.Grid        `json:"grid"`
	Footprints []Footprint       `json:"footprints"` // 全部藏品占的格子, 已公开品质的带品质
	Odds       []venue.PieceOdds `json:"odds"`       // 按公示概率算出的藏品分布
	Carry      int64             `json:"carry"`      // 带入资金, 出价的上限
	Bid        int64             `json:"bid"`        // 自己的当前出价, 0 表示还没出过价
	Feedback   []Feedback        `json:"feedback"`   // 之前每轮的反馈
	Seed       int64             `json:"seed"`       // 这个人机在这一局的随机种子, 用于整局不变的个性, 如估值偏高还是偏低
}

// Decision 人机一轮的决策
type Decision struct {
	Bid      int64 `json:"bid"`      // 出价, 不高于当前出价或低于起拍价时为不加价, 超过带入资金时按带入资金
	Estimate int64 `json:"estimate"` // 对仓库价值的估值, 只用于后台复盘
}

// basicParams 基础策略的参数, 系数和概率都是基点
type basicParams struct {
	Perception  string `json:"perception" label:"估值方式" options:"size:按尺寸和品质,area:按面积和品质,quality:只按品质" binding:"oneof=size area quality"`
	Bias        int32  `json:"bias" label:"估值偏差" unit:"bp" help:"每局的估值在 ± 偏差内随机偏高或偏低, 整局不变" binding:"min=0,max=5000"`
	OpenFactor  int32  `json:"open_factor" label:"首轮系数" unit:"bp" help:"第 1 轮按估值 × 首轮系数出价, 之后逐轮增加, 最后一轮为末轮系数" binding:"min=100,max=15000"`
	CloseFactor int32  `json:"close_factor" label:"末轮系数" unit:"bp" binding:"min=100,max=15000"`
	MaxFactor   int32  `json:"max_factor" label:"上限系数" unit:"bp" help:"出价不超过估值 × 上限系数" binding:"min=100,max=15000"`
	Jitter      int32  `json:"jitter" label:"出价浮动" unit:"bp" help:"每轮按系数算出的出价在 ± 浮动内随机" binding:"min=0,max=5000"`
	JumpChance  int32  `json:"jump_chance" label:"抢拍概率" unit:"bp" help:"第 1 轮直接按估值 × 抢拍系数出价的概率, 用于争取速胜" binding:"min=0,max=10000"`
	JumpFactor  int32  `json:"jump_factor" label:"抢拍系数" unit:"bp" binding:"min=100,max=15000"`
	Chase       int32  `json:"chase" label:"追价幅度" unit:"bp" help:"提示落后较多且上限够得着时, 出价追到反超所需的最低价之上这么多; 够不着时放弃" binding:"min=0,max=10000"`
	Guard       bool   `json:"guard" label:"防速胜" help:"最后一轮之前, 出价不低于上限 ÷ 本轮速胜倍率, 对手要速胜只能出到上限以上"`
}

// Validate 系数需满足首轮 ≤ 末轮 ≤ 上限
func (p *basicParams) Validate(*gorm.DB) (string, error) {
	if p.OpenFactor > p.CloseFactor || p.CloseFactor > p.MaxFactor {
		return BotFactorsInvalid, nil
	}
	return "", nil
}

// registerStrategies 注册人机的决策策略, 在 init 中调用
func registerStrategies() {
	bot.RegisterStrategy(StrategyBasic, "暗拍基础策略",
		"按公开信息估值: 藏品按尺寸(已公开品质的再按品质)取地图公示概率下的平均回收价; 按轮次逐步加价, 看反馈追价或放弃",
		basicParams{
			Perception:  PerceptionSize,
			Bias:        1000,
			OpenFactor:  4200,
			CloseFactor: 7800,
			MaxFactor:   8800,
			Jitter:      300,
			JumpChance:  0,
			JumpFactor:  9000,
			Chase:       1500,
			Guard:       true,
		}, decideBasic)
}

// decideBasic 基础策略
//
//   - 估值: 按公开信息估算仓库价值, 再按种子偏高或偏低, 偏差在 ± Bias 内且整局不变; 压轴揭示后没公开的藏品按不比压轴的贵估算
//   - 上限: min(估值 × 上限系数, 带入资金), 低于起拍价或不高于当前出价时不再加价
//   - 抢拍: 第 1 轮有 JumpChance 的概率按估值 × 抢拍系数出价
//   - 按轮次: 估值 × 本轮系数 × (1 ± 浮动), 系数从首轮系数逐轮线性增加到末轮系数
//   - 看反馈: 落后较多时, 上限够得着就追到反超所需的最低价之上 Chase, 够不着就放弃;
//     小幅落后加价 10%-20%, 并列加价 3%-10%, 领先较多时只加一半
//   - 防速胜: 最后一轮之前, 出价不低于上限 ÷ 本轮速胜倍率
//   - 加轮: 在当前出价和上限之间加 30%-70%
func decideBasic(obs Observation, p *basicParams, rng *rand.Rand) Decision {
	rules := &obs.Rules
	personal := rand.New(rand.NewPCG(uint64(obs.Seed), 0))
	// 压轴揭示: 到压轴的一轮时最贵的一件已经公开, 没公开的都不比它贵
	topRevealed := rules.Finale && obs.Round >= min(rules.Rounds, int32(len(obs.Footprints)))
	appraisal := appraise(obs.Odds, obs.Footprints, p.Perception, topRevealed)
	estimate := int64(appraisal * float64(venue.BasisPoints+spread(personal, p.Bias)) / venue.BasisPoints)
	decision := Decision{Estimate: estimate}
	limit := min(scale(estimate, int64(p.MaxFactor)), obs.Carry)
	if limit < rules.StartPrice || limit <= obs.Bid {
		return decision
	}

	previous := obs.Bid
	var target int64
	switch {
	case obs.Round > rules.Rounds:
		target = previous + scale(limit-previous, between(rng, 3000, 7000))
	case obs.Round == 1 && rng.Int64N(venue.BasisPoints) < int64(p.JumpChance):
		target = scale(estimate, int64(p.JumpFactor))
	default:
		factor := p.CloseFactor
		if rules.Rounds > 1 {
			factor = p.OpenFactor + (p.CloseFactor-p.OpenFactor)*(obs.Round-1)/(rules.Rounds-1)
		}
		target = scale(scale(estimate, int64(factor)), venue.BasisPoints+spread(rng, p.Jitter))
		if previous > 0 && len(obs.Feedback) > 0 {
			switch obs.Feedback[len(obs.Feedback)-1].Hint {
			case HintFarBehind:
				// 其他人的最高价至少是自己的 hintFarRatio 倍
				need := ceilDiv(previous*hintFarRatio, venue.BasisPoints)
				if need > limit {
					return decision
				}
				target = max(target, scale(need, venue.BasisPoints+int64(p.Chase)))
			case HintClose:
				target = max(target, scale(previous, venue.BasisPoints+between(rng, 1000, 2000)))
			case HintTied:
				target = max(target, scale(previous, venue.BasisPoints+between(rng, 300, 1000)))
			case HintFarAhead:
				target = previous + (target-previous)/2
			}
		}
	}
	// 速胜倍率只有最后一轮之前的轮次有
	if p.Guard && obs.Round >= 1 && int(obs.Round) <= len(rules.QuickWin) {
		target = max(target, ceilDiv(limit*venue.BasisPoints, int64(rules.QuickWin[obs.Round-1])))
	}
	target = min(max(target, rules.StartPrice), limit)
	if target > previous {
		decision.Bid = target
	}
	return decision
}

// appraise 按公开信息估算仓库价值: 每件藏品按估值方式归类, 取公示概率下同类藏品按期望件数加权的平均回收价
//
// 已公开品质的按品质和尺寸(或面积)归类, 没公开的只按尺寸(或面积); 只按品质时, 没公开的按全部藏品的平均价。
// 公示概率中没有同类时按全部藏品的平均价。
//
// topRevealed 表示最贵的一件已经公开(压轴揭示): 已公开的估价最高的一件视为最贵的, 没公开的只取同类中平均回收价不超过它的,
// 没有时按它的估价; 否则最后一轮会把没公开的按包含最贵藏品的平均价估算, 系统性地估高。
func appraise(odds []venue.PieceOdds, footprints []Footprint, perception string, topRevealed bool) float64 {
	type key struct{ width, height, quality int16 }
	type group struct{ count, value float64 }
	keyOf := func(width, height, quality int16) key {
		switch perception {
		case PerceptionArea:
			return key{width * height, 0, quality}
		case PerceptionQuality:
			return key{0, 0, quality}
		}
		return key{width, height, quality}
	}
	groups := map[key]*group{}
	var all group
	for _, o := range odds {
		// 每种藏品同时计入带品质的类和不看品质的类, 品质从 1 开始, 0 表示没公开
		for _, k := range []key{keyOf(o.Width, o.Height, o.Quality), keyOf(o.Width, o.Height, 0)} {
			g, ok := groups[k]
			if !ok {
				g = &group{}
				groups[k] = g
			}
			g.count += o.Count
			g.value += o.Count * o.Price
		}
		all.count += o.Count
		all.value += o.Count * o.Price
	}
	if all.count <= 0 {
		return 0
	}
	price := func(f Footprint) float64 {
		if g, ok := groups[keyOf(f.Width, f.Height, f.Quality)]; ok && g.count > 0 {
			return g.value / g.count
		}
		return all.value / all.count
	}
	top := math.Inf(1)
	if topRevealed {
		top = 0
		for _, f := range footprints {
			if f.Quality > 0 {
				top = max(top, price(f))
			}
		}
	}
	var total float64
	for _, f := range footprints {
		if f.Quality > 0 || math.IsInf(top, 1) {
			total += price(f)
			continue
		}
		var below group
		for _, o := range odds {
			if keyOf(o.Width, o.Height, 0) == keyOf(f.Width, f.Height, 0) && o.Price <= top {
				below.count += o.Count
				below.value += o.Count * o.Price
			}
		}
		if below.count > 0 {
			total += below.value / below.count
		} else {
			total += min(price(f), top)
		}
	}
	return total
}

// scale v × 系数, 系数为基点
func scale(v, factor int64) int64 {
	return v * factor / venue.BasisPoints
}

// spread [-n, n] 内均匀随机
func spread(rng *rand.Rand, n int32) int64 {
	return rng.Int64N(2*int64(n)+1) - int64(n)
}

// between [low, high] 内均匀随机
func between(rng *rand.Rand, low, high int64) int64 {
	return low + rng.Int64N(high-low+1)
}

// ceilDiv a ÷ b 向上取整, a、b 都是正数
func ceilDiv(a, b int64) int64 {
	return (a + b - 1) / b
}
