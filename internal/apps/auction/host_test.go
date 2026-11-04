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

// testHost 第 2 轮的局面: 还没公开的是扫描顺序 1、2、4、6 (品质 1、4、5、3), 价值高于期望
func testHost(persona string, lieRate int32) *hostInput {
	return &hostInput{
		Round:     2,
		Items:     testPieces(),
		Value:     100,
		Expected:  80,
		Qualities: []int16{1, 2, 3, 4, 5},
		Rule:      &venue.HostRule{Enabled: true, LieRate: lieRate, HighQuality: 4},
		Persona:   persona,
		Hype:      hypeStart,
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
	// 不撒谎: 候选为 件数(品质 4 及以上 3 件)、最好(5)、单件(按随机数选还没公开的第 2 件, 扫描顺序 2)、价值(高于期望)
	cases := []struct {
		name   string
		last   string
		random []int64
		want   HostHint
	}{
		{"件数", "", []int64{1, 0}, HostHint{Round: 2, Kind: HostHintCount, Quality: 4, Count: 3, Truthful: true}},
		{"最好", "", []int64{1, 1}, HostHint{Round: 2, Kind: HostHintBest, Quality: 5, Truthful: true}},
		{"单件", "", []int64{1, 2}, HostHint{Round: 2, Kind: HostHintPiece, Quality: 4, ScanOrder: 2, Truthful: true}},
		{"价值", "", []int64{1, 3}, HostHint{Round: 2, Kind: HostHintValue, Above: true, Truthful: true}},
		{"不连续出同一种", HostHintCount, []int64{1, 0}, HostHint{Round: 2, Kind: HostHintBest, Quality: 5, Truthful: true}},
	}
	for _, c := range cases {
		in := testHost(venue.HostShowman, 0)
		in.Last = c.last
		got, ok := makeHint(in, sequence(c.random...))
		if !ok || got != c.want {
			t.Errorf("%s: got %+v %v, want %+v", c.name, got, ok, c.want)
		}
	}

	// 撒谎: 随机数依次为 单件选第 1 件(扫描顺序 1, 品质 1)、选单件、撒谎判定 0 < 撒谎率
	in := testHost(venue.HostShowman, 1800)
	got, _ := makeHint(in, sequence(0, 2, 0))
	want := HostHint{Round: 2, Kind: HostHintPiece, Quality: 2, ScanOrder: 1}
	if got != want {
		t.Errorf("lie: got %+v, want %+v", got, want)
	}

	// 全部公开后只剩件数和价值, 没有期望价值和高品质的仓库没有能给的提示
	in = testHost(venue.HostSteady, 0)
	in.Round = 6
	got, _ = makeHint(in, sequence(1))
	if got.Kind != HostHintValue {
		t.Errorf("revealed: got %+v", got)
	}
	in.Expected, in.Rule.HighQuality, in.Qualities = 0, 0, nil
	if _, ok := makeHint(in, first); ok {
		t.Error("nothing to say: got a hint")
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

func TestIntelFor(t *testing.T) {
	cases := []struct {
		name   string
		round  int32
		rounds int32
		known  int32 // 已公开前几件
		want   string
	}{
		{"前三分之一看品质", 1, 5, 1, EffectRevealQuality},
		{"中间看名称", 2, 5, 2, EffectRevealName},
		{"中间的最后一轮", 3, 5, 3, EffectRevealName},
		{"之后看价值区间", 4, 5, 4, EffectValueRange},
		{"加轮看价值区间", 7, 5, 6, EffectValueRange},
		{"品质都知道了改看名称", 1, 5, 6, EffectRevealName},
	}
	for _, c := range cases {
		ctx := newUseContext(c.known, 6, 1000, testPieces(), nil, first)
		code, intel, err := intelFor(c.round, c.rounds, 1500, ctx)
		if err != nil || code != c.want {
			t.Errorf("%s: got %s %v, want %s", c.name, code, err, c.want)
			continue
		}
		switch code {
		case EffectRevealQuality:
			if len(intel.Pieces) != 1 || ctx.Known[intel.Pieces[0].ScanOrder] {
				t.Errorf("%s: got %+v", c.name, intel)
			}
		case EffectRevealName:
			if len(intel.Pieces) != 1 || intel.Pieces[0].Name == "" {
				t.Errorf("%s: got %+v", c.name, intel)
			}
		case EffectValueRange:
			if intel.ValueMin != 1000 || intel.ValueMax != 1300 {
				t.Errorf("%s: got %+v", c.name, intel)
			}
		}
	}
}

func TestIntelCost(t *testing.T) {
	uses := []Use{{Seat: 1, ItemID: 9}, {Seat: 1, Cost: 30}, {Seat: 2, Cost: 30}, {Seat: 1, Cost: 30}}
	if got := intelCost(uses, 1); got != 60 {
		t.Errorf("cost: got %d", got)
	}
	if got := itemUses(uses); got != 1 {
		t.Errorf("item uses: got %d", got)
	}
	seats := intelSeats([]Player{{Seat: 1}, {Seat: 2}, {Seat: 3}}, []Use{{Seat: 1, Round: 1}, {Seat: 1, Round: 3}, {Seat: 2, Round: 2, ItemID: 5}})
	if len(seats) != 1 || seats[0] != (SeatIntel{Seat: 1, Count: 2, Round: 3}) {
		t.Errorf("seats: got %+v", seats)
	}
}
