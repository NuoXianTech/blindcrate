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

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"gorm.io/gorm"
)

// StrategyBasic 暗拍的基础策略, 见 decideBasic
const StrategyBasic = "auction_basic"

// 基础策略的估值方式, 越粗越容易看错
const (
	PerceptionSize    = "size"    // 按尺寸
	PerceptionArea    = "area"    // 按面积, 不分横竖和形状
	PerceptionQuality = "quality" // 不看尺寸, 扫出轮廓的也按全部藏品的平均价
)

// Observation 人机每轮决策时看到的信息, 与同座位的真人在这一轮出价时看到的相同:
// 地图的拍卖规则(含首轮扫描比例)和公示概率, 第 1 轮扫出轮廓的藏品占的格子, 自己的带入资金、出价和之前每轮的反馈;
// 不包含仓库价值、件数、品质、藏品名称和其他人的出价。策略不能修改观察。
type Observation struct {
	Round      int32             `json:"round"` // 第几轮, 大于规则的轮数时为加轮, 此时自己与其他可以出价的人并列最高
	Rules      venue.AuctionRule `json:"rules"`
	Grid       venue.Grid        `json:"grid"`
	Footprints []Footprint       `json:"footprints"` // 第 1 轮扫出轮廓的藏品占的格子
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
	Perception  string `json:"perception" label:"估值方式" options:"size:按尺寸,area:按面积,quality:不看尺寸" binding:"oneof=size area quality"`
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
		"按公开信息估值: 扫出轮廓的藏品按尺寸取地图公示概率下的平均回收价, 没扫出的按期望价值折算; 按轮次逐步加价, 看反馈追价或放弃",
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
//   - 估值: 按公开信息估算仓库价值(见 appraise), 再按种子偏高或偏低, 偏差在 ± Bias 内且整局不变; 公开信息开局后不再变化, 估值整局不变
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
	appraisal := appraise(obs.Odds, obs.Footprints, p.Perception, rules.PreviewRate)
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

// appraise 按公开信息估算仓库价值: 扫出轮廓的每件按估值方式归类, 取公示概率下同类藏品按期望件数加权的平均回收价;
// 没扫出的按期望价值 × (1 - 首轮扫描比例) 估算
//
// 每件藏品扫不扫出与它是什么无关, 没扫出的部分平均就是期望价值的 (1 - 比例); 不看尺寸或公示概率中没有同类时按全部藏品的平均价。
// 扫出的件数也透露了总件数, 这里不利用, 留给真人。previewRate 为基点。
func appraise(odds []venue.PieceOdds, footprints []Footprint, perception string, previewRate int32) float64 {
	type key struct{ width, height int16 }
	type group struct{ count, value float64 }
	keyOf := func(width, height int16) key {
		switch perception {
		case PerceptionArea:
			return key{width * height, 0}
		case PerceptionQuality:
			return key{}
		}
		return key{width, height}
	}
	groups := map[key]*group{}
	var all group
	for _, o := range odds {
		k := keyOf(o.Width, o.Height)
		g, ok := groups[k]
		if !ok {
			g = &group{}
			groups[k] = g
		}
		g.count += o.Count
		g.value += o.Count * o.Price
		all.count += o.Count
		all.value += o.Count * o.Price
	}
	if all.count <= 0 {
		return 0
	}
	total := all.value * float64(venue.BasisPoints-previewRate) / venue.BasisPoints
	for _, f := range footprints {
		if g, ok := groups[keyOf(f.Width, f.Height)]; ok && g.count > 0 {
			total += g.value / g.count
		} else {
			total += all.value / all.count
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
