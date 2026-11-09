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
	"cmp"
	"errors"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"gorm.io/gorm"
)

// slotPool 一个档位的掉落池, 生成时在内存中抽取
type slotPool struct {
	slot    Slot
	entries []loot.EntryDetail
	weights []int64
}

// Generator 按地图的仓库规则生成仓库; 掉落池和物品在创建时读取, 之后生成不再查库
//
// 不是并发安全的, 每次开局或模拟各自创建。
type Generator struct {
	rule     WarehouseRule
	currency string
	slots    []slotPool
	items    map[uint64]*item.Item
	randN    func(n int64) int64 // 返回 [0, n) 内的随机数
}

// NewGenerator 读取仓库规则引用的掉落池和物品, 使用 crypto/rand 抽取
//
// 掉落池不存在、已停用、没有可抽取的条目, 有货币条目或不是藏品的物品时返回 *InvalidError。
func NewGenerator(tx *gorm.DB, rule WarehouseRule, bidCurrency string) (*Generator, error) {
	g := &Generator{rule: rule, currency: bidCurrency, slots: make([]slotPool, len(rule.Slots)), randN: loot.RandInt64N}
	var itemIDs []uint64
	for i, slot := range rule.Slots {
		_, entries, err := loot.QueryAvailable(tx, slot.Pool)
		if err != nil {
			switch err.Error() {
			case loot.PoolNotFound, loot.PoolDisabled, loot.PoolEmpty:
				return nil, invalid(SlotPoolInvalid, i+1, err.Error())
			}
			return nil, err
		}
		weights := make([]int64, len(entries))
		for j, entry := range entries {
			if entry.ItemID == 0 {
				return nil, invalid(SlotPoolInvalid, i+1, SlotPoolHasMoney)
			}
			weights[j] = int64(entry.Weight)
			itemIDs = append(itemIDs, entry.ItemID)
		}
		g.slots[i] = slotPool{slot: slot, entries: entries, weights: weights}
	}
	items, err := item.Resolve(tx, itemIDs)
	if err != nil {
		return nil, err
	}
	// 仓库只抽藏品, 道具等其他类别只能通过奖励和商城获得
	for i, sp := range g.slots {
		for _, entry := range sp.entries {
			if target := items[entry.ItemID]; target.Category != item.CategoryCollectible {
				return nil, invalid(SlotNotCollectible, i+1, target.Name)
			}
		}
	}
	g.items = items
	return g, nil
}

// Generate 抽取藏品并摆进格子, 摆不下时整仓重抽, 超过重抽次数上限时返回 WarehouseNotFit
//
// valueFactor 为基点, 按它缩放各档位的抽取次数, 小数部分按概率进位; 10000 表示不缩放。
// 配置了数量波动时先抽一档, 再乘上它的系数, 重抽时不换档。
func (g *Generator) Generate(valueFactor int32) (*Warehouse, error) {
	factor := int32(int64(valueFactor) * g.swing() / BasisPoints)
	for range g.rule.MaxRerolls + 1 {
		pieces := g.draw(factor)
		if place(g.rule.Grid, pieces, g.randN) {
			warehouse := &Warehouse{Pieces: pieces}
			for _, piece := range pieces {
				if piece.SellCurrency == g.currency {
					warehouse.Value += piece.SellPrice
				}
			}
			return warehouse, nil
		}
	}
	return nil, errors.New(WarehouseNotFit)
}

// draw 按各档位抽取藏品, 还没有坐标
func (g *Generator) draw(valueFactor int32) []Piece {
	area := int64(g.rule.Grid.Width) * int64(g.rule.Grid.Height)
	var (
		pieces   []Piece
		occupied int64
	)
	for _, sp := range g.slots {
		if g.randN(BasisPoints) >= int64(sp.slot.Chance) {
			continue
		}
		times := int64(sp.slot.Min) + g.randN(int64(sp.slot.Max-sp.slot.Min)+1)
		scaled := times * int64(valueFactor)
		times = scaled / BasisPoints
		if g.randN(BasisPoints) < scaled%BasisPoints {
			times++
		}
		for range times {
			entry := &sp.entries[g.pick(sp.weights)]
			quantity := entry.MinQuantity + g.randN(entry.MaxQuantity-entry.MinQuantity+1)
			target := g.items[entry.ItemID]
			for range quantity {
				pieces = append(pieces, Piece{
					ItemID:       target.ID,
					Quality:      target.Quality,
					Width:        target.Width,
					Height:       target.Height,
					SellCurrency: target.SellCurrency,
					SellPrice:    target.SellPrice,
				})
				// 面积已经超出格子时一定摆不下, 不用再抽
				if occupied += int64(target.Width) * int64(target.Height); occupied > area {
					return pieces
				}
			}
		}
	}
	return pieces
}

// swing 按权重抽一档数量波动, 返回它的系数; 没有配置时为 BasisPoints
func (g *Generator) swing() int64 {
	if len(g.rule.Swings) == 0 {
		return BasisPoints
	}
	weights := make([]int64, len(g.rule.Swings))
	for i, s := range g.rule.Swings {
		weights[i] = int64(s.Weight)
	}
	return int64(g.rule.Swings[g.pick(weights)].Factor)
}

// meanSwing 数量波动的平均系数, 基点; 没有配置时为 BasisPoints
func (g *Generator) meanSwing() float64 {
	var total, weighted float64
	for _, s := range g.rule.Swings {
		total += float64(s.Weight)
		weighted += float64(s.Weight) * float64(s.Factor)
	}
	if total == 0 {
		return BasisPoints
	}
	return weighted / total
}

// pick 按权重随机选出一个下标, 权重都大于 0
func (g *Generator) pick(weights []int64) int {
	var total int64
	for _, weight := range weights {
		total += weight
	}
	n := g.randN(total)
	for i, weight := range weights {
		if n < weight {
			return i
		}
		n -= weight
	}
	return len(weights) - 1
}

// placeAttempts 随机摆放的尝试次数, 都摆不下时改为逐行扫描
const placeAttempts = 8

// place 按面积从大到小摆放, 每件在所有放得下的位置中随机选一个; 随机摆放 placeAttempts 次都摆不下时改为逐行扫描找第一个放得下的位置,
// 摆放成功率不低于逐行扫描; 全部放下时写入坐标并返回 true
//
// 位置要随机: 逐行扫描时藏品都挤在左上角, 空出的行能透露件数。randN 返回 [0, n) 的随机数。
func place(grid Grid, pieces []Piece, randN func(n int64) int64) bool {
	if len(pieces) == 0 {
		return false
	}
	slices.SortStableFunc(pieces, func(a, b Piece) int {
		return cmp.Compare(int32(b.Width)*int32(b.Height), int32(a.Width)*int32(a.Height))
	})
	for range placeAttempts {
		if arrange(grid, pieces, randN) {
			return true
		}
	}
	return arrange(grid, pieces, nil)
}

// arrange 依次摆放每一件: randN 为空时取逐行扫描的第一个放得下的位置, 否则在所有放得下的位置中随机选; 有一件摆不下时返回 false
func arrange(grid Grid, pieces []Piece, randN func(n int64) int64) bool {
	width, height := int(grid.Width), int(grid.Height)
	used := make([]bool, width*height)
	fits := func(x, y, w, h int) bool {
		if x+w > width || y+h > height {
			return false
		}
		for dy := range h {
			for dx := range w {
				if used[(y+dy)*width+x+dx] {
					return false
				}
			}
		}
		return true
	}
	var spots []int // 放得下的位置, y × width + x
	for i := range pieces {
		w, h := int(pieces[i].Width), int(pieces[i].Height)
		spots = spots[:0]
		for y := 0; y < height && (randN != nil || len(spots) == 0); y++ {
			for x := 0; x < width && (randN != nil || len(spots) == 0); x++ {
				if fits(x, y, w, h) {
					spots = append(spots, y*width+x)
				}
			}
		}
		if len(spots) == 0 {
			return false
		}
		spot := spots[0]
		if randN != nil {
			spot = spots[randN(int64(len(spots)))]
		}
		x, y := spot%width, spot/width
		for dy := range h {
			for dx := range w {
				used[(y+dy)*width+x+dx] = true
			}
		}
		pieces[i].X, pieces[i].Y = int16(x), int16(y)
	}
	return true
}

// withRand 换用 randN 抽取的副本, 不影响 g
func (g *Generator) withRand(randN func(n int64) int64) *Generator {
	clone := *g
	clone.randN = randN
	return &clone
}

// Seeded 换用以 seed 为种子的 math/rand 抽取的副本, 用于模拟: 生成快, 相同的种子生成相同的仓库; 不影响 g
func (g *Generator) Seeded(seed uint64) *Generator {
	return g.withRand(rand.New(rand.NewPCG(seed, seed^simulationSalt)).Int64N)
}

// simulationSalt 模拟用的随机数的第二个种子
const simulationSalt = 0x9e3779b97f4a7c15

// Simulate 用固定种子的 math/rand 快速生成 runs 个仓库, 统计摆放失败率和价值分布, 不影响 g 之后的生成
//
// 相同的配置得到相同的结果, 重新保存没改过的地图时推导出的金额不变。
func (g *Generator) Simulate(runs int, valueFactor int32) ValueStats {
	return simulateValues(runs, g.Seeded(1).Generate, valueFactor)
}

// simulateValues 用 generate 生成 runs 个仓库, 统计摆放失败率和价值分布
func simulateValues(runs int, generate func(valueFactor int32) (*Warehouse, error), valueFactor int32) ValueStats {
	values := make([]int64, 0, runs)
	var (
		failures int
		pieces   int
		total    int64
	)
	for range runs {
		warehouse, err := generate(valueFactor)
		if err != nil {
			failures++
			continue
		}
		values = append(values, warehouse.Value)
		pieces += len(warehouse.Pieces)
		total += warehouse.Value
	}

	stats := ValueStats{FailureRate: int32(failures * BasisPoints / runs)}
	if len(values) == 0 {
		return stats
	}
	slices.Sort(values)
	percentile := func(p int) int64 {
		return values[(len(values)-1)*p/100]
	}
	stats.Mean = total / int64(len(values))
	stats.P10, stats.P50, stats.P90 = percentile(10), percentile(50), percentile(90)
	stats.Max = values[len(values)-1]
	stats.MeanPieces = int32(pieces / len(values))
	if total > 0 {
		mean := float64(total) / float64(len(values))
		var squares float64
		for _, v := range values {
			squares += (float64(v) - mean) * (float64(v) - mean)
		}
		stats.CV = int32(math.Round(math.Sqrt(squares/float64(len(values))) / mean * BasisPoints))
	}
	return stats
}

// PieceOdds 按公示概率算出每种藏品在一个仓库中的期望件数和平均回收价, 品质和尺寸相同的合在一起, 按品质、宽、高排列
//
// 只用玩家在地图详情中能看到的档位和掉落池概率, 以及物品的品质、尺寸和回收价; 不考虑摆不下时的重抽。
// valueFactor 同 Generate, 数量波动按平均系数计算。
func (g *Generator) PieceOdds(valueFactor int32) []PieceOdds {
	type key struct{ quality, width, height int16 }
	type sum struct{ count, value float64 }
	sums := map[key]*sum{}
	for _, sp := range g.slots {
		var total int64
		for _, weight := range sp.weights {
			total += weight
		}
		times := float64(sp.slot.Chance) / BasisPoints * float64(sp.slot.Min+sp.slot.Max) / 2 * float64(valueFactor) / BasisPoints * g.meanSwing() / BasisPoints
		for i, entry := range sp.entries {
			target := g.items[entry.ItemID]
			count := times * float64(sp.weights[i]) / float64(total) * float64(entry.MinQuantity+entry.MaxQuantity) / 2
			if count <= 0 {
				continue
			}
			k := key{target.Quality, target.Width, target.Height}
			s, ok := sums[k]
			if !ok {
				s = &sum{}
				sums[k] = s
			}
			s.count += count
			if target.SellCurrency == g.currency {
				s.value += count * float64(target.SellPrice)
			}
		}
	}
	odds := make([]PieceOdds, 0, len(sums))
	for k, s := range sums {
		odds = append(odds, PieceOdds{Quality: k.quality, Width: k.width, Height: k.height, Count: s.count, Price: s.value / s.count})
	}
	slices.SortFunc(odds, func(a, b PieceOdds) int {
		return cmp.Or(cmp.Compare(a.Quality, b.Quality), cmp.Compare(a.Width, b.Width), cmp.Compare(a.Height, b.Height))
	})
	return odds
}

// Lineup 按地图的档次生成拍场的拍品: 拍品数在 LotRule 的范围内均匀随机, 每个拍品按权重抽一种档次
//
// 没有配置档次时只有 Warehouse 一档, 拍品数为 1。与 Generator 相同, 不是并发安全的。
type Lineup struct {
	min, max int32
	tiers    []*Generator
	weights  []int64
	randN    func(n int64) int64
}

// NewLineup 读取各档次引用的掉落池和物品, 使用 crypto/rand 抽取; 错误同 NewGenerator
func NewLineup(tx *gorm.DB, v *Venue) (*Lineup, error) {
	l := &Lineup{randN: loot.RandInt64N}
	l.min, l.max = v.Lots.Range()
	for _, t := range v.LotTiers() {
		g, err := NewGenerator(tx, t.Warehouse, v.BidCurrency)
		if err != nil {
			return nil, err
		}
		l.tiers = append(l.tiers, g)
		l.weights = append(l.weights, int64(t.Weight))
	}
	return l, nil
}

// Count 抽一个拍场的拍品数
func (l *Lineup) Count() int {
	if l.min == l.max {
		return int(l.min)
	}
	return int(l.min + int32(l.randN(int64(l.max-l.min+1))))
}

// Pick 按权重抽一种档次, 返回下标, 同 LotTiers 的顺序
func (l *Lineup) Pick() int {
	if len(l.weights) == 1 {
		return 0
	}
	var total int64
	for _, w := range l.weights {
		total += w
	}
	n := l.randN(total)
	for i, w := range l.weights {
		if n < w {
			return i
		}
		n -= w
	}
	return len(l.weights) - 1
}

// Tier 第 i 种档次的生成器
func (l *Lineup) Tier(i int) *Generator { return l.tiers[i] }

// Tiers 档次的个数
func (l *Lineup) Tiers() int { return len(l.tiers) }

// Seeded 换用以 seed 为种子的 math/rand 抽取的副本, 各档次的生成器也换用这个随机数; 不影响 l
func (l *Lineup) Seeded(seed uint64) *Lineup {
	randN := rand.New(rand.NewPCG(seed, seed^simulationSalt)).Int64N
	clone := *l
	clone.randN = randN
	clone.tiers = make([]*Generator, len(l.tiers))
	for i, g := range l.tiers {
		clone.tiers[i] = g.withRand(randN)
	}
	return &clone
}

// Simulate 模拟 runs 个拍品(按权重混合各档次)的价值分布, 不影响 l 之后的生成
func (l *Lineup) Simulate(runs int, valueFactor int32) ValueStats {
	fast := l.Seeded(1)
	return simulateValues(runs, func(valueFactor int32) (*Warehouse, error) {
		return fast.Tier(fast.Pick()).Generate(valueFactor)
	}, valueFactor)
}
