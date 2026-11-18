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
	"fmt"
	"math"
	"slices"

	"github.com/NuoXianTech/blindcrate/internal/apps/effect"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"gorm.io/gorm"
)

// 对局中可用的情报道具, 都只给使用者自己提供情报; 看到的藏品同时扫出轮廓
const (
	EffectRevealOutline = "reveal_outline" // 看几件还不知道轮廓的藏品的轮廓
	EffectRevealQuality = "reveal_quality" // 看几件还不知道品质的藏品的品质和轮廓
	EffectCountQuality  = "count_quality"  // 某品质及以上的藏品各有几件
	EffectRevealName    = "reveal_name"    // 看几件某品质及以上藏品的名称、外观和轮廓
	EffectValueRange    = "value_range"    // 货仓价值的区间
	EffectCountRange    = "count_range"    // 藏品件数的区间
	EffectRevealTop     = "reveal_top"     // 回收价最高的几件藏品的位置和轮廓
	EffectProbeEmpty    = "probe_empty"    // 几个确定没有藏品的格子
	EffectVerifyHint    = "verify_hint"    // 主持人这一轮的提示是真是假
	EffectForeseeScan   = "foresee_scan"   // 下一次再次扫描会扫出的轮廓
	EffectPeekNext      = "peek_next"      // 下一个拍品里几件藏品的轮廓
)

// registerEffects 注册对局场景的道具效果, 在 init 中调用
func registerEffects() {
	effect.Register(EffectRevealOutline, effect.SceneMatch, "扫描货仓",
		"随机扫出几件还不知道轮廓的藏品的轮廓, 不含品质和名称", revealOutline)
	effect.Register(EffectRevealQuality, effect.SceneMatch, "鉴定品质",
		"随机扫出几件还不知道品质的藏品的品质和轮廓", revealQuality)
	effect.Register(EffectCountQuality, effect.SceneMatch, "品质清单",
		"知道某品质及以上的藏品各有几件", countQuality)
	effect.Register(EffectRevealName, effect.SceneMatch, "透视名称",
		"随机扫出几件某品质及以上藏品的名称、外观、品质和轮廓, 没有时提示没有", revealName)
	effect.Register(EffectValueRange, effect.SceneMatch, "估价区间",
		"给出货仓价值的区间, 区间的位置随机, 真实价值不一定在中间", valueRange)
	effect.Register(EffectCountRange, effect.SceneMatch, "件数区间",
		"给出藏品件数的区间, 区间的位置随机, 真实件数不一定在中间", countRange)
	effect.Register(EffectRevealTop, effect.SceneMatch, "锁定大货",
		"标出回收价最高的几件藏品的位置和轮廓, 不含品质、名称和价格", revealTop)
	effect.Register(EffectProbeEmpty, effect.SceneMatch, "空位探测",
		"随机找出几个确定没有藏品的格子", probeEmpty)
	effect.Register(EffectVerifyHint, effect.SceneMatch, "验证提示",
		"知道主持人这一轮的提示是真是假; 这一轮没有提示时不能使用, 也不扣除", verifyHint)
	effect.Register(EffectForeseeScan, effect.SceneMatch, "预知扫描",
		"提前看到下一次再次扫描会扫出的轮廓; 之后不再扫描时不能使用, 也不扣除", foreseeScan)
	effect.Register(EffectPeekNext, effect.SceneMatch, "探查下一个货仓",
		"随机扫出下一个拍品里几件藏品的轮廓, 不含品质和名称; 最后一个拍品时不能使用, 也不扣除", peekNext)
}

// labelAtLeast 条目标签: 某品质及以上, 品质由 Fact.Quality 给出
const labelAtLeast = "及以上"

// topNotes 回收价最高的几件藏品上的标注, 按名次
var topNotes = [...]string{"最贵", "第 2 贵", "第 3 贵"}

// cell 一个格子的位置; 藏品按它左上角的格子区分, 一个拍品中不重复
type cell struct{ x, y int16 }

// PastUse 使用者之前用过的一个道具, 效果用它判断现在是不是重复使用
type PastUse struct {
	Effect string
	LotNo  int32 // 使用时的拍品
	Round  int32 // 使用时是当前拍品的第几轮, 用在其他拍品上的为 0
}

// NextLot 使用道具时还没开拍的下一个拍品
type NextLot struct {
	LotNo    int32
	Pieces   []DepotItem   // 全部藏品, 按扫描顺序
	Outlined map[cell]bool // 使用者已知轮廓的藏品, 按位置; 只有之前用道具看到的
}

// UseContext 对局中使用道具时的上下文, 效果只读不写
//
// 现在不能使用时(如这一轮没有提示)效果返回 effect.Unusable 的错误, 原因展示给玩家, 道具不扣除;
// 不能因为货仓里的内容而拒绝, 否则不花道具就能知道结果: 没有可给的情报时照常扣除并说明没有。
type UseContext struct {
	LotNo         int32               // 当前拍品的序号
	Round         int32               // 当前拍品的第几轮
	Rule          venue.AuctionRule   // 当前拍品的拍卖参数, 是公开信息
	Grid          venue.Grid          // 货仓的格子大小
	Pieces        []DepotItem         // 货仓里的全部藏品, 按扫描顺序
	Currency      string              // 出价货币, 回收货币不同的藏品不计入货仓价值
	Value         int64               // 货仓价值
	Expected      int64               // 货仓的期望价值, 按公示概率算出, 见 expectedValue; 是公开信息
	ExpectedCount float64             // 货仓的期望件数, 同上
	Hint          *HostHint           // 主持人这一轮给出的提示, 含是否属实; 没有时为 nil
	Outlined      map[cell]bool       // 使用者已知轮廓的藏品, 按位置; 到这一轮为止扫出的和之前用道具看到的
	Known         map[cell]bool       // 使用者已知品质的藏品, 按位置; 品质从不公开, 只有之前用道具看到的
	KnownNames    map[cell]bool       // 使用者已知名称的藏品, 按位置
	Notes         map[cell]string     // 使用者之前看到的藏品上的标注, 按位置, 如 "最贵"
	Empties       map[cell]bool       // 使用者已知没有藏品的格子
	Next          *NextLot            // 下一个拍品, 当前是最后一个时为 nil
	Past          []PastUse           // 使用者在这一局之前用过的道具, 按使用顺序
	Rand          func(n int64) int64 // 返回 [0, n) 的随机数, 测试时可以替换
}

// useScene 生成 UseContext 用的局面
type useScene struct {
	Lot       *Lot              // 当前拍品
	Rule      venue.AuctionRule // 当前拍品的拍卖参数
	Currency  string            // 出价货币
	Round     int32             // 仓库内的当前轮次
	Items     []DepotItem       // 当前拍品的藏品, 按扫描顺序
	Hints     []HostHint        // 当前拍品已经给出的提示
	Next      *Lot              // 下一个拍品, 没有时为 nil
	NextItems []DepotItem       // 下一个拍品的藏品, 按扫描顺序
	Uses      []Use             // 使用者在这一局用过的全部道具, 按使用顺序
}

// newUseContext 以使用者的视角生成上下文
//
// 之前的道具看到的格子按拍品归属: 格子自己给出拍品时属于那个拍品(如提前看到的下一个拍品), 否则属于使用道具时的拍品。
func newUseContext(scene *useScene, random func(n int64) int64) *UseContext {
	lot := scene.Lot
	ctx := &UseContext{
		LotNo:         lot.LotNo,
		Round:         lot.Round(scene.Round),
		Rule:          scene.Rule,
		Grid:          lot.Grid,
		Pieces:        scene.Items,
		Currency:      scene.Currency,
		Value:         lot.Value,
		Expected:      expectedValue(lot.Odds),
		ExpectedCount: expectedCount(lot.Odds),
		Outlined:      map[cell]bool{},
		Known:         map[cell]bool{},
		KnownNames:    map[cell]bool{},
		Notes:         map[cell]string{},
		Empties:       map[cell]bool{},
		Rand:          random,
	}
	for _, it := range scene.Items {
		if it.Outlined(ctx.Round) {
			ctx.Outlined[cell{it.X, it.Y}] = true
		}
	}
	for i := range scene.Hints {
		if scene.Hints[i].Round == scene.Round {
			ctx.Hint = &scene.Hints[i]
		}
	}
	if scene.Next != nil {
		ctx.Next = &NextLot{LotNo: scene.Next.LotNo, Pieces: scene.NextItems, Outlined: map[cell]bool{}}
	}
	for _, use := range scene.Uses {
		past := PastUse{Effect: use.Effect, LotNo: use.LotNo}
		if use.LotNo == lot.LotNo {
			past.Round = lot.Round(use.Round)
		}
		ctx.Past = append(ctx.Past, past)
		for _, piece := range use.Result.Pieces {
			at := cell{piece.X, piece.Y}
			switch lotNo := cmp.Or(piece.LotNo, use.LotNo); {
			case ctx.Next != nil && lotNo == ctx.Next.LotNo:
				if !piece.Empty {
					ctx.Next.Outlined[at] = true
				}
			case lotNo != lot.LotNo:
			case piece.Empty:
				ctx.Empties[at] = true
			default:
				ctx.Outlined[at] = true
				if piece.Quality > 0 {
					ctx.Known[at] = true
				}
				if piece.Name != "" {
					ctx.KnownNames[at] = true
				}
				if piece.Note != "" {
					ctx.Notes[at] = piece.Note
				}
			}
		}
	}
	return ctx
}

// used 使用者在当前拍品上是否用过效果为 code 且满足 match 的道具
func (ctx *UseContext) used(code string, match func(round int32) bool) bool {
	return slices.ContainsFunc(ctx.Past, func(u PastUse) bool {
		return u.Effect == code && u.LotNo == ctx.LotNo && match(u.Round)
	})
}

// price 一件藏品计入货仓价值的回收价, 回收货币不是出价货币的为 0
func (ctx *UseContext) price(it *DepotItem) int64 {
	if it.SellCurrency != ctx.Currency {
		return 0
	}
	return it.SellPrice
}

// expectedCount 按公示概率算出的货仓期望件数
func expectedCount(odds []venue.PieceOdds) float64 {
	var total float64
	for _, o := range odds {
		total += o.Count
	}
	return total
}

// outline 一件藏品的轮廓: 位置和大小, 不含品质和名称
func outline(it *DepotItem) PieceIntel {
	return PieceIntel{X: it.X, Y: it.Y, Width: it.Width, Height: it.Height}
}

// pieceIntel 看到的一件藏品, 带品质; withName 为是否带名称和外观
func pieceIntel(it *DepotItem, withName bool) PieceIntel {
	intel := outline(it)
	intel.Quality = it.Quality
	if withName {
		intel.Name, intel.ImageURL = it.Name, it.ImageURL
	}
	return intel
}

// pick 从 candidates 中随机选 n 个, 不够时全选; 会打乱 candidates
func pick[T any](ctx *UseContext, candidates []T, n int) []T {
	n = min(len(candidates), n)
	for i := range n {
		j := i + int(ctx.Rand(int64(len(candidates)-i)))
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}
	return candidates[:n]
}

// textFact 只有一句话的情报
func textFact(label, text string) Intel {
	return Intel{Facts: []Fact{{Kind: FactText, Label: label, Text: text}}}
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

// RevealOutlineParams 扫描货仓的参数
type RevealOutlineParams struct {
	Count int32 `json:"count" label:"件数" binding:"min=1,max=10"`
}

// revealOutline 从还不知道轮廓的藏品中随机选几件, 只给位置和大小
func revealOutline(ctx *UseContext, params *RevealOutlineParams) (Intel, error) {
	var unknown []DepotItem
	for _, it := range ctx.Pieces {
		if !ctx.Outlined[cell{it.X, it.Y}] {
			unknown = append(unknown, it)
		}
	}
	picked := pick(ctx, unknown, int(params.Count))
	intel := Intel{Pieces: make([]PieceIntel, len(picked))}
	for i := range picked {
		intel.Pieces[i] = outline(&picked[i])
	}
	sortPieces(intel.Pieces)
	return intel, nil
}

// RevealQualityParams 鉴定品质的参数
type RevealQualityParams struct {
	Count int32 `json:"count" label:"件数" binding:"min=1,max=10"`
}

// revealQuality 从还不知道品质的藏品中随机选几件
func revealQuality(ctx *UseContext, params *RevealQualityParams) (Intel, error) {
	var unknown []DepotItem
	for _, it := range ctx.Pieces {
		if !ctx.Known[cell{it.X, it.Y}] {
			unknown = append(unknown, it)
		}
	}
	picked := pick(ctx, unknown, int(params.Count))
	intel := Intel{Pieces: make([]PieceIntel, len(picked))}
	for i := range picked {
		intel.Pieces[i] = pieceIntel(&picked[i], false)
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
	fact := Fact{Kind: FactCounts, Label: labelAtLeast, Quality: params.MinQuality, Counts: []QualityCount{}}
	for quality, count := range counts {
		fact.Counts = append(fact.Counts, QualityCount{Quality: quality, Count: count})
	}
	slices.SortFunc(fact.Counts, func(a, b QualityCount) int { return cmp.Compare(b.Quality, a.Quality) })
	return Intel{Facts: []Fact{fact}}, nil
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
	var candidates []DepotItem
	for _, it := range ctx.Pieces {
		if it.Quality >= params.MinQuality && !ctx.KnownNames[cell{it.X, it.Y}] {
			candidates = append(candidates, it)
		}
	}
	picked := pick(ctx, candidates, int(params.Count))
	if len(picked) == 0 {
		return Intel{Facts: []Fact{{Kind: FactText, Label: labelAtLeast, Quality: params.MinQuality, Text: "没有找到"}}}, nil
	}
	intel := Intel{Pieces: make([]PieceIntel, len(picked))}
	for i := range picked {
		intel.Pieces[i] = pieceIntel(&picked[i], true)
	}
	sortPieces(intel.Pieces)
	return intel, nil
}

// ValueRangeParams 估价区间的参数
type ValueRangeParams struct {
	Precision int32 `json:"precision" label:"误差" help:"区间宽度为货仓的期望价值 × 误差 × 2, 与货仓实际值多少无关; 20% 表示期望价值的 ±20%" unit:"bp" binding:"min=100,max=10000"`
}

// valueRange 区间宽度为期望价值 × 误差 × 2, 真实价值在区间中的位置均匀随机, 下限不低于 0
//
// 宽度只能由公开的期望价值决定: 按真实价值算时, 宽度 ÷ (误差 × 2) 就是真实价值, 下限也总是价值的固定比例以上, 压着下限出价稳赚。
// 位置均匀随机时, 区间除了"价值在里面"不透露别的; 随机数不能由对局 ID 等可以推算的值决定, 否则知道算法的人可以从区间反推出真实价值。
func valueRange(ctx *UseContext, params *ValueRangeParams) (Intel, error) {
	if ctx.Value < 0 || ctx.Expected < 0 {
		return Intel{}, errors.New("auction: negative depot value")
	}
	// 宽度为 0 的区间就是真实价值
	if ctx.Expected == 0 && ctx.Value > 0 {
		return Intel{}, errors.New("auction: depot has a value but no expected value")
	}
	width := ctx.Expected * int64(params.Precision) * 2 / venue.BasisPoints
	low := ctx.Value - ctx.Rand(width+1)
	high := low + width
	return Intel{Facts: []Fact{{Kind: FactAmount, Label: "货仓价值", Min: max(low, 0), Max: high}}}, nil
}

// CountRangeParams 件数区间的参数
type CountRangeParams struct {
	Precision int32 `json:"precision" label:"误差" help:"区间宽度为货仓的期望件数 × 误差 × 2(至少相差 1 件), 与货仓实际有几件无关; 20% 表示期望件数的 ±20%" unit:"bp" binding:"min=100,max=10000"`
}

// countRange 区间宽度为期望件数 × 误差 × 2, 真实件数在区间中的位置均匀随机; 下限低于 0 时整个区间上移
//
// 宽度按公开的期望件数算, 道理同 valueRange; 至少相差 1 件, 否则就是直接说出件数。
// 下限不能像估价区间那样截到 0: 截断后区间变窄, 件数少的货仓会被直接说出有几件。
func countRange(ctx *UseContext, params *CountRangeParams) (Intel, error) {
	span := max(int64(math.Round(ctx.ExpectedCount*float64(params.Precision)*2/venue.BasisPoints)), 1)
	low := max(int64(len(ctx.Pieces))-ctx.Rand(span+1), 0)
	return Intel{Facts: []Fact{{Kind: FactRange, Label: "藏品件数", Min: low, Max: low + span, Unit: "件"}}}, nil
}

// RevealTopParams 锁定大货的参数
type RevealTopParams struct {
	Count int32 `json:"count" label:"件数" help:"标出回收价最高的几件, 按第几贵标注" binding:"min=1,max=3"`
}

// revealTop 标出回收价最高的几件藏品: 只有位置、轮廓和第几贵, 不含品质、名称和价格; 一样贵时按扫描顺序
//
// 这几件之前都标过时不能再用; 货仓是空的时照常扣除并说明没有, 不能拒绝, 否则不花道具就知道货仓是空的。
func revealTop(ctx *UseContext, params *RevealTopParams) (Intel, error) {
	if len(ctx.Pieces) == 0 {
		return textFact("最贵的藏品", "没有找到"), nil
	}
	ranked := slices.Clone(ctx.Pieces)
	slices.SortStableFunc(ranked, func(a, b DepotItem) int { return cmp.Compare(ctx.price(&b), ctx.price(&a)) })
	ranked = ranked[:min(len(ranked), int(params.Count), len(topNotes))]

	fresh := false
	intel := Intel{Pieces: make([]PieceIntel, len(ranked))}
	for i := range ranked {
		intel.Pieces[i] = outline(&ranked[i])
		intel.Pieces[i].Note = topNotes[i]
		if ctx.Notes[cell{ranked[i].X, ranked[i].Y}] != topNotes[i] {
			fresh = true
		}
	}
	if !fresh {
		return Intel{}, effect.Unusable(UseTopKnown)
	}
	sortPieces(intel.Pieces)
	return intel, nil
}

// ProbeEmptyParams 空位探测的参数
type ProbeEmptyParams struct {
	Count int32 `json:"count" label:"格数" binding:"min=1,max=32"`
}

// probeEmpty 从没有藏品、使用者也还不知道是空的格子中随机选几格; 没有这样的格子时说明没有
func probeEmpty(ctx *UseContext, params *ProbeEmptyParams) (Intel, error) {
	occupied := map[cell]bool{}
	for _, it := range ctx.Pieces {
		for dy := range it.Height {
			for dx := range it.Width {
				occupied[cell{it.X + dx, it.Y + dy}] = true
			}
		}
	}
	var free []cell
	for y := range ctx.Grid.Height {
		for x := range ctx.Grid.Width {
			if at := (cell{x, y}); !occupied[at] && !ctx.Empties[at] {
				free = append(free, at)
			}
		}
	}
	picked := pick(ctx, free, int(params.Count))
	if len(picked) == 0 {
		return textFact("空位", "没有找到新的空格子"), nil
	}
	intel := Intel{Pieces: make([]PieceIntel, len(picked))}
	for i, at := range picked {
		intel.Pieces[i] = PieceIntel{X: at.x, Y: at.y, Width: 1, Height: 1, Empty: true}
	}
	sortPieces(intel.Pieces)
	return intel, nil
}

// VerifyHintParams 验证提示没有参数
type VerifyHintParams struct{}

// verifyHint 主持人这一轮的提示是真是假; 这一轮没有提示(如加轮、没有主持人的地图)或已经验证过时不能用
func verifyHint(ctx *UseContext, _ *VerifyHintParams) (Intel, error) {
	switch {
	case ctx.Hint == nil:
		return Intel{}, effect.Unusable(UseNoHint)
	case ctx.used(EffectVerifyHint, func(round int32) bool { return round == ctx.Round }):
		return Intel{}, effect.Unusable(UseHintVerified)
	case ctx.Hint.Truthful:
		return textFact("这一轮的提示", "属实"), nil
	}
	return textFact("这一轮的提示", "说错了"), nil
}

// ForeseeScanParams 预知扫描没有参数
type ForeseeScanParams struct{}

// nextRescan 拍品的第 round 轮之后, 下一次再次扫描在第几轮; 之后不再扫描时为 0。预先抽好的加轮扫描也算, 见 outlineRounds
func nextRescan(rule *venue.AuctionRule, round int32) int32 {
	if rule.RescanEvery <= 0 || rule.RescanRate <= 0 {
		return 0
	}
	next := 1 + (rule.Rescans(round)+1)*rule.RescanEvery
	if next > rule.Rounds+rule.MaxExtraRounds {
		return 0
	}
	return next
}

// foreseeScan 下一次再次扫描会扫出的轮廓和它在第几轮; 自己已经知道的轮廓不重复给, 一件也没有时说明没有
//
// 之后不再扫描、或这一次扫描已经预知过时不能用; 这两样只看规则和自己用过的道具, 与货仓里的内容无关。
func foreseeScan(ctx *UseContext, _ *ForeseeScanParams) (Intel, error) {
	next := nextRescan(&ctx.Rule, ctx.Round)
	switch {
	case next == 0:
		return Intel{}, effect.Unusable(UseNoRescan)
	case ctx.used(EffectForeseeScan, func(round int32) bool { return nextRescan(&ctx.Rule, round) == next }):
		return Intel{}, effect.Unusable(UseScanForeseen)
	}
	when := fmt.Sprintf("第 %d 轮", next)
	if next > ctx.Rule.Rounds {
		when += "(加轮)"
	}
	intel := Intel{}
	for i := range ctx.Pieces {
		if it := &ctx.Pieces[i]; it.OutlineRound == next && !ctx.Outlined[cell{it.X, it.Y}] {
			intel.Pieces = append(intel.Pieces, outline(it))
		}
	}
	if len(intel.Pieces) == 0 {
		when += ", 不会扫出新的轮廓"
	}
	intel.Facts = []Fact{{Kind: FactText, Label: "下一次扫描", Text: when}}
	sortPieces(intel.Pieces)
	return intel, nil
}

// PeekNextParams 探查下一个货仓的参数
type PeekNextParams struct {
	Count int32 `json:"count" label:"件数" binding:"min=1,max=10"`
}

// peekNext 从下一个拍品里还不知道轮廓的藏品中随机选几件, 只给位置和大小; 当前是最后一个拍品时不能用
func peekNext(ctx *UseContext, params *PeekNextParams) (Intel, error) {
	if ctx.Next == nil {
		return Intel{}, effect.Unusable(UseLastLot)
	}
	var unknown []DepotItem
	for _, it := range ctx.Next.Pieces {
		if !ctx.Next.Outlined[cell{it.X, it.Y}] {
			unknown = append(unknown, it)
		}
	}
	picked := pick(ctx, unknown, int(params.Count))
	if len(picked) == 0 {
		return textFact("下一个货仓", "没有找到新的轮廓"), nil
	}
	intel := Intel{Pieces: make([]PieceIntel, len(picked))}
	for i := range picked {
		intel.Pieces[i] = outline(&picked[i])
		intel.Pieces[i].LotNo = ctx.Next.LotNo
	}
	sortPieces(intel.Pieces)
	return intel, nil
}

// sortPieces 按位置从上到下、从左到右排列, 与扫描顺序相同
func sortPieces(pieces []PieceIntel) {
	slices.SortFunc(pieces, func(a, b PieceIntel) int { return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X)) })
}
