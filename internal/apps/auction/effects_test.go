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
	"errors"
	"reflect"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/effect"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
)

// testPieces 6 件 1×1 的藏品排成一行, 扫描顺序 1-6 即 X 为 0-5, 品质各不相同; 第 1、3 件第 1 轮扫出轮廓, 第 6 件第 3 轮扫出
func testPieces() []DepotItem {
	items := []DepotItem{
		{OutlineRound: 1, Quality: 1, Name: "a"},
		{Quality: 4, Name: "b"},
		{OutlineRound: 1, Quality: 2, Name: "c"},
		{Quality: 5, Name: "d"},
		{Quality: 4, Name: "e"},
		{OutlineRound: 3, Quality: 3, Name: "f"},
	}
	for i := range items {
		items[i].ScanOrder, items[i].X, items[i].Width, items[i].Height = int32(i+1), int16(i), 1, 1
		items[i].ImageURL = "/" + items[i].Name
	}
	return items
}

// nextPieces 下一个拍品的 3 件藏品: (0,0) 的 2×1、(2,0) 的 1×1 和 (0,1) 的 1×2
func nextPieces() []DepotItem {
	return []DepotItem{
		{LotNo: 3, ScanOrder: 1, X: 0, Y: 0, Width: 2, Height: 1, Quality: 5, Name: "x"},
		{LotNo: 3, ScanOrder: 2, X: 2, Y: 0, Width: 1, Height: 1, Quality: 2, Name: "y"},
		{LotNo: 3, ScanOrder: 3, X: 0, Y: 1, Width: 1, Height: 2, Quality: 3, Name: "z"},
	}
}

// 测试用的拍品: 3 个拍品中的第 2 个, 从仓库的第 5 轮开拍
const (
	testLotNo      = 2
	testFirstRound = 5
)

// testScene 拍品 2 的第 round 轮: 货仓 6×2, 第一行摆满 testPieces, 第二行是空的; 规则为 4 轮、加轮 1 轮、每 2 轮再扫描一次;
// 期望价值 80、期望 5 件; 下一个拍品见 nextPieces。uses 没写拍品和轮次的算用在这个拍品的第 1 轮
func testScene(round int32, uses ...Use) *useScene {
	for i := range uses {
		if uses[i].LotNo == 0 {
			uses[i].LotNo = testLotNo
		}
		if uses[i].Round == 0 {
			uses[i].Round = testFirstRound
		}
	}
	return &useScene{
		Lot: &Lot{
			LotNo: testLotNo, FirstRound: testFirstRound, Grid: venue.Grid{Width: 6, Height: 2}, Value: 100,
			Odds: []venue.PieceOdds{{Quality: 1, Count: 4, Price: 15}, {Quality: 4, Count: 1, Price: 20}},
		},
		Rule:      venue.AuctionRule{Rounds: 4, MaxExtraRounds: 1, RescanEvery: 2, RescanRate: 1500},
		Round:     testFirstRound + round - 1,
		Items:     testPieces(),
		Next:      &Lot{LotNo: testLotNo + 1, Grid: venue.Grid{Width: 3, Height: 3}},
		NextItems: nextPieces(),
		Uses:      uses,
	}
}

// testContext 见 testScene, 随机数总是 0
func testContext(round int32, uses ...Use) *UseContext {
	return newUseContext(testScene(round, uses...), first)
}

// unusable 效果返回的是现在不能使用的错误, 原因为 reason
func unusable(err error, reason string) bool {
	var use *effect.UseError
	return errors.As(err, &use) && use.Reason == reason
}

// intelAt 第 x 件(从 0 开始)的情报, name 为空时只有品质
func intelAt(x int16, quality int16, name string) PieceIntel {
	piece := PieceIntel{X: x, Width: 1, Height: 1, Quality: quality, Name: name}
	if name != "" {
		piece.ImageURL = "/" + name
	}
	return piece
}

// first 总是返回 0 的随机数
func first(int64) int64 { return 0 }

func TestUseContextKnown(t *testing.T) {
	uses := []Use{{Result: Intel{Pieces: []PieceIntel{intelAt(1, 4, ""), intelAt(3, 5, "d"), intelAt(5, 0, "")}}}}
	ctx := testContext(1, uses...)
	// 第 1 轮扫出的 0、2 加上之前看到的, 只扫出轮廓的不算已知品质
	if want := map[cell]bool{{0, 0}: true, {1, 0}: true, {2, 0}: true, {3, 0}: true, {5, 0}: true}; !reflect.DeepEqual(ctx.Outlined, want) {
		t.Errorf("outlined: got %v, want %v", ctx.Outlined, want)
	}
	if want := map[cell]bool{{1, 0}: true, {3, 0}: true}; !reflect.DeepEqual(ctx.Known, want) {
		t.Errorf("known: got %v, want %v", ctx.Known, want)
	}
	if !reflect.DeepEqual(ctx.KnownNames, map[cell]bool{{3, 0}: true}) {
		t.Errorf("known names: got %v", ctx.KnownNames)
	}
}

func TestRevealOutline(t *testing.T) {
	// 跳过第 1 轮扫出的 0、2 和之前看到的 1, 只给轮廓
	uses := []Use{{Result: Intel{Pieces: []PieceIntel{intelAt(1, 4, "")}}}}
	ctx := testContext(1, uses...)
	got, _ := revealOutline(ctx, &RevealOutlineParams{Count: 2})
	want := []PieceIntel{intelAt(3, 0, ""), intelAt(4, 0, "")}
	if !reflect.DeepEqual(got.Pieces, want) {
		t.Errorf("got %+v, want %+v", got.Pieces, want)
	}
	// 不够时全给
	got, _ = revealOutline(ctx, &RevealOutlineParams{Count: 10})
	if len(got.Pieces) != 3 {
		t.Errorf("got %d pieces, want 3", len(got.Pieces))
	}
	// 第 3 轮再次扫描扫出的 5 也跳过
	got, _ = revealOutline(testContext(3, uses...), &RevealOutlineParams{Count: 10})
	if len(got.Pieces) != 2 {
		t.Errorf("round 3: got %d pieces, want 2", len(got.Pieces))
	}
}

func TestRevealQuality(t *testing.T) {
	// 随机数为 0 时依次选剩下的第一件, 结果按位置排列, 带轮廓不带名称
	ctx := testContext(1)
	got, _ := revealQuality(ctx, &RevealQualityParams{Count: 2})
	want := []PieceIntel{intelAt(0, 1, ""), intelAt(1, 4, "")}
	if !reflect.DeepEqual(got.Pieces, want) {
		t.Errorf("got %+v, want %+v", got.Pieces, want)
	}

	// 已经看过的跳过, 不够时有几件给几件; 首轮扫出的轮廓不算知道品质
	uses := []Use{{Result: Intel{Pieces: []PieceIntel{intelAt(1, 4, ""), intelAt(3, 5, ""), intelAt(4, 4, ""), intelAt(5, 3, "")}}}}
	ctx = testContext(1, uses...)
	got, _ = revealQuality(ctx, &RevealQualityParams{Count: 5})
	want = []PieceIntel{intelAt(0, 1, ""), intelAt(2, 2, "")}
	if !reflect.DeepEqual(got.Pieces, want) {
		t.Errorf("rest: got %+v, want %+v", got.Pieces, want)
	}
}

func TestCountQuality(t *testing.T) {
	ctx := testContext(1)
	got, _ := countQuality(ctx, &CountQualityParams{MinQuality: 4})
	want := Intel{Facts: []Fact{{Kind: FactCounts, Label: labelAtLeast, Quality: 4, Counts: []QualityCount{{Quality: 5, Count: 1}, {Quality: 4, Count: 2}}}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	got, _ = countQuality(ctx, &CountQualityParams{MinQuality: 6})
	if len(got.Facts) != 1 || len(got.Facts[0].Counts) != 0 || got.Facts[0].Counts == nil {
		t.Errorf("none: got %+v", got.Facts)
	}
}

func TestRevealName(t *testing.T) {
	ctx := testContext(1)
	got, _ := revealName(ctx, &RevealNameParams{Count: 2, MinQuality: 4})
	want := []PieceIntel{intelAt(1, 4, "b"), intelAt(3, 5, "d")}
	if !reflect.DeepEqual(got.Pieces, want) {
		t.Errorf("got %+v, want %+v", got.Pieces, want)
	}

	// 已知名称的跳过; 没有符合条件的藏品时为空
	ctx = testContext(1, Use{Result: got})
	got, _ = revealName(ctx, &RevealNameParams{Count: 2, MinQuality: 4})
	if want := []PieceIntel{intelAt(4, 4, "e")}; !reflect.DeepEqual(got.Pieces, want) {
		t.Errorf("rest: got %+v, want %+v", got.Pieces, want)
	}
	got, _ = revealName(ctx, &RevealNameParams{Count: 1, MinQuality: 6})
	if len(got.Pieces) != 0 || len(got.Facts) != 1 || got.Facts[0].Quality != 6 || got.Facts[0].Kind != FactText {
		t.Errorf("none: got %+v", got)
	}
}

func TestValueRange(t *testing.T) {
	// 期望价值 10000、误差 20% 时区间宽 4000, 与货仓实际值多少无关: 从区间的宽度推不出货仓价值
	cases := []struct {
		name     string
		value    int64
		offset   int64
		min, max int64
	}{
		{"value at top", 10000, 4000, 6000, 10000},
		{"value at bottom", 10000, 0, 10000, 14000},
		{"value in middle", 10000, 1500, 8500, 12500},
		{"cheap depot", 2000, 1500, 500, 4500},
		{"valuable depot", 30000, 1500, 28500, 32500},
		{"lower bound not below zero", 2000, 3000, 0, 3000},
		{"zero value", 0, 0, 0, 4000},
	}
	for _, c := range cases {
		var bound int64
		ctx := &UseContext{Value: c.value, Expected: 10000, Rand: func(n int64) int64 { bound = n; return c.offset }}
		got, err := valueRange(ctx, &ValueRangeParams{Precision: 2000})
		if err != nil || len(got.Facts) != 1 || got.Facts[0].Min != c.min || got.Facts[0].Max != c.max {
			t.Errorf("%s: got %+v %v, want [%d, %d]", c.name, got, err, c.min, c.max)
		}
		if bound != 4001 {
			t.Errorf("%s: rand bound got %d, want 4001", c.name, bound)
		}
	}
	// 空货仓的期望价值和货仓价值都是 0
	if got, err := valueRange(&UseContext{Rand: first}, &ValueRangeParams{Precision: 2000}); err != nil || got.Facts[0].Min != 0 || got.Facts[0].Max != 0 {
		t.Errorf("empty: got %+v %v", got, err)
	}
	// 没有期望价值时区间没有宽度, 等于直接说出货仓价值, 不给
	if got, err := valueRange(&UseContext{Value: 100, Rand: first}, &ValueRangeParams{Precision: 2000}); err == nil {
		t.Errorf("no expected value: got %+v", got)
	}
}

func TestUseCounts(t *testing.T) {
	uses := []Use{
		{Seat: 1, ItemID: 9, Name: "放大镜"},
		{Seat: 2, ItemID: 9, Name: "放大镜"},
		{Seat: 1, ItemID: 7, Name: "清单"},
		{Seat: 1, ItemID: 9, Name: "放大镜"},
	}
	want := []UseCount{{ItemID: 9, Name: "放大镜", Count: 2}, {ItemID: 7, Name: "清单", Count: 1}}
	if got := useCounts(uses, 1); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if got := useCounts(uses, 3); got == nil || len(got) != 0 {
		t.Errorf("none: got %+v", got)
	}
}

// TestUseContextLots 之前的道具看到的格子按拍品归属, 其他拍品的不算在这个拍品上
func TestUseContextLots(t *testing.T) {
	scene := testScene(2,
		// 上一个拍品上看到的, 位置碰巧相同
		Use{LotNo: 1, Round: 3, Effect: EffectRevealQuality, Result: Intel{Pieces: []PieceIntel{intelAt(4, 4, "")}}},
		// 上一个拍品时提前看到的这个拍品的轮廓
		Use{LotNo: 1, Round: 4, Effect: EffectPeekNext, Result: Intel{Pieces: []PieceIntel{{X: 3, Width: 1, Height: 1, LotNo: testLotNo}}}},
		// 这个拍品上看到的: 一个空格子、一件标了最贵的藏品
		Use{Effect: EffectProbeEmpty, Result: Intel{Pieces: []PieceIntel{{X: 0, Y: 1, Width: 1, Height: 1, Empty: true}}}},
		Use{Round: testFirstRound + 1, Effect: EffectRevealTop, Result: Intel{Pieces: []PieceIntel{{X: 1, Width: 1, Height: 1, Note: "最贵"}}}},
		// 这个拍品上提前看到的下一个拍品的轮廓
		Use{Round: testFirstRound + 1, Effect: EffectPeekNext, Result: Intel{Pieces: []PieceIntel{{X: 2, Width: 1, Height: 1, LotNo: testLotNo + 1}}}},
	)
	scene.Hints = []HostHint{{Round: testFirstRound, Truthful: true}, {Round: testFirstRound + 1}}
	ctx := newUseContext(scene, first)

	if ctx.LotNo != testLotNo || ctx.Round != 2 || ctx.Expected != 80 || ctx.ExpectedCount != 5 || ctx.Grid != (venue.Grid{Width: 6, Height: 2}) || ctx.Value != 100 {
		t.Errorf("ctx = %+v", ctx)
	}
	// 第 1 轮扫出的 0、2, 提前看到的 3 和标了最贵的 1; 上一个拍品的 4 和下一个拍品的 2 不算
	if want := map[cell]bool{{0, 0}: true, {1, 0}: true, {2, 0}: true, {3, 0}: true}; !reflect.DeepEqual(ctx.Outlined, want) {
		t.Errorf("outlined: got %v, want %v", ctx.Outlined, want)
	}
	if len(ctx.Known) != 0 || len(ctx.KnownNames) != 0 {
		t.Errorf("known: got %v, %v", ctx.Known, ctx.KnownNames)
	}
	if want := map[cell]bool{{0, 1}: true}; !reflect.DeepEqual(ctx.Empties, want) {
		t.Errorf("empties: got %v, want %v", ctx.Empties, want)
	}
	if want := map[cell]string{{1, 0}: "最贵"}; !reflect.DeepEqual(ctx.Notes, want) {
		t.Errorf("notes: got %v, want %v", ctx.Notes, want)
	}
	if ctx.Next == nil || ctx.Next.LotNo != testLotNo+1 || len(ctx.Next.Pieces) != 3 || !reflect.DeepEqual(ctx.Next.Outlined, map[cell]bool{{2, 0}: true}) {
		t.Errorf("next: got %+v", ctx.Next)
	}
	// 只有这一轮的提示
	if ctx.Hint != &scene.Hints[1] {
		t.Errorf("hint: got %+v", ctx.Hint)
	}
	// 用在这个拍品上的记下是第几轮, 其他拍品上的不记
	wantPast := []PastUse{
		{Effect: EffectRevealQuality, LotNo: 1},
		{Effect: EffectPeekNext, LotNo: 1},
		{Effect: EffectProbeEmpty, LotNo: testLotNo, Round: 1},
		{Effect: EffectRevealTop, LotNo: testLotNo, Round: 2},
		{Effect: EffectPeekNext, LotNo: testLotNo, Round: 2},
	}
	if !reflect.DeepEqual(ctx.Past, wantPast) {
		t.Errorf("past: got %+v, want %+v", ctx.Past, wantPast)
	}

	// 最后一个拍品没有下一个; 没有主持人时没有提示
	scene = testScene(1)
	scene.Next, scene.NextItems = nil, nil
	if ctx := newUseContext(scene, first); ctx.Next != nil || ctx.Hint != nil || len(ctx.Past) != 0 {
		t.Errorf("last lot: got %+v", ctx)
	}
}

func TestCountRange(t *testing.T) {
	// 期望 5 件、误差 20% 时区间相差 2 件, 与货仓实际有几件无关
	for _, c := range []struct {
		name     string
		pieces   int
		offset   int64
		min, max int64
	}{
		{"count at bottom", 6, 0, 6, 8},
		{"count in middle", 6, 1, 5, 7},
		{"count at top", 6, 2, 4, 6},
		{"few pieces", 3, 2, 1, 3},
		// 下限低于 0 时整个区间上移, 宽度不变: 截断成 0-1 件就等于说出只有 1 件
		{"shifted up", 1, 2, 0, 2},
		{"empty depot", 0, 1, 0, 2},
	} {
		var bound int64
		ctx := &UseContext{Pieces: make([]DepotItem, c.pieces), ExpectedCount: 5, Rand: func(n int64) int64 { bound = n; return c.offset }}
		got, err := countRange(ctx, &CountRangeParams{Precision: 2000})
		want := Intel{Facts: []Fact{{Kind: FactRange, Label: "藏品件数", Min: c.min, Max: c.max, Unit: "件"}}}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %+v %v, want %+v", c.name, got, err, want)
		}
		if bound != 3 {
			t.Errorf("%s: rand bound got %d, want 3", c.name, bound)
		}
	}
	// 至少相差 1 件, 不会直接说出件数; 期望件数按公示概率
	ctx := testContext(1)
	got, err := countRange(ctx, &CountRangeParams{Precision: 100})
	if err != nil || got.Facts[0].Min != 6 || got.Facts[0].Max != 7 {
		t.Errorf("narrow: got %+v %v", got, err)
	}
	got, _ = countRange(ctx, &CountRangeParams{Precision: 10000})
	if got.Facts[0].Min != 6 || got.Facts[0].Max != 16 {
		t.Errorf("wide: got %+v", got)
	}
}

func TestRevealTop(t *testing.T) {
	// 回收价 10、50、30、50、5、20: 第 2、4 件一样贵, 按扫描顺序第 2 件在前
	priced := func(uses ...Use) *UseContext {
		scene := testScene(1, uses...)
		for i, price := range []int64{10, 50, 30, 50, 5, 20} {
			scene.Items[i].SellPrice = price
		}
		return newUseContext(scene, first)
	}
	top := func(x int16, note string) PieceIntel { return PieceIntel{X: x, Width: 1, Height: 1, Note: note} }

	// 只有位置、轮廓和第几贵, 不含品质和名称; 按位置排列
	got, err := revealTop(priced(), &RevealTopParams{Count: 1})
	if want := (Intel{Pieces: []PieceIntel{top(1, "最贵")}}); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("top 1: got %+v %v, want %+v", got, err, want)
	}
	got, err = revealTop(priced(), &RevealTopParams{Count: 3})
	if want := (Intel{Pieces: []PieceIntel{top(1, "最贵"), top(2, "第 3 贵"), top(3, "第 2 贵")}}); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("top 3: got %+v %v, want %+v", got, err, want)
	}

	// 标过的不能再用; 能标出更多时可以
	marked := Use{Effect: EffectRevealTop, Result: Intel{Pieces: []PieceIntel{top(1, "最贵")}}}
	if _, err := revealTop(priced(marked), &RevealTopParams{Count: 1}); !unusable(err, UseTopKnown) {
		t.Errorf("known: err = %v", err)
	}
	if got, err := revealTop(priced(marked), &RevealTopParams{Count: 2}); err != nil || len(got.Pieces) != 2 {
		t.Errorf("more: got %+v %v", got, err)
	}

	// 回收货币不是出价货币的不计入货仓价值, 不算贵
	ctx := priced()
	ctx.Currency = "chip"
	for i := range ctx.Pieces {
		ctx.Pieces[i].SellCurrency = "chip"
	}
	ctx.Pieces[1].SellCurrency = "gem"
	if got, _ := revealTop(ctx, &RevealTopParams{Count: 1}); len(got.Pieces) != 1 || got.Pieces[0].X != 3 {
		t.Errorf("other currency: got %+v", got)
	}

	// 件数不够时有几件标几件; 空货仓照常使用, 说明没有
	ctx = priced()
	ctx.Pieces = ctx.Pieces[:2]
	if got, _ := revealTop(ctx, &RevealTopParams{Count: 3}); len(got.Pieces) != 2 {
		t.Errorf("two pieces: got %+v", got)
	}
	ctx.Pieces = nil
	if got, err := revealTop(ctx, &RevealTopParams{Count: 1}); err != nil || len(got.Pieces) != 0 || len(got.Facts) != 1 || got.Facts[0].Kind != FactText {
		t.Errorf("empty depot: got %+v %v", got, err)
	}
}

func TestProbeEmpty(t *testing.T) {
	empty := func(x, y int16) PieceIntel { return PieceIntel{X: x, Y: y, Width: 1, Height: 1, Empty: true} }

	// 第一行摆满, 只有第二行是空的; 随机数为 0 时依次选
	got, err := probeEmpty(testContext(1), &ProbeEmptyParams{Count: 2})
	if want := (Intel{Pieces: []PieceIntel{empty(0, 1), empty(1, 1)}}); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v %v, want %+v", got, err, want)
	}
	// 已经知道是空的跳过, 不够时有几格给几格
	known := Use{Effect: EffectProbeEmpty, Result: got}
	got, _ = probeEmpty(testContext(1, known), &ProbeEmptyParams{Count: 10})
	if want := []PieceIntel{empty(2, 1), empty(3, 1), empty(4, 1), empty(5, 1)}; !reflect.DeepEqual(got.Pieces, want) {
		t.Errorf("rest: got %+v, want %+v", got.Pieces, want)
	}
	// 都知道了时说明没有
	got, err = probeEmpty(testContext(1, known, Use{Effect: EffectProbeEmpty, Result: got}), &ProbeEmptyParams{Count: 1})
	if err != nil || len(got.Pieces) != 0 || len(got.Facts) != 1 || got.Facts[0].Kind != FactText {
		t.Errorf("none: got %+v %v", got, err)
	}

	// 大件占的每一格都不是空的
	ctx := &UseContext{
		Grid:   venue.Grid{Width: 3, Height: 2},
		Pieces: []DepotItem{{X: 0, Y: 0, Width: 2, Height: 2}},
		Rand:   first,
	}
	got, _ = probeEmpty(ctx, &ProbeEmptyParams{Count: 6})
	if want := []PieceIntel{empty(2, 0), empty(2, 1)}; !reflect.DeepEqual(got.Pieces, want) {
		t.Errorf("large piece: got %+v, want %+v", got.Pieces, want)
	}
}

func TestVerifyHint(t *testing.T) {
	verify := func(round int32, hints []HostHint, uses ...Use) (Intel, error) {
		scene := testScene(round, uses...)
		scene.Hints = hints
		return verifyHint(newUseContext(scene, first), &VerifyHintParams{})
	}
	hints := []HostHint{{Round: testFirstRound, Truthful: true}, {Round: testFirstRound + 1, Truthful: false}}

	if got, err := verify(1, hints); err != nil || len(got.Facts) != 1 || got.Facts[0].Text != "属实" || len(got.Pieces) != 0 {
		t.Errorf("truthful: got %+v %v", got, err)
	}
	if got, err := verify(2, hints); err != nil || len(got.Facts) != 1 || got.Facts[0].Text != "说错了" {
		t.Errorf("lie: got %+v %v", got, err)
	}
	// 这一轮没有提示: 没有主持人、加轮, 之前的提示不能回头验
	if _, err := verify(1, nil); !unusable(err, UseNoHint) {
		t.Errorf("no host: err = %v", err)
	}
	if _, err := verify(3, hints); !unusable(err, UseNoHint) {
		t.Errorf("no hint this round: err = %v", err)
	}
	// 同一条提示只验一次; 上一轮验过的不影响这一轮, 其他拍品上验过的也不影响
	if _, err := verify(2, hints, Use{Round: testFirstRound + 1, Effect: EffectVerifyHint}); !unusable(err, UseHintVerified) {
		t.Errorf("verified: err = %v", err)
	}
	if _, err := verify(2, hints, Use{Round: testFirstRound, Effect: EffectVerifyHint}, Use{LotNo: 1, Round: 2, Effect: EffectVerifyHint}); err != nil {
		t.Errorf("verified earlier: err = %v", err)
	}
}

func TestNextRescan(t *testing.T) {
	// 4 轮、加轮 1 轮、每 2 轮再扫描一次: 第 3 轮和加轮的第 5 轮
	rule := venue.AuctionRule{Rounds: 4, MaxExtraRounds: 1, RescanEvery: 2, RescanRate: 1500}
	for round, want := range map[int32]int32{1: 3, 2: 3, 3: 5, 4: 5, 5: 0} {
		if got := nextRescan(&rule, round); got != want {
			t.Errorf("round %d: got %d, want %d", round, got, want)
		}
	}
	rule.MaxExtraRounds = 0
	if got := nextRescan(&rule, 3); got != 0 {
		t.Errorf("no extra rounds: got %d", got)
	}
	for name, rule := range map[string]venue.AuctionRule{
		"no rescan":    {Rounds: 4, RescanRate: 1500},
		"zero rate":    {Rounds: 4, RescanEvery: 2},
		"single round": {Rounds: 1, RescanEvery: 1, RescanRate: 1500},
	} {
		if got := nextRescan(&rule, 1); got != 0 {
			t.Errorf("%s: got %d", name, got)
		}
	}
}

func TestForeseeScan(t *testing.T) {
	fact := func(text string) []Fact { return []Fact{{Kind: FactText, Label: "下一次扫描", Text: text}} }

	// 第 1 轮预知第 3 轮: 第 6 件, 只有轮廓
	got, err := foreseeScan(testContext(1), &ForeseeScanParams{})
	if want := (Intel{Pieces: []PieceIntel{intelAt(5, 0, "")}, Facts: fact("第 3 轮")}); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("round 1: got %+v %v, want %+v", got, err, want)
	}
	// 自己已经知道的轮廓不重复给
	seen := Use{Effect: EffectRevealOutline, Result: Intel{Pieces: []PieceIntel{intelAt(5, 0, "")}}}
	got, err = foreseeScan(testContext(2, seen), &ForeseeScanParams{})
	if want := (Intel{Facts: fact("第 3 轮, 不会扫出新的轮廓")}); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("known: got %+v %v, want %+v", got, err, want)
	}
	// 第 3 轮之后下一次在加轮, 没有预先抽到的藏品
	got, err = foreseeScan(testContext(3), &ForeseeScanParams{})
	if want := (Intel{Facts: fact("第 5 轮(加轮), 不会扫出新的轮廓")}); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("round 3: got %+v %v, want %+v", got, err, want)
	}

	// 之后不再扫描时不能用
	if _, err := foreseeScan(testContext(5), &ForeseeScanParams{}); !unusable(err, UseNoRescan) {
		t.Errorf("last round: err = %v", err)
	}
	scene := testScene(1)
	scene.Rule.RescanEvery = 0
	if _, err := foreseeScan(newUseContext(scene, first), &ForeseeScanParams{}); !unusable(err, UseNoRescan) {
		t.Errorf("no rescan: err = %v", err)
	}
	// 同一次扫描只预知一次, 下一次扫描可以再用
	foreseen := Use{Effect: EffectForeseeScan, Result: Intel{Pieces: []PieceIntel{intelAt(5, 0, "")}}}
	if _, err := foreseeScan(testContext(2, foreseen), &ForeseeScanParams{}); !unusable(err, UseScanForeseen) {
		t.Errorf("foreseen: err = %v", err)
	}
	if _, err := foreseeScan(testContext(3, foreseen), &ForeseeScanParams{}); err != nil {
		t.Errorf("next scan: err = %v", err)
	}
}

func TestPeekNext(t *testing.T) {
	next := func(x, y, width, height int16) PieceIntel {
		return PieceIntel{X: x, Y: y, Width: width, Height: height, LotNo: testLotNo + 1}
	}

	// 只有轮廓, 标出是下一个拍品的
	got, err := peekNext(testContext(1), &PeekNextParams{Count: 2})
	if want := (Intel{Pieces: []PieceIntel{next(0, 0, 2, 1), next(2, 0, 1, 1)}}); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v %v, want %+v", got, err, want)
	}
	// 已经看到的跳过, 不够时有几件给几件
	seen := Use{Effect: EffectPeekNext, Result: got}
	got, _ = peekNext(testContext(2, seen), &PeekNextParams{Count: 5})
	if want := []PieceIntel{next(0, 1, 1, 2)}; !reflect.DeepEqual(got.Pieces, want) {
		t.Errorf("rest: got %+v, want %+v", got.Pieces, want)
	}
	// 都看到了时说明没有
	got, err = peekNext(testContext(2, seen, Use{Effect: EffectPeekNext, Result: got}), &PeekNextParams{Count: 1})
	if err != nil || len(got.Pieces) != 0 || len(got.Facts) != 1 || got.Facts[0].Kind != FactText {
		t.Errorf("none: got %+v %v", got, err)
	}
	// 这个拍品上同样位置的轮廓不算看过下一个拍品的
	here := Use{Effect: EffectRevealOutline, Result: Intel{Pieces: []PieceIntel{intelAt(0, 0, ""), intelAt(2, 0, "")}}}
	if got, _ := peekNext(testContext(1, here), &PeekNextParams{Count: 5}); len(got.Pieces) != 3 {
		t.Errorf("other lot: got %+v", got)
	}

	// 最后一个拍品不能用
	scene := testScene(1)
	scene.Next, scene.NextItems = nil, nil
	if _, err := peekNext(newUseContext(scene, first), &PeekNextParams{Count: 1}); !unusable(err, UseLastLot) {
		t.Errorf("last lot: err = %v", err)
	}
}

// TestMatchEffects 情报道具都注册在对局场景
func TestMatchEffects(t *testing.T) {
	for _, code := range []string{
		EffectRevealOutline, EffectRevealQuality, EffectCountQuality, EffectRevealName, EffectValueRange,
		EffectCountRange, EffectRevealTop, EffectProbeEmpty, EffectVerifyHint, EffectForeseeScan, EffectPeekNext,
	} {
		if got := effect.SceneOf(code); got != effect.SceneMatch {
			t.Errorf("%s: scene = %q", code, got)
		}
	}
}
