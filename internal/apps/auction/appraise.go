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
	"slices"

	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
)

// 本文件是人机按公开信息对货仓的估值, 见 appraise
//
// 把公示分布看成: 每种藏品(品质和尺寸相同的算一种)的件数相互独立, 服从以期望件数为均值的泊松分布, 每件扫不扫出轮廓与它是什么无关。
// 这样没扫出的各种藏品仍然相互独立, 扫出的每件是哪一种只看同类藏品的期望件数之比; 每条提示按"说对的可能性为信任度, 说错时只差一档"
// 修正这两样, 不用枚举货仓里的组合。实际的货仓不完全是这样: 各档位的抽取次数有上下限, 数量波动让各种藏品一起变多变少;
// 所以件数推断和信任度留给风格去调, 这里只求方向对、平均不偏。

// negligible 按当前的推断, 一条提示出现的可能性(或它数的藏品的期望件数)小于它时当作不可能发生, 不理会这条提示;
// 也挡住浮点误差凑出来的可能性
const negligible = 1e-9

// belief 对一个货仓里有什么的推断
type belief struct {
	kinds  []venue.PieceOdds // 公示分布中期望件数大于 0 的藏品
	levels []int16           // 可能出现的品质, 从低到高
	hidden []float64         // 每种藏品还没扫出的期望件数, 同 kinds 的顺序
	prints []Footprint       // 扫出轮廓的藏品
	shares []float64         // 扫出轮廓的每件是各种藏品的概率, 每件占 len(kinds) 个, 和为 1; 见 row
}

// sizeClass 估值方式下藏品按尺寸归的类: 按尺寸时为宽和高, 按面积时为面积, 不看尺寸时都是同一类
type sizeClass struct{ width, height int16 }

func classOf(perception string, width, height int16) sizeClass {
	switch perception {
	case PerceptionArea:
		return sizeClass{width * height, 0}
	case PerceptionQuality:
		return sizeClass{}
	}
	return sizeClass{width, height}
}

// newBelief 只看公示分布和轮廓的推断, 公示分布为空时返回 nil
//
// 没扫出的每种藏品按期望件数 × hidden(每件到这一轮还没扫出的概率, 见 hiddenShare)计。inference 为件数推断(0-1), 见 basicParams:
// 扫出的件数 ÷ 扫出的概率是对总件数的估计, 按 inference 与期望件数加权; 平均下来仍是期望件数, 不会让估值整体偏高或偏低。
// 扫出的每件按估值方式归类, 是同类中各种藏品的概率与期望件数成正比; 不看尺寸或公示分布中没有同类时按全部藏品算。
func newBelief(odds []venue.PieceOdds, footprints []Footprint, hidden float64, perception string, inference float64) *belief {
	b := &belief{prints: footprints}
	var total float64
	for _, o := range odds {
		if o.Count > 0 {
			b.kinds = append(b.kinds, o)
			b.levels = append(b.levels, o.Quality)
			total += o.Count
		}
	}
	if total <= 0 {
		return nil
	}
	slices.Sort(b.levels)
	b.levels = slices.Compact(b.levels)

	scale := hidden
	if seen := 1 - hidden; inference > 0 && seen > 0 {
		scale *= 1 - inference + inference*float64(len(footprints))/(total*seen)
	}
	b.hidden = make([]float64, len(b.kinds))
	for i, k := range b.kinds {
		b.hidden[i] = k.Count * scale
	}
	b.shares = make([]float64, len(footprints)*len(b.kinds))
	for f, print := range footprints {
		row, class := b.row(f), classOf(perception, print.Width, print.Height)
		var sum float64
		for i, k := range b.kinds {
			if classOf(perception, k.Width, k.Height) == class {
				row[i] = k.Count
				sum += k.Count
			}
		}
		if sum <= 0 {
			for i, k := range b.kinds {
				row[i] = k.Count
			}
			sum = total
		}
		for i := range row {
			row[i] /= sum
		}
	}
	return b
}

// row 扫出轮廓的第 f 件是各种藏品的概率, 同 kinds 的顺序
func (b *belief) row(f int) []float64 {
	n := len(b.kinds)
	return b.shares[f*n : (f+1)*n]
}

// value 货仓价值的期望和方差: 没扫出的每种藏品的件数按泊松分布, 扫出的每件按它是各种藏品的概率
func (b *belief) value() (mean, variance float64) {
	for i, k := range b.kinds {
		mean += b.hidden[i] * k.Price
		variance += b.hidden[i] * k.Price * k.Price
	}
	for f := range b.prints {
		var m, square float64
		for i, share := range b.row(f) {
			m += share * b.kinds[i].Price
			square += share * b.kinds[i].Price * b.kinds[i].Price
		}
		mean += m
		variance += square - m*m
	}
	return mean, max(variance, 0)
}

// nearby 品质 quality 的下一档和上一档, 没有时为 0
func (b *belief) nearby(quality int16) (lower, upper int16) {
	i, found := slices.BinarySearch(b.levels, quality)
	if i > 0 {
		lower = b.levels[i-1]
	}
	if found {
		i++
	}
	if i < len(b.levels) {
		upper = b.levels[i]
	}
	return lower, upper
}

// told 主持人说品质是 said 时, 实际是 actual 的可能性: 说对的可能性为 trust; 说错时只会说成相邻的品质(见 shiftHint),
// 即实际是 said 的下一档 lower 或上一档 upper, 各占一半
func told(actual, said, lower, upper int16, trust float64) float64 {
	switch actual {
	case said:
		return trust
	case lower, upper:
		return (1 - trust) / 2
	}
	return 0
}

// piece 提示: 位置为 (x, y) 的藏品是 quality。只改这一件是各种藏品的概率; 这件既不可能是这个品质, 也不可能被说错成这个品质时不理会
func (b *belief) piece(x, y, quality int16, trust float64) {
	f := slices.IndexFunc(b.prints, func(p Footprint) bool { return p.X == x && p.Y == y })
	if f < 0 {
		return
	}
	row := b.row(f)
	lower, upper := b.nearby(quality)
	var sum float64
	for i, share := range row {
		sum += share * told(b.kinds[i].Quality, quality, lower, upper, trust)
	}
	if sum < negligible {
		return
	}
	for i := range row {
		row[i] *= told(b.kinds[i].Quality, quality, lower, upper, trust) / sum
	}
}

// count 提示: 品质不低于 quality 的藏品共有 n 件。把这些藏品的总件数近似成泊松分布, 说对时就是 n 件, 说错时差一件(见 shiftHint),
// 得出总件数的期望并按它调整(见 resize); 返回这个期望, 按当前的推断货仓里不可能有这些藏品时不理会, 返回 false
func (b *belief) count(quality int16, n int32, trust float64) (float64, bool) {
	rate, seen := b.group(quality)
	prior := rate + seen
	if prior < negligible {
		return 0, false
	}
	// 实际有 c 件的可能性正比于 prior^c ÷ c!, 都除以 n 件的: 多一件的是 prior ÷ (n + 1), 少一件的是 n ÷ prior
	exact, more, fewer := trust, (1-trust)/2*prior/float64(n+1), (1-trust)/2*float64(n)/prior
	total := (exact*float64(n) + more*float64(n+1) + fewer*float64(n-1)) / (exact + more + fewer)
	b.resize(quality, total)
	return total, true
}

// group 品质不低于 quality 的藏品没扫出的和已扫出的期望件数
func (b *belief) group(quality int16) (rate, seen float64) {
	for i, k := range b.kinds {
		if k.Quality >= quality {
			rate += b.hidden[i]
		}
	}
	for f := range b.prints {
		seen += b.atLeast(f, quality)
	}
	return rate, seen
}

// resize 把品质不低于 quality 的藏品的期望件数调整为 total: 扫出的每件是这些藏品的概率按 total 与原来之比修正,
// 其余的件数算在没扫出的里面, 各种藏品之间的比例不变
func (b *belief) resize(quality int16, total float64) {
	rate, seen := b.group(quality)
	if rate+seen < negligible {
		return
	}
	factor := total / (rate + seen)
	var left float64 // 修正后已扫出的期望件数
	for f := range b.prints {
		// 这件肯定是或肯定不是这些藏品时不用改, 与提示矛盾时(全信"0 件", 却扫出一件只可能是它的)也不改
		share := b.atLeast(f, quality)
		if share < negligible || share > 1-negligible {
			left += share
			continue
		}
		scaled := share * factor / (share*factor + 1 - share)
		row := b.row(f)
		for i, k := range b.kinds {
			if k.Quality >= quality {
				row[i] *= scaled / share
			} else {
				row[i] *= (1 - scaled) / (1 - share)
			}
		}
		left += scaled
	}
	if rate < negligible {
		return
	}
	rest := max(total-left, 0) / rate
	for i, k := range b.kinds {
		if k.Quality >= quality {
			b.hidden[i] *= rest
		}
	}
}

// atLeast 扫出轮廓的第 f 件的品质不低于 quality 的概率
func (b *belief) atLeast(f int, quality int16) float64 {
	var p float64
	for i, share := range b.row(f) {
		if b.kinds[i].Quality >= quality {
			p += share
		}
	}
	return min(p, 1)
}

// below 扫出轮廓的第 f 件的品质不高于 level 的概率
func (b *belief) below(f int, level int16) float64 {
	var p float64
	for i, share := range b.row(f) {
		if b.kinds[i].Quality <= level {
			p += share
		}
	}
	return min(p, 1)
}

// within 货仓里所有藏品的品质都不高于 level 的概率; level 低于所有品质时为货仓是空的概率
func (b *belief) within(level int16) float64 {
	var rate float64
	for i, k := range b.kinds {
		if k.Quality > level {
			rate += b.hidden[i]
		}
	}
	p := math.Exp(-rate)
	for f := range b.prints {
		p *= b.below(f, level)
	}
	return p
}

// best 提示: 货仓里最好的藏品是 quality。"最好的是某一档"等于"都不高于这一档"去掉"都不高于它的下一档";
// 按说对和说错成相邻品质的可能性, 把这几种情况下各种藏品的期望件数和每件的概率加权起来。
// 这之后各种藏品不再相互独立(至少有一件是最好的那一档), 只保留各自的期望, 所以放在指认和件数之后, 见 appraise
func (b *belief) best(quality int16, trust float64) {
	// scene 最好的品质是 top 的情况: weight 为这时主持人说成 quality 的可能性, all 和 less 为所有藏品都不高于 top、都不高于 top 的下一档 floor 的概率
	type scene struct {
		top, floor        int16
		weight, all, less float64
	}
	lower, upper := b.nearby(quality)
	scenes := make([]scene, 0, 3)
	var total float64
	for _, top := range [...]int16{lower, quality, upper} {
		if !slices.Contains(b.levels, top) {
			continue
		}
		s := scene{top: top, weight: told(top, quality, lower, upper, trust)}
		s.floor, _ = b.nearby(top)
		s.all, s.less = b.within(top), b.within(s.floor)
		total += s.weight * (s.all - s.less)
		scenes = append(scenes, s)
	}
	if total < negligible {
		return
	}

	for i, k := range b.kinds {
		var factor float64
		for _, s := range scenes {
			if k.Quality <= s.top {
				factor += s.weight * s.all
			}
			if k.Quality <= s.floor {
				factor -= s.weight * s.less
			}
		}
		b.hidden[i] *= max(factor, 0) / total
	}
	for f := range b.prints {
		// 除了这一件, 其余的都不高于 top、都不高于 floor 的概率
		var others [3][2]float64
		for j, s := range scenes {
			if p := b.below(f, s.top); p > 0 {
				others[j][0] = s.all / p
			}
			if p := b.below(f, s.floor); p > 0 {
				others[j][1] = s.less / p
			}
		}
		row := b.row(f)
		for i, k := range b.kinds {
			var factor float64
			for j, s := range scenes {
				if k.Quality <= s.top {
					factor += s.weight * others[j][0]
				}
				if k.Quality <= s.floor {
					factor -= s.weight * others[j][1]
				}
			}
			row[i] *= max(factor, 0) / total
		}
	}
}

// lean 提示: 货仓价值高于(above 次)或不高于(below 次)期望价值 expected。把货仓价值近似成期望为 mean、方差为 variance 的对数正态分布,
// 返回修正后的期望; 两种说法一样多或估值没有不确定性时不变
func lean(mean, variance, expected float64, above, below int, trust float64) float64 {
	if above == below || mean <= 0 || variance <= 0 || expected <= 0 {
		return mean
	}
	spread := math.Sqrt(math.Log1p(variance / (mean * mean)))
	if !(spread > 0) || math.IsInf(spread, 1) {
		return mean
	}
	at := (math.Log(mean/expected) - spread*spread/2) / spread
	// 价值高于期望的概率, 以及这种情况在估值中占的份额
	over, share := normal(at), normal(at+spread)
	// 这些提示在价值高于、不高于期望时出现的可能性
	high := math.Pow(trust, float64(above)) * math.Pow(1-trust, float64(below))
	low := math.Pow(trust, float64(below)) * math.Pow(1-trust, float64(above))
	chance := high*over + low*(1-over)
	if chance <= 0 {
		return mean
	}
	return mean * (high*share + low*(1-share)) / chance
}

// normal 标准正态分布不超过 x 的概率
func normal(x float64) float64 {
	return math.Erfc(-x/math.Sqrt2) / 2
}

// appraise 按公开信息估算货仓价值: 没扫出的按期望价值 × 还没扫出的概率和扫出的件数折算, 扫出轮廓的每件按估值方式归类,
// 取公示概率下同类藏品按期望件数加权的平均回收价(见 newBelief), 再按主持人的提示修正; 信任度为 0 时不看提示
//
// 提示按种类依次修正, 与给出的先后无关(件数、最好的品质和价值高低每个拍品各只说一次, 见 makeHint): 先是指认藏品的, 再是件数,
// 然后是最好的品质——它让"至少有一件"的那一档变多, 件数的提示已经算过这一档有几件, 所以之后把件数调回去, 只留下它对各档之间比例的修正,
// 不然同样的藏品会算两次, 估值偏高; 最后是价值高低, 它只在前面得出的估值上再偏一些, 前面的提示已经说明货仓值钱时几乎不改变估值。
func appraise(odds []venue.PieceOdds, footprints []Footprint, hints []HintView, hidden float64, p *basicParams) float64 {
	b := newBelief(odds, footprints, hidden, p.Perception, float64(p.CountInference)/venue.BasisPoints)
	if b == nil {
		return 0
	}
	trust := float64(p.HintTrust) / venue.BasisPoints
	if trust <= 0 {
		mean, _ := b.value()
		return mean
	}
	of := func(kind string, apply func(h *HintView)) {
		for i := range hints {
			if hints[i].Kind == kind {
				apply(&hints[i])
			}
		}
	}
	// counted 件数的提示得出的期望件数
	type counted struct {
		quality int16
		total   float64
	}
	var counts []counted
	of(HostHintPiece, func(h *HintView) { b.piece(h.X, h.Y, h.Quality, trust) })
	of(HostHintCount, func(h *HintView) {
		if total, ok := b.count(h.Quality, h.Count, trust); ok {
			counts = append(counts, counted{h.Quality, total})
		}
	})
	of(HostHintBest, func(h *HintView) { b.best(h.Quality, trust) })
	for _, c := range counts {
		b.resize(c.quality, c.total)
	}
	var above, below int
	of(HostHintValue, func(h *HintView) {
		if h.Above {
			above++
		} else {
			below++
		}
	})
	mean, variance := b.value()
	return lean(mean, variance, float64(expectedValue(odds)), above, below, trust)
}
