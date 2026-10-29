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
	"reflect"
	"testing"
)

// testPieces 6 件藏品, 扫描顺序 1-6, 公开顺序和品质各不相同
func testPieces() []WarehouseItem {
	return []WarehouseItem{
		{ScanOrder: 1, RevealOrder: 3, Quality: 1, Name: "a"},
		{ScanOrder: 2, RevealOrder: 6, Quality: 4, Name: "b"},
		{ScanOrder: 3, RevealOrder: 1, Quality: 2, Name: "c"},
		{ScanOrder: 4, RevealOrder: 5, Quality: 5, Name: "d"},
		{ScanOrder: 5, RevealOrder: 2, Quality: 4, Name: "e"},
		{ScanOrder: 6, RevealOrder: 4, Quality: 3, Name: "f"},
	}
}

// first 总是返回 0 的随机数
func first(int64) int64 { return 0 }

func TestUseContextKnown(t *testing.T) {
	uses := []Use{{Result: Intel{Pieces: []PieceIntel{{ScanOrder: 2, Quality: 4}, {ScanOrder: 4, Quality: 5, Name: "d"}}}}}
	ctx := newUseContext(2, 6, 100, testPieces(), uses, first)
	want := map[int32]bool{3: true, 5: true, 2: true, 4: true}
	if !reflect.DeepEqual(ctx.Known, want) {
		t.Errorf("known: got %v, want %v", ctx.Known, want)
	}
	if !reflect.DeepEqual(ctx.KnownNames, map[int32]bool{4: true}) {
		t.Errorf("known names: got %v", ctx.KnownNames)
	}
}

func TestRevealQuality(t *testing.T) {
	// 第 2 轮已公开公开顺序 1、2 (扫描顺序 3、5), 从公开顺序最大的开始选
	ctx := newUseContext(2, 6, 100, testPieces(), nil, first)
	got, _ := revealQuality(ctx, &RevealQualityParams{Count: 2})
	want := []PieceIntel{{ScanOrder: 2, Quality: 4}, {ScanOrder: 4, Quality: 5}}
	if !reflect.DeepEqual(got.Pieces, want) {
		t.Errorf("got %+v, want %+v", got.Pieces, want)
	}

	// 已经看过的跳过, 不够时有几件给几件
	uses := []Use{{Result: got}}
	ctx = newUseContext(2, 6, 100, testPieces(), uses, first)
	got, _ = revealQuality(ctx, &RevealQualityParams{Count: 5})
	want = []PieceIntel{{ScanOrder: 1, Quality: 1}, {ScanOrder: 6, Quality: 3}}
	if !reflect.DeepEqual(got.Pieces, want) {
		t.Errorf("rest: got %+v, want %+v", got.Pieces, want)
	}
}

func TestCountQuality(t *testing.T) {
	ctx := newUseContext(1, 6, 100, testPieces(), nil, first)
	got, _ := countQuality(ctx, &CountQualityParams{MinQuality: 4})
	want := Intel{MinQuality: 4, Counts: []QualityCount{{Quality: 5, Count: 1}, {Quality: 4, Count: 2}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	got, _ = countQuality(ctx, &CountQualityParams{MinQuality: 6})
	if len(got.Counts) != 0 || got.Counts == nil {
		t.Errorf("none: got %+v", got.Counts)
	}
}

func TestRevealName(t *testing.T) {
	ctx := newUseContext(1, 6, 100, testPieces(), nil, first)
	got, _ := revealName(ctx, &RevealNameParams{Count: 2, MinQuality: 4})
	want := []PieceIntel{{ScanOrder: 2, Quality: 4, Name: "b"}, {ScanOrder: 4, Quality: 5, Name: "d"}}
	if !reflect.DeepEqual(got.Pieces, want) {
		t.Errorf("got %+v, want %+v", got.Pieces, want)
	}

	// 已知名称的跳过; 没有符合条件的藏品时为空
	ctx = newUseContext(1, 6, 100, testPieces(), []Use{{Result: got}}, first)
	got, _ = revealName(ctx, &RevealNameParams{Count: 2, MinQuality: 4})
	if want := []PieceIntel{{ScanOrder: 5, Quality: 4, Name: "e"}}; !reflect.DeepEqual(got.Pieces, want) {
		t.Errorf("rest: got %+v, want %+v", got.Pieces, want)
	}
	got, _ = revealName(ctx, &RevealNameParams{Count: 1, MinQuality: 6})
	if len(got.Pieces) != 0 || got.MinQuality != 6 {
		t.Errorf("none: got %+v", got)
	}
}

func TestValueRange(t *testing.T) {
	cases := []struct {
		name     string
		value    int64
		offset   int64
		min, max int64
	}{
		{"value at top", 10000, 4000, 6000, 10000},
		{"value at bottom", 10000, 0, 10000, 14000},
		{"value in middle", 10000, 1500, 8500, 12500},
		{"zero value", 0, 0, 0, 0},
	}
	for _, c := range cases {
		var bound int64
		ctx := &UseContext{Value: c.value, Rand: func(n int64) int64 { bound = n; return c.offset }}
		got, err := valueRange(ctx, &ValueRangeParams{Precision: 2000})
		if err != nil || got.ValueMin != c.min || got.ValueMax != c.max {
			t.Errorf("%s: got %+v %v, want [%d, %d]", c.name, got, err, c.min, c.max)
		}
		if want := c.value*2*2000/10000 + 1; bound != want {
			t.Errorf("%s: rand bound got %d, want %d", c.name, bound, want)
		}
	}
	// 价值较小时下限不低于 0
	ctx := &UseContext{Value: 100, Rand: func(n int64) int64 { return n - 1 }}
	if got, _ := valueRange(ctx, &ValueRangeParams{Precision: 10000}); got.ValueMin != 0 || got.ValueMax != 100 {
		t.Errorf("clamp: got %+v", got)
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
