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
	"cmp"
	"errors"
	"slices"

	"github.com/NuoXianTech/blindcrate/internal/apps/effect"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"gorm.io/gorm"
)

// 对局中可用的道具效果, 都只给使用者自己提供情报
const (
	EffectRevealQuality = "reveal_quality" // 看几件还不知道品质的藏品的品质
	EffectCountQuality  = "count_quality"  // 某品质及以上的藏品各有几件
	EffectRevealName    = "reveal_name"    // 看几件某品质及以上藏品的名称
	EffectValueRange    = "value_range"    // 仓库价值的区间
)

// registerEffects 注册对局场景的道具效果, 在 init 中调用
func registerEffects() {
	effect.Register(EffectRevealQuality, effect.SceneMatch, "鉴定品质",
		"查看几件还不知道品质的藏品的品质, 优先选之后几轮最晚公开的", revealQuality)
	effect.Register(EffectCountQuality, effect.SceneMatch, "品质清单",
		"知道某品质及以上的藏品各有几件", countQuality)
	effect.Register(EffectRevealName, effect.SceneMatch, "透视名称",
		"随机查看几件某品质及以上藏品的名称和品质, 没有时提示没有", revealName)
	effect.Register(EffectValueRange, effect.SceneMatch, "估价区间",
		"给出仓库价值的区间, 区间的位置随机, 真实价值不一定在中间", valueRange)
}

// UseContext 对局中使用道具时的上下文, 效果只读不写
type UseContext struct {
	Pieces     []WarehouseItem     // 仓库里的全部藏品, 按扫描顺序
	Value      int64               // 仓库价值
	Known      map[int32]bool      // 使用者已知品质的藏品, 按扫描顺序; 包括已公开的和之前用道具看到的
	KnownNames map[int32]bool      // 使用者已知名称的藏品, 按扫描顺序
	Rand       func(n int64) int64 // 返回 [0, n) 的随机数, 测试时可以替换
}

// newUseContext 以座位 seat 的视角生成上下文; uses 为这个座位之前用过的道具
func newUseContext(round, pieceCount int32, value int64, items []WarehouseItem, uses []Use, random func(n int64) int64) *UseContext {
	ctx := &UseContext{Pieces: items, Value: value, Known: map[int32]bool{}, KnownNames: map[int32]bool{}, Rand: random}
	revealed := min(round, pieceCount)
	for _, it := range items {
		if it.RevealOrder <= revealed {
			ctx.Known[it.ScanOrder] = true
		}
	}
	for _, use := range uses {
		for _, piece := range use.Result.Pieces {
			ctx.Known[piece.ScanOrder] = true
			if piece.Name != "" {
				ctx.KnownNames[piece.ScanOrder] = true
			}
		}
	}
	return ctx
}

// checkQuality 品质不存在时返回原因
func checkQuality(tx *gorm.DB, level int16) (string, error) {
	if err := item.CheckQuality(tx, level); err != nil {
		if err.Error() == item.QualityNotFound {
			return err.Error(), nil
		}
		return "", err
	}
	return "", nil
}

// RevealQualityParams 鉴定品质的参数
type RevealQualityParams struct {
	Count int32 `json:"count" label:"件数" binding:"min=1,max=10"`
}

// revealQuality 按公开顺序从后往前选还不知道品质的藏品, 避开之后几轮马上会公开的
func revealQuality(ctx *UseContext, params *RevealQualityParams) (Intel, error) {
	var unknown []WarehouseItem
	for _, it := range ctx.Pieces {
		if !ctx.Known[it.ScanOrder] {
			unknown = append(unknown, it)
		}
	}
	slices.SortFunc(unknown, func(a, b WarehouseItem) int { return cmp.Compare(b.RevealOrder, a.RevealOrder) })
	unknown = unknown[:min(len(unknown), int(params.Count))]
	intel := Intel{Pieces: make([]PieceIntel, len(unknown))}
	for i, it := range unknown {
		intel.Pieces[i] = PieceIntel{ScanOrder: it.ScanOrder, Quality: it.Quality}
	}
	sortPieces(intel.Pieces)
	return intel, nil
}

// CountQualityParams 品质清单的参数
type CountQualityParams struct {
	MinQuality int16 `json:"min_quality" label:"最低品质" input:"quality" binding:"min=1"`
}

// Validate 品质需要存在
func (p *CountQualityParams) Validate(tx *gorm.DB) (string, error) {
	return checkQuality(tx, p.MinQuality)
}

// countQuality 统计最低品质及以上各品质的件数
func countQuality(ctx *UseContext, params *CountQualityParams) (Intel, error) {
	counts := map[int16]int32{}
	for _, it := range ctx.Pieces {
		if it.Quality >= params.MinQuality {
			counts[it.Quality]++
		}
	}
	intel := Intel{MinQuality: params.MinQuality, Counts: []QualityCount{}}
	for quality, count := range counts {
		intel.Counts = append(intel.Counts, QualityCount{Quality: quality, Count: count})
	}
	slices.SortFunc(intel.Counts, func(a, b QualityCount) int { return cmp.Compare(b.Quality, a.Quality) })
	return intel, nil
}

// RevealNameParams 透视名称的参数
type RevealNameParams struct {
	Count      int32 `json:"count" label:"件数" binding:"min=1,max=5"`
	MinQuality int16 `json:"min_quality" label:"最低品质" input:"quality" binding:"min=1"`
}

// Validate 品质需要存在
func (p *RevealNameParams) Validate(tx *gorm.DB) (string, error) {
	return checkQuality(tx, p.MinQuality)
}

// revealName 从最低品质及以上、还不知道名称的藏品中随机选几件
func revealName(ctx *UseContext, params *RevealNameParams) (Intel, error) {
	var candidates []WarehouseItem
	for _, it := range ctx.Pieces {
		if it.Quality >= params.MinQuality && !ctx.KnownNames[it.ScanOrder] {
			candidates = append(candidates, it)
		}
	}
	n := min(len(candidates), int(params.Count))
	for i := range n {
		j := i + int(ctx.Rand(int64(len(candidates)-i)))
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}
	intel := Intel{MinQuality: params.MinQuality, Pieces: make([]PieceIntel, n)}
	for i, it := range candidates[:n] {
		intel.Pieces[i] = PieceIntel{ScanOrder: it.ScanOrder, Quality: it.Quality, Name: it.Name, ImageURL: it.ImageURL}
	}
	sortPieces(intel.Pieces)
	return intel, nil
}

// ValueRangeParams 估价区间的参数
type ValueRangeParams struct {
	Precision int32 `json:"precision" label:"误差" help:"区间宽度为仓库价值 × 误差 × 2, 基点, 2000 表示 ±20%" binding:"min=100,max=10000"`
}

// valueRange 区间宽度为价值 × 误差 × 2, 真实价值在区间中的位置均匀随机, 下限不低于 0
//
// 随机数不能由对局 ID 等可以推算的值决定, 否则知道算法的人可以从区间反推出真实价值。
func valueRange(ctx *UseContext, params *ValueRangeParams) (Intel, error) {
	if ctx.Value < 0 {
		return Intel{}, errors.New("auction: negative warehouse value")
	}
	width := ctx.Value * int64(params.Precision) * 2 / venue.BasisPoints
	low := ctx.Value - ctx.Rand(width+1)
	high := low + width
	return Intel{ValueMin: max(low, 0), ValueMax: high}, nil
}

// sortPieces 按扫描顺序排列
func sortPieces(pieces []PieceIntel) {
	slices.SortFunc(pieces, func(a, b PieceIntel) int { return cmp.Compare(a.ScanOrder, b.ScanOrder) })
}
