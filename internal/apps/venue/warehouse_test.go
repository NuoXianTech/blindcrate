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

package venue

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
)

// entry 掉落池中的一个物品条目
func entry(itemID uint64, weight int32, minQuantity, maxQuantity int64) loot.EntryDetail {
	return loot.EntryDetail{Entry: loot.Entry{ItemID: itemID, Weight: weight, MinQuantity: minQuantity, MaxQuantity: maxQuantity}}
}

func TestPieceOdds(t *testing.T) {
	g := &Generator{
		currency: "chip",
		slots: []slotPool{
			// 平均抽 2 次: 物品 1、2、3 各占 1/4、1/4、1/2, 物品 2 每次 1-3 件
			{slot: Slot{Min: 1, Max: 3, Chance: BasisPoints}, entries: []loot.EntryDetail{entry(1, 1, 1, 1), entry(2, 1, 1, 3), entry(3, 2, 1, 1)}, weights: []int64{1, 1, 2}},
			// 一半的概率出现, 物品 4 的回收货币不同, 按 0 计价
			{slot: Slot{Min: 1, Max: 1, Chance: BasisPoints / 2}, entries: []loot.EntryDetail{entry(4, 1, 1, 1)}, weights: []int64{1}},
			// 不抽取的档位不计入
			{slot: Slot{Min: 0, Max: 0, Chance: BasisPoints}, entries: []loot.EntryDetail{entry(5, 1, 1, 1)}, weights: []int64{1}},
		},
		items: map[uint64]*item.Item{
			1: {ID: 1, Quality: 1, Width: 1, Height: 1, SellCurrency: "chip", SellPrice: 40},
			2: {ID: 2, Quality: 1, Width: 1, Height: 1, SellCurrency: "chip", SellPrice: 60},
			3: {ID: 3, Quality: 2, Width: 2, Height: 1, SellCurrency: "chip", SellPrice: 500},
			4: {ID: 4, Quality: 2, Width: 2, Height: 1, SellCurrency: "gold", SellPrice: 999},
			5: {ID: 5, Quality: 3, Width: 1, Height: 1, SellCurrency: "chip", SellPrice: 9000},
		},
	}
	cases := []struct {
		valueFactor int32
		want        []PieceOdds
	}{
		{BasisPoints, []PieceOdds{
			{Quality: 1, Width: 1, Height: 1, Count: 1.5, Price: 80 / 1.5},
			{Quality: 2, Width: 2, Height: 1, Count: 1.5, Price: 500 / 1.5},
		}},
		// 价值系数只缩放件数
		{BasisPoints / 2, []PieceOdds{
			{Quality: 1, Width: 1, Height: 1, Count: 0.75, Price: 80 / 1.5},
			{Quality: 2, Width: 2, Height: 1, Count: 0.75, Price: 500 / 1.5},
		}},
	}
	for _, c := range cases {
		got := g.PieceOdds(c.valueFactor)
		if len(got) != len(c.want) {
			t.Fatalf("value factor %d: got %+v, want %+v", c.valueFactor, got, c.want)
		}
		for i, want := range c.want {
			o := got[i]
			if o.Quality != want.Quality || o.Width != want.Width || o.Height != want.Height ||
				math.Abs(o.Count-want.Count) > 1e-9 || math.Abs(o.Price-want.Price) > 1e-9 {
				t.Errorf("value factor %d: odds %d got %+v, want %+v", c.valueFactor, i, o, want)
			}
		}
	}
}

func TestSwing(t *testing.T) {
	g := &Generator{
		currency: "chip",
		rule: WarehouseRule{
			Grid: Grid{Width: 10, Height: 10},
			// 平均系数 (5000 + 15000 × 3) ÷ 4 = 12500
			Swings: []Swing{{Factor: 5000, Weight: 1}, {Factor: 15000, Weight: 3}},
		},
		slots: []slotPool{
			{slot: Slot{Min: 4, Max: 4, Chance: BasisPoints}, entries: []loot.EntryDetail{entry(1, 1, 1, 1)}, weights: []int64{1}},
		},
		items: map[uint64]*item.Item{1: {ID: 1, Quality: 1, Width: 1, Height: 1, SellCurrency: "chip", SellPrice: 10}},
	}
	// 公示概率按平均系数计算件数
	if odds := g.PieceOdds(BasisPoints); len(odds) != 1 || math.Abs(odds[0].Count-5) > 1e-9 {
		t.Fatalf("PieceOdds got %+v, want count 5", odds)
	}

	// 抽中 5000 时 2 件, 抽中 15000 时 6 件, 重抽时不换档
	counts := map[int]int{}
	for range 400 {
		w, err := g.withRand(rand.Int64N).Generate(BasisPoints)
		if err != nil {
			t.Fatal(err)
		}
		counts[len(w.Pieces)]++
	}
	if len(counts) != 2 || counts[2] == 0 || counts[6] == 0 || counts[6] < counts[2] {
		t.Errorf("piece counts got %v, want only 2 and 6 with 6 about 3 times as often", counts)
	}

	// 没有配置时不缩放
	g.rule.Swings = nil
	if w, err := g.withRand(rand.Int64N).Generate(BasisPoints); err != nil || len(w.Pieces) != 4 {
		t.Errorf("without swings got %v pieces, err %v, want 4", w, err)
	}
}
