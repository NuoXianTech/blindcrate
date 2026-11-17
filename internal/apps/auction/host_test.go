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

	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
)

// sequence 依次返回给定的随机数, 用完后返回 0
func sequence(values ...int64) func(int64) int64 {
	return func(int64) int64 {
		if len(values) == 0 {
			return 0
		}
		v := values[0]
		values = values[1:]
		return v
	}
}

// testHost 第 2 轮的局面: 藏品见 testPieces, 首轮扫出轮廓的是第 1、3 件 (品质 1、2), 价值高于期望; 件数提示数品质 4 及以上
func testHost(persona string, lieRate int32) *hostInput {
	return &hostInput{
		Round:        2,
		Items:        testPieces(),
		Value:        100,
		Expected:     80,
		Qualities:    []int16{1, 2, 3, 4, 5},
		CountQuality: 4,
		Rule:         &venue.HostRule{Enabled: true, LieRate: lieRate, HighQuality: 4},
		Persona:      persona,
		Hype:         hypeStart,
	}
}

func TestLieRate(t *testing.T) {
	low := HostHint{Kind: HostHintPiece, Quality: 2}
	high := HostHint{Kind: HostHintBest, Quality: 5}
	value := HostHint{Kind: HostHintValue}
	cases := []struct {
		name    string
		persona string
		hype    int32
		hint    HostHint
		want    int32
	}{
		{"稳重减半", venue.HostSteady, hypeStart, low, 900},
		{"稳重的高品质", venue.HostSteady, hypeStart, high, 1350},
		{"煽动平时", venue.HostShowman, hypeStart, low, 1800},
		{"煽动热度高", venue.HostShowman, hypeHot, low, 2500},
		{"狡黠热度低", venue.HostSly, hypeCold, low, 2500},
		{"狡黠热度高", venue.HostSly, hypeHot, low, 1800},
		{"价值提示不算高品质", venue.HostSly, hypeStart, value, 1800},
	}
	for _, c := range cases {
		in := testHost(c.persona, 1800)
		in.Hype = c.hype
		if got := lieRate(in, &c.hint); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestLie(t *testing.T) {
	cases := []struct {
		name     string
		persona  string
		hint     HostHint
		random   []int64
		want     HostHint
		truthful bool
	}{
		{"煽动往高说", venue.HostShowman, HostHint{Kind: HostHintPiece, Quality: 3}, nil, HostHint{Kind: HostHintPiece, Quality: 4}, false},
		{"煽动遇到最高品质照实说", venue.HostShowman, HostHint{Kind: HostHintBest, Quality: 5}, nil, HostHint{Kind: HostHintBest, Quality: 5}, true},
		{"狡黠往低说", venue.HostSly, HostHint{Kind: HostHintCount, Count: 2}, nil, HostHint{Kind: HostHintCount, Count: 1}, false},
		{"狡黠遇到 0 件照实说", venue.HostSly, HostHint{Kind: HostHintCount}, nil, HostHint{Kind: HostHintCount}, true},
		{"狡黠说不值钱", venue.HostSly, HostHint{Kind: HostHintValue, Above: true}, nil, HostHint{Kind: HostHintValue}, false},
		{"煽动遇到本来就值钱照实说", venue.HostShowman, HostHint{Kind: HostHintValue, Above: true}, nil, HostHint{Kind: HostHintValue, Above: true}, true},
		{"稳重往高说不了时往低说", venue.HostSteady, HostHint{Kind: HostHintBest, Quality: 5}, []int64{0}, HostHint{Kind: HostHintBest, Quality: 4}, false},
		{"稳重往低说", venue.HostSteady, HostHint{Kind: HostHintPiece, Quality: 3}, []int64{1}, HostHint{Kind: HostHintPiece, Quality: 2}, false},
	}
	for _, c := range cases {
		hint := c.hint
		hint.Truthful = true
		lie(testHost(c.persona, 1800), &hint, sequence(c.random...))
		c.want.Truthful = c.truthful
		if hint != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, hint, c.want)
		}
	}
}

func TestMakeHint(t *testing.T) {
	// 不撒谎: 候选为 件数(品质 4 及以上 3 件)、最好(5)、单件(按随机数选扫出轮廓的第 2 件, 位置 (2, 0))、价值(高于期望);
	// 随机数依次为 指认哪一件(有可以指认的才抽)、选哪一种
	count := HostHint{Round: 2, Kind: HostHintCount, Quality: 4, Count: 3, Truthful: true}
	best := HostHint{Round: 2, Kind: HostHintBest, Quality: 5, Truthful: true}
	piece := HostHint{Round: 2, Kind: HostHintPiece, Quality: 2, X: 2, Truthful: true}
	value := HostHint{Round: 2, Kind: HostHintValue, Above: true, Truthful: true}
	pointed := HostHint{Kind: HostHintPiece, Quality: 1} // 已经指过位置 (0, 0) 的那件
	cases := []struct {
		name   string
		given  []HostHint
		random []int64
		want   HostHint
	}{
		{"件数", nil, []int64{1, 0}, count},
		{"最好", nil, []int64{1, 1}, best},
		{"单件", nil, []int64{1, 2}, piece},
		{"价值", nil, []int64{1, 3}, value},
		{"说过的不再说", []HostHint{count, best}, []int64{1, 1}, value},
		{"指过的不再指", []HostHint{pointed, value}, []int64{0, 2}, piece},
		{"不连续出同一种", []HostHint{pointed}, []int64{0, 2}, value},
		{"只剩一种时可以连续", []HostHint{count, best, value, pointed}, []int64{0, 0}, piece},
	}
	for _, c := range cases {
		in := testHost(venue.HostShowman, 0)
		in.Given = c.given
		got, ok := makeHint(in, sequence(c.random...))
		if !ok || got != c.want {
			t.Errorf("%s: got %+v %v, want %+v", c.name, got, ok, c.want)
		}
	}
	// 能说的都说过、扫出轮廓的都指过时没有提示
	in := testHost(venue.HostShowman, 0)
	in.Given = []HostHint{count, best, value, pointed, piece}
	if got, ok := makeHint(in, first); ok {
		t.Errorf("everything said: got %+v", got)
	}

	// 撒谎: 随机数依次为 单件选第 1 件(位置 (0, 0), 品质 1)、选单件、撒谎判定 0 < 撒谎率
	in = testHost(venue.HostShowman, 1800)
	got, _ := makeHint(in, sequence(0, 2, 0))
	want := HostHint{Round: 2, Kind: HostHintPiece, Quality: 2}
	if got != want {
		t.Errorf("lie: got %+v, want %+v", got, want)
	}

	// 没有扫出轮廓时不指认单件: 候选为 件数、最好、价值
	in = testHost(venue.HostSteady, 0)
	for i := range in.Items {
		in.Items[i].OutlineRound = 0
	}
	got, _ = makeHint(in, sequence(2))
	if got.Kind != HostHintValue {
		t.Errorf("no preview: got %+v", got)
	}
	// 公示分布里没有合适的品质时不给件数提示: 候选为 最好、价值
	in.CountQuality = 0
	got, _ = makeHint(in, sequence(0))
	if got.Kind != HostHintBest {
		t.Errorf("no count quality: got %+v", got)
	}
	// 空货仓没有期望价值时没有能给的提示
	in.Items, in.Expected, in.Qualities = nil, 0, nil
	if _, ok := makeHint(in, first); ok {
		t.Error("nothing to say: got a hint")
	}
}

func TestHintCountQuality(t *testing.T) {
	// 品质从 1 开始, 依次给出每个品质的期望件数
	odds := func(counts ...float64) []venue.PieceOdds {
		var odds []venue.PieceOdds
		for i, count := range counts {
			odds = append(odds, venue.PieceOdds{Quality: int16(i + 1), Width: 1, Height: 1, Count: count / 2, Price: 100}, venue.PieceOdds{Quality: int16(i + 1), Width: 2, Height: 1, Count: count / 2, Price: 300})
		}
		return odds
	}
	cases := []struct {
		name string
		odds []venue.PieceOdds
		want int16
	}{
		// 不低于各品质的期望件数为 9.45、4.45、1.45、0.25、0.05
		{"最接近 1 件的品质", odds(5, 3, 1.2, 0.2, 0.05), 3},
		// 6.8、1.8、0.5: 1.8 差 0.8 件, 0.5 只差 0.5 件, 按倍数比 1.8 倍比 2 倍近
		{"远近按倍数比", odds(5, 1.3, 0.5), 2},
		// 6、2、0.5: 2 倍和一半一样远
		{"一样近时取低的", odds(4, 1.5, 0.5), 2},
		// 8.1 太多, 0.1 太少
		{"都不合适", odds(8, 0.1), 0},
		{"没有公示分布", nil, 0},
	}
	for _, c := range cases {
		if got := hintCountQuality(c.odds); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestHypeDelta(t *testing.T) {
	cases := []struct {
		name     string
		raises   []int64
		expected int64
		want     int32
	}{
		{"没人加价", nil, 1000, -hypeIdle},
		{"大幅加价封顶, 小幅至少 1", []int64{2000, 1}, 1000, hypeRaiseMax + 1 - hypeCooling},
		{"按正常节奏", []int64{200, 200, 200, 200}, 1000, 4*hypePacePoints - hypeCooling},
		{"两倍节奏", []int64{400}, 1000, 2*hypePacePoints - hypeCooling},
		{"没有期望价值时按封顶", []int64{1}, 0, hypeRaiseMax - hypeCooling},
	}
	for _, c := range cases {
		if got := hypeDelta(c.raises, c.expected, 5); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}
