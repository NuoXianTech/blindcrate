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
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
)

// testOdds 四种藏品, 期望价值 2900, 全部藏品的平均价为 2900 / 6.5
var testOdds = []venue.PieceOdds{
	{Quality: 1, Width: 1, Height: 1, Count: 4, Price: 50},
	{Quality: 1, Width: 2, Height: 1, Count: 1, Price: 200},
	{Quality: 3, Width: 1, Height: 1, Count: 1, Price: 1000},
	{Quality: 3, Width: 1, Height: 2, Count: 0.5, Price: 3000},
}

// pairOdds 两种 1×1 的藏品, 期望价值 1200: 品质 1 的期望 4 件、每件 50, 品质 3 的期望 1 件、每件 1000;
// 扫出轮廓的一件有 0.8 的可能是品质 1, 估 240
var pairOdds = []venue.PieceOdds{
	{Quality: 1, Width: 1, Height: 1, Count: 4, Price: 50},
	{Quality: 3, Width: 1, Height: 1, Count: 1, Price: 1000},
}

// squares n 个 1×1 的轮廓, 位置为 (0, 0)、(1, 0)……
func squares(n int) []Footprint {
	prints := make([]Footprint, n)
	for i := range prints {
		prints[i] = Footprint{X: int16(i), Width: 1, Height: 1}
	}
	return prints
}

// trusting 按尺寸估值、不推断件数、提示信任度为 trust(基点)的参数
func trusting(trust int32) *basicParams {
	return &basicParams{Perception: PerceptionSize, HintTrust: trust}
}

func near(got, want float64) bool {
	return math.Abs(got-want) <= 1e-6*max(1, math.Abs(want))
}

// TestAppraise 不推断件数、不看提示时: 扫出的每件按同类藏品的平均价, 没扫出的按期望价值折算
func TestAppraise(t *testing.T) {
	average := 2900 / 6.5
	cases := []struct {
		name       string
		perception string
		footprints []Footprint
		hidden     float64 // 每件还没扫出的概率
		want       float64
	}{
		{"size", PerceptionSize, []Footprint{{Width: 1, Height: 1}}, 0, 240},
		{"size by shape", PerceptionSize, []Footprint{{Width: 2, Height: 1}, {Width: 1, Height: 2}}, 0, 3200},
		{"size unknown shape", PerceptionSize, []Footprint{{Width: 3, Height: 3}}, 0, average},
		{"area merges shapes", PerceptionArea, []Footprint{{Width: 1, Height: 2}}, 0, 1700 / 1.5},
		{"quality ignores size", PerceptionQuality, []Footprint{{Width: 1, Height: 2}}, 0, average},
		{"nothing previewed", PerceptionSize, nil, 1, 2900},
		{"unseen at the expected value", PerceptionSize, []Footprint{{Width: 1, Height: 1}}, 0.7, 240 + 2900*0.7},
	}
	for _, c := range cases {
		if got := appraise(testOdds, c.footprints, nil, c.hidden, &basicParams{Perception: c.perception}); !near(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	if got := appraise(nil, []Footprint{{Width: 1, Height: 1}}, nil, 0.7, trusting(8000)); got != 0 {
		t.Errorf("no odds: got %v, want 0", got)
	}
}

// TestAppraiseCountInference 扫出的件数比预期多时没扫出的部分估得多, 少时估得少; 每件扫出的概率 0.3 时预期扫出 6.5 × 0.3 件
func TestAppraiseCountInference(t *testing.T) {
	const expected, seen = 2900.0, 6.5 * 0.3
	average := expected / 6.5
	// 不看尺寸: 扫出的每件按平均价, 件数只影响没扫出的部分
	value := func(inference int32, n int, hidden float64) float64 {
		return appraise(testOdds, squares(n), nil, hidden, &basicParams{Perception: PerceptionQuality, CountInference: inference})
	}
	cases := []struct {
		name      string
		inference int32
		n         int
		want      float64
	}{
		{"不推断", 0, 3, 3*average + expected*0.7},
		{"扫出的多", 5000, 3, 3*average + expected*0.7*(0.5+0.5*3/seen)},
		{"扫出的少", 5000, 1, average + expected*0.7*(0.5+0.5*1/seen)},
		{"只按扫出的件数", 10000, 3, 3*average + expected*0.7*3/seen},
		{"一件也没扫出", 10000, 0, 0},
	}
	for _, c := range cases {
		if got := value(c.inference, c.n, 0.7); !near(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	// 全部扫出或都还没扫的时候没有可推断的
	if got := value(5000, 3, 0); !near(got, 3*average) {
		t.Errorf("all outlined: got %v, want %v", got, 3*average)
	}
	if got := value(5000, 0, 1); !near(got, expected) {
		t.Errorf("nothing scanned: got %v, want %v", got, expected)
	}

	// 件数服从泊松分布时, 不管推断多少, 估值的平均值都是期望价值
	for _, inference := range []int32{0, 3000, 10000} {
		var mean float64
		chance := math.Exp(-seen) // 扫出 n 件的概率
		for n := range 60 {
			mean += chance * value(inference, n, 0.7)
			chance *= seen / float64(n+1)
		}
		if math.Abs(mean-expected) > 1e-6*expected {
			t.Errorf("inference %d: mean appraisal %v, want %v", inference, mean, expected)
		}
	}
}

// TestAppraisePieceHint 指认一件藏品的提示只改这一件是什么的概率
func TestAppraisePieceHint(t *testing.T) {
	prints := []Footprint{{X: 2, Y: 1, Width: 1, Height: 1}}
	hint := func(x, quality int16) []HintView {
		return []HintView{{Kind: HostHintPiece, X: x, Y: 1, Quality: quality}}
	}
	cases := []struct {
		name  string
		odds  []venue.PieceOdds
		trust int32
		hints []HintView
		want  float64
	}{
		// 是品质 3 且说对: 0.2 × 0.8; 是品质 1 被说高: 0.8 × 0.1
		{"说是品质 3", pairOdds, 8000, hint(2, 3), (0.16*1000 + 0.08*50) / 0.24},
		// 是品质 1 且说对: 0.8 × 0.8; 是品质 3 被说低: 0.2 × 0.1
		{"说是品质 1", pairOdds, 8000, hint(2, 1), (0.64*50 + 0.02*1000) / 0.66},
		{"全信", pairOdds, 10000, hint(2, 3), 1000},
		{"不信", pairOdds, 0, hint(2, 3), 240},
		{"指的不是扫出的", pairOdds, 8000, hint(0, 3), 240},
		{"说的比能出现的都高, 只可能是品质 3 被说高", pairOdds, 8000, hint(2, 5), 1000},
		{"同一件说两次", pairOdds, 8000, append(hint(2, 3), hint(2, 3)...), (0.128*1000 + 0.008*50) / 0.136},
	}
	for _, c := range cases {
		if got := appraise(c.odds, prints, c.hints, 0, trusting(c.trust)); !near(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	// 2×1 的只可能是品质 1, 说成隔了一档以上的品质不可能发生, 不理会
	wide := []Footprint{{X: 2, Y: 1, Width: 2, Height: 1}}
	if got := appraise(testOdds, wide, hint(2, 5), 0, trusting(8000)); !near(got, 200) {
		t.Errorf("impossible: got %v, want 200", got)
	}
}

// TestAppraiseCountHint 件数的提示把品质不低于它的藏品调整到提示的件数, 说错时差一件
func TestAppraiseCountHint(t *testing.T) {
	hint := func(n int32) []HintView {
		return []HintView{{Kind: HostHintCount, Quality: 3, Count: n}}
	}
	// 什么都没扫出: 品质 3 的期望 1 件, 实际 0、1、2 件的可能性之比为 1 : 1 : 0.5
	cases := []struct {
		name  string
		trust int32
		n     int32
		want  float64
	}{
		{"全信 0 件", 10000, 0, 200},
		{"全信 2 件", 10000, 2, 2200},
		{"0 件可能是 1 件说少了", 8000, 0, 200 + 1000*0.1/0.9},
		{"1 件可能是 0 件或 2 件", 8000, 1, 200 + 1000*(0.8+0.05*2)/0.95},
		{"不信", 0, 0, 1200},
	}
	for _, c := range cases {
		if got := appraise(pairOdds, nil, hint(c.n), 1, trusting(c.trust)); !near(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}

	// 扫出一件、其余的有一半还没扫出: 全信时扫出的和没扫出的加起来正好是提示的件数, 别的品质不变
	for _, n := range []int32{0, 1, 3} {
		b := newBelief(pairOdds, squares(1), 0.5, PerceptionSize, 0)
		b.count(3, n, 1)
		row := b.row(0)
		if high := b.hidden[1] + row[1]; !near(high, float64(n)) || !near(row[0]+row[1], 1) || !near(b.hidden[0], 2) {
			t.Errorf("count %d: hidden %v, outlined %v", n, b.hidden, row)
		}
	}
	// 0 件时扫出的那件只能是品质 1
	if got := appraise(pairOdds, squares(1), hint(0), 0.5, trusting(10000)); !near(got, 50+100) {
		t.Errorf("none with an outline: got %v, want 150", got)
	}
	// 数的是全部藏品时, 扫出的那件一定在里面, 其余的算在没扫出的里面
	b := newBelief(pairOdds, squares(1), 0.5, PerceptionSize, 0)
	b.count(1, 3, 1)
	if !near(b.hidden[0], 1.6) || !near(b.hidden[1], 0.4) || !near(b.row(0)[0], 0.8) {
		t.Errorf("count all: hidden %v, outlined %v", b.hidden, b.row(0))
	}
}

// TestAppraiseBestHint 最好的品质的提示: 比它好的都没有, 它至少有一件
func TestAppraiseBestHint(t *testing.T) {
	hint := func(quality int16) []HintView {
		return []HintView{{Kind: HostHintBest, Quality: quality}}
	}
	// 什么都没扫出: 件数服从泊松分布, 已知至少有一件时期望为 λ ÷ (1 - e^-λ)
	low := 50 * 4 / (1 - math.Exp(-4))
	high := 200 + 1000/(1-math.Exp(-1))
	if got := appraise(pairOdds, nil, hint(1), 1, trusting(10000)); !near(got, low) {
		t.Errorf("best 1: got %v, want %v", got, low)
	}
	if got := appraise(pairOdds, nil, hint(3), 1, trusting(10000)); !near(got, high) {
		t.Errorf("best 3: got %v, want %v", got, high)
	}
	if got := appraise(pairOdds, nil, hint(3), 1, trusting(0)); !near(got, 1200) {
		t.Errorf("no trust: got %v, want 1200", got)
	}
	// 半信半疑时在不看提示和全信之间
	if got := appraise(pairOdds, nil, hint(3), 1, trusting(8000)); got <= 1200 || got >= high {
		t.Errorf("best 3 at 80%%: got %v, want between 1200 and %v", got, high)
	}
	if got := appraise(pairOdds, nil, hint(1), 1, trusting(8000)); got <= low || got >= 1200 {
		t.Errorf("best 1 at 80%%: got %v, want between %v and 1200", got, low)
	}

	// 全部扫出: 只有一件时它就是最好的; 两件时至少一件是品质 3, 每件是品质 3 的概率为 0.2 ÷ (1 - 0.8²)
	if got := appraise(pairOdds, squares(1), hint(3), 0, trusting(10000)); !near(got, 1000) {
		t.Errorf("one piece, best 3: got %v, want 1000", got)
	}
	if got := appraise(pairOdds, squares(1), hint(1), 0, trusting(10000)); !near(got, 50) {
		t.Errorf("one piece, best 1: got %v, want 50", got)
	}
	share := 0.2 / 0.36
	if got, want := appraise(pairOdds, squares(2), hint(3), 0, trusting(10000)), 2*(share*1000+(1-share)*50); !near(got, want) {
		t.Errorf("two pieces, best 3: got %v, want %v", got, want)
	}
	// 说的品质不可能是最好的(已经指认出更好的)时不理会
	b := newBelief(pairOdds, squares(1), 0, PerceptionSize, 0)
	b.piece(0, 0, 3, 1)
	b.best(1, 1)
	if mean, _ := b.value(); !near(mean, 1000) {
		t.Errorf("contradiction: got %v, want 1000", mean)
	}
}

// TestLean 价值高低的提示: 按对数正态分布, 与直接抽样算出的条件期望一致
func TestLean(t *testing.T) {
	const mean, variance, expected = 3000.0, 4e6, 2900.0
	sigma := math.Sqrt(math.Log(1 + variance/(mean*mean)))
	mu := math.Log(mean) - sigma*sigma/2
	rng := rand.New(rand.NewPCG(1, 2))
	const samples = 400000
	var sum, count [2]float64 // 不高于期望、高于期望
	for range samples {
		v := math.Exp(mu + sigma*rng.NormFloat64())
		i := 0
		if v > expected {
			i = 1
		}
		sum[i] += v
		count[i]++
	}
	below, above, over := sum[0]/count[0], sum[1]/count[1], count[1]/samples
	// 信任度 0.8 时, 说"高于期望"而实际也高于期望的可能性
	likely := 0.8 * over / (0.8*over + 0.2*(1-over))
	cases := []struct {
		name         string
		above, below int
		trust        float64
		want         float64
	}{
		{"全信高于期望", 1, 0, 1, above},
		{"全信不高于期望", 0, 1, 1, below},
		{"半信高于期望", 1, 0, 0.8, likely*above + (1-likely)*below},
		{"信任度一半等于没说", 1, 0, 0.5, mean},
		{"两种说法各一次", 1, 1, 0.8, mean},
	}
	for _, c := range cases {
		if got := lean(mean, variance, expected, c.above, c.below, c.trust); math.Abs(got-c.want) > 0.01*c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	if got := lean(mean, 0, expected, 1, 0, 0.8); got != mean {
		t.Errorf("no uncertainty: got %v, want %v", got, mean)
	}
	// 估值已经远高于期望时, 再说"高于期望"几乎不改变估值
	if got := lean(10*expected, variance, expected, 1, 0, 0.8); math.Abs(got-10*expected) > 0.001*expected {
		t.Errorf("already far above: got %v, want about %v", got, 10*expected)
	}
}

// TestAppraiseHints 提示按种类依次修正, 与给出的先后无关; 单独一条说值钱的提示, 信任度越高估值越高; 件数和最好的品质都说了时, 同样的藏品不算两次
func TestAppraiseHints(t *testing.T) {
	prints := []Footprint{{X: 0, Width: 1, Height: 1}, {X: 3, Width: 1, Height: 2}, {X: 5, Width: 2, Height: 1}}
	hints := []HintView{
		{Kind: HostHintValue, Above: true},
		{Kind: HostHintBest, Quality: 3},
		{Kind: HostHintPiece, X: 0, Quality: 3},
		{Kind: HostHintCount, Quality: 3, Count: 4},
	}
	p := &basicParams{Perception: PerceptionSize, CountInference: 3000, HintTrust: 8000}
	want := appraise(testOdds, prints, hints, 0.6, p)
	if plain := appraise(testOdds, prints, nil, 0.6, p); want <= plain {
		t.Fatalf("with hints %v, without %v", want, plain)
	}
	shuffled := slices.Clone(hints)
	for range 20 {
		rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		if got := appraise(testOdds, prints, shuffled, 0.6, p); !near(got, want) {
			t.Fatalf("order %v: got %v, want %v", shuffled, got, want)
		}
	}

	// 没有扫出 1×2 的那件(它只可能是品质 3)时, 每条提示都有信息量; 信任度一半时价值高低的提示等于没说
	prints = []Footprint{{X: 0, Width: 1, Height: 1}, {X: 5, Width: 2, Height: 1}}
	for _, h := range hints {
		previous := appraise(testOdds, prints, nil, 0.6, p)
		for _, trust := range []int32{6000, 8000, 10000} {
			got := appraise(testOdds, prints, []HintView{h}, 0.6, &basicParams{Perception: PerceptionSize, CountInference: 3000, HintTrust: trust})
			if got <= previous {
				t.Errorf("%s at trust %d: got %v, want above %v", h.Kind, trust, got, previous)
			}
			previous = got
		}
	}

	// 件数的提示已经说了品质 3 的有 1 件, "最好的是品质 3"不会再让它变多
	both := []HintView{{Kind: HostHintBest, Quality: 3}, {Kind: HostHintCount, Quality: 3, Count: 1}}
	if got := appraise(pairOdds, nil, both, 1, trusting(10000)); !near(got, 200+1000) {
		t.Errorf("count and best: got %v, want 1200", got)
	}
	// 品质 3 及以上共 2 件、最好的是品质 3: 这 2 件都是品质 3
	odds := append(slices.Clone(pairOdds), venue.PieceOdds{Quality: 5, Width: 1, Height: 1, Count: 0.5, Price: 3000})
	both[1].Count = 2
	if got := appraise(odds, nil, both, 1, trusting(10000)); !near(got, 200+2*1000) {
		t.Errorf("count and best below the top: got %v, want 2200", got)
	}
	// 最好的是品质 5 时, 这 2 件里至少有一件是品质 5, 比只知道件数时估得高
	both[0].Quality = 5
	if got, only := appraise(odds, nil, both, 1, trusting(10000)), appraise(odds, nil, both[1:], 1, trusting(10000)); got <= only || got >= 200+2*3000 {
		t.Errorf("count and best at the top: got %v, want between %v and 6200", got, only)
	}
}

// TestAppraiseRobust 随机的公示分布、轮廓和提示(包括互相矛盾的)下, 估值都是有限的非负数, 扫出的每件是各种藏品的概率之和仍为 1
func TestAppraiseRobust(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	kinds := []string{HostHintCount, HostHintBest, HostHintPiece, HostHintValue}
	for trial := range 20000 {
		var odds []venue.PieceOdds
		for range 1 + rng.IntN(8) {
			odds = append(odds, venue.PieceOdds{
				Quality: int16(1 + rng.IntN(6)), Width: int16(1 + rng.IntN(3)), Height: int16(1 + rng.IntN(3)),
				Count: math.Pow(10, 3*rng.Float64()-2), Price: math.Pow(10, 5*rng.Float64()),
			})
		}
		prints := make([]Footprint, rng.IntN(6))
		for i := range prints {
			prints[i] = Footprint{X: int16(i), Width: int16(1 + rng.IntN(3)), Height: int16(1 + rng.IntN(3))}
		}
		hints := make([]HintView, rng.IntN(7))
		for i := range hints {
			hints[i] = HintView{Kind: kinds[rng.IntN(len(kinds))], Quality: int16(rng.IntN(8)), Count: int32(rng.IntN(12)), X: int16(rng.IntN(7)), Above: rng.IntN(2) == 0}
		}
		p := &basicParams{
			Perception:     []string{PerceptionSize, PerceptionArea, PerceptionQuality}[rng.IntN(3)],
			CountInference: int32(rng.IntN(3) * 5000),
			HintTrust:      []int32{1, 5000, 8000, 10000}[rng.IntN(4)],
		}
		hidden := []float64{0, 0.3, 0.7, 1}[rng.IntN(4)]
		got := appraise(odds, prints, hints, hidden, p)
		if math.IsNaN(got) || math.IsInf(got, 0) || got < 0 {
			t.Fatalf("trial %d: got %v for odds %+v, outlines %+v, hints %+v, hidden %v, params %+v", trial, got, odds, prints, hints, hidden, p)
		}

		b := newBelief(odds, prints, hidden, p.Perception, float64(p.CountInference)/venue.BasisPoints)
		trust := float64(p.HintTrust) / venue.BasisPoints
		for _, h := range hints {
			switch h.Kind {
			case HostHintPiece:
				b.piece(h.X, h.Y, h.Quality, trust)
			case HostHintCount:
				b.count(h.Quality, h.Count, trust)
			case HostHintBest:
				b.best(h.Quality, trust)
			}
		}
		for f := range prints {
			var sum float64
			for _, share := range b.row(f) {
				sum += share
			}
			if math.Abs(sum-1) > 1e-6 || slices.ContainsFunc(b.row(f), func(share float64) bool { return share < 0 }) {
				t.Fatalf("trial %d: outline %d shares %v after hints %+v", trial, f, b.row(f), hints)
			}
		}
		if slices.ContainsFunc(b.hidden, func(n float64) bool { return n < 0 || math.IsNaN(n) || math.IsInf(n, 0) }) {
			t.Fatalf("trial %d: hidden %v after hints %+v", trial, b.hidden, hints)
		}
	}
}
