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
