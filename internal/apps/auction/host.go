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
	"math"
	"math/rand/v2"
	"slices"

	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"gorm.io/gorm"
)

// hostInput 生成一轮提示时的局面
type hostInput struct {
	Round        int32       // 即将开始的轮次
	Items        []DepotItem // 货仓里的全部藏品
	Value        int64       // 货仓价值
	Expected     int64       // 期望价值, 见 expectedValue
	Qualities    []int16     // 这张地图可能出现的品质, 从低到高; 说错时换成相邻的品质
	CountQuality int16       // "件数"提示统计的最低品质, 见 hintCountQuality; 0 表示不给这种提示
	Rule         *venue.HostRule
	Persona      string
	Hype         int32
	Given        []HostHint // 这个拍品已经给过的提示, 按轮次
}

// makeHint 生成一条提示: 列出这一轮能给的提示, 随机选一种, 再按撒谎率决定是否说错; 没有能给的提示时返回 false
//
// 提示只涉及开局时就确定的信息, 不涉及其他人的出价; 指认一件藏品时只指这一轮已经扫出轮廓的, 不透露其他藏品的位置。
// 件数、最好的品质和价值高低在一个拍品里不会变, 每个拍品各只说一次, 重复说只是让人多一次机会分辨真假; 指认藏品时不指已经指过的。
// 能选别的时不连续出同一种。
func makeHint(in *hostInput, randN func(n int64) int64) (HostHint, bool) {
	said, pointed, last := map[string]bool{}, map[cell]bool{}, ""
	for _, h := range in.Given {
		said[h.Kind], last = true, h.Kind
		if h.Kind == HostHintPiece {
			pointed[cell{h.X, h.Y}] = true
		}
	}
	var fresh []DepotItem
	for _, it := range in.Items {
		if it.Outlined(in.Round) && !pointed[cell{it.X, it.Y}] {
			fresh = append(fresh, it)
		}
	}
	var options []HostHint
	if q := in.CountQuality; q > 0 && !said[HostHintCount] {
		var n int32
		for _, it := range in.Items {
			if it.Quality >= q {
				n++
			}
		}
		options = append(options, HostHint{Kind: HostHintCount, Quality: q, Count: n})
	}
	if len(in.Items) > 0 && !said[HostHintBest] {
		best := slices.MaxFunc(in.Items, func(a, b DepotItem) int { return cmp.Compare(a.Quality, b.Quality) })
		options = append(options, HostHint{Kind: HostHintBest, Quality: best.Quality})
	}
	if len(fresh) > 0 {
		piece := fresh[randN(int64(len(fresh)))]
		options = append(options, HostHint{Kind: HostHintPiece, Quality: piece.Quality, X: piece.X, Y: piece.Y})
	}
	if in.Expected > 0 && !said[HostHintValue] {
		options = append(options, HostHint{Kind: HostHintValue, Above: in.Value > in.Expected})
	}
	if len(options) > 1 {
		options = slices.DeleteFunc(options, func(h HostHint) bool { return h.Kind == last })
	}
	if len(options) == 0 {
		return HostHint{}, false
	}

	hint := options[randN(int64(len(options)))]
	hint.Round, hint.Truthful = in.Round, true
	if randN(venue.BasisPoints) < int64(lieRate(in, &hint)) {
		lie(in, &hint, randN)
	}
	return hint, true
}

// hintCountQuality "件数"提示统计的最低品质: 公示概率下不低于它的藏品的期望件数最接近 1 件的品质, 远近按倍数比, 一样近时取低的;
// 期望件数不在 [hintCountMin, hintCountMax] 内的品质不用, 都不在时返回 0, 不给这种提示
//
// 每个拍品按自己的公示分布选, 不用地图配置的高品质: 期望件数接近 0 的品质说出来几乎总是 0 件, 说了等于没说,
// 说错时又总是凭空多出一件; 期望 1 件左右时有没有、有几件才有信息量。
func hintCountQuality(odds []venue.PieceOdds) int16 {
	best, gap := int16(0), math.Inf(1)
	for _, q := range hostQualities(odds, nil) {
		var count float64
		for _, o := range odds {
			if o.Quality >= q {
				count += o.Count
			}
		}
		if count < hintCountMin || count > hintCountMax {
			continue
		}
		if g := math.Abs(math.Log(count)); g < gap {
			best, gap = q, g
		}
	}
	return best
}

// lieRate 这条提示说错的概率, 基点: 稳重的减半, 煽动的在热度高时、狡黠的在热度低时 × 1.5, 涉及高品质时再 × 1.5, 不超过 maxLieRate
func lieRate(in *hostInput, hint *HostHint) int32 {
	rate := in.Rule.LieRate
	switch in.Persona {
	case venue.HostSteady:
		rate /= 2
	case venue.HostShowman:
		if in.Hype >= hypeHot {
			rate = rate * 3 / 2
		}
	case venue.HostSly:
		if in.Hype <= hypeCold {
			rate = rate * 3 / 2
		}
	}
	if high := in.Rule.HighQuality; high > 0 && hint.Kind != HostHintValue && hint.Quality >= high {
		rate = rate * 3 / 2
	}
	return min(rate, maxLieRate)
}

// lie 把提示改成说错的版本: 煽动的只往高了说, 狡黠的只往低了说, 稳重的随机;
// 往这个方向说不了时(如已是最高品质)照实说, 稳重的改往另一边
func lie(in *hostInput, hint *HostHint, randN func(n int64) int64) {
	var directions []bool
	switch in.Persona {
	case venue.HostShowman:
		directions = []bool{true}
	case venue.HostSly:
		directions = []bool{false}
	default:
		up := randN(2) == 0
		directions = []bool{up, !up}
	}
	for _, up := range directions {
		if shiftHint(in, hint, up) {
			hint.Truthful = false
			return
		}
	}
}

// shiftHint 把提示往高(up)或往低改一档, 改不了时返回 false
func shiftHint(in *hostInput, hint *HostHint, up bool) bool {
	switch hint.Kind {
	case HostHintCount:
		if up {
			hint.Count++
			return true
		}
		if hint.Count > 0 {
			hint.Count--
			return true
		}
	case HostHintBest, HostHintPiece:
		i := slices.Index(in.Qualities, hint.Quality)
		switch {
		case i < 0:
		case up && i+1 < len(in.Qualities):
			hint.Quality = in.Qualities[i+1]
			return true
		case !up && i > 0:
			hint.Quality = in.Qualities[i-1]
			return true
		}
	case HostHintValue:
		if hint.Above != up {
			hint.Above = up
			return true
		}
	}
	return false
}

// expectedValue 按公示概率算出的货仓期望价值, 与人机估值用的分布相同
func expectedValue(odds []venue.PieceOdds) int64 {
	var total float64
	for _, o := range odds {
		total += o.Count * o.Price
	}
	return int64(math.Round(total))
}

// hostQualities 可能出现的品质, 从低到高: 公示概率中的品质加上货仓里实际有的
func hostQualities(odds []venue.PieceOdds, items []DepotItem) []int16 {
	var qualities []int16
	for _, o := range odds {
		qualities = append(qualities, o.Quality)
	}
	for _, it := range items {
		qualities = append(qualities, it.Quality)
	}
	slices.Sort(qualities)
	return slices.Compact(qualities)
}

// hypeDelta 一轮结束后的热度变动: 每个加价的座位按加价与正常节奏(期望价值 ÷ 轮数)之比推高, 每个至少 1、至多 hypeRaiseMax,
// 再统一降温; 大家按正常节奏加价时热度大致不变
//
// 只看加价的人数和幅度, 不透露谁加了多少; 随便报个价也能推高一点, 可以用来虚张声势。
func hypeDelta(raises []int64, expected int64, rounds int32) int32 {
	if len(raises) == 0 {
		return -hypeIdle
	}
	delta := int32(-hypeCooling)
	for _, raise := range raises {
		points := int64(hypeRaiseMax)
		if expected > 0 {
			points = min(max(raise*hypePacePoints*int64(max(rounds, 1))/expected, 1), hypeRaiseMax)
		}
		delta += int32(points)
	}
	return delta
}

// startHost 开局时选出主持人的性格(仓库结束后才公开), 记录初始热度, 给出第 1 个拍品第 1 轮的提示; 没有开启主持人提示时什么都不做
func startHost(tx *gorm.DB, a *Match, lot *Lot, items []DepotItem) error {
	rule := &a.Venue.Host
	if !rule.Enabled {
		return nil
	}
	personas := rule.Personas
	if len(personas) == 0 {
		personas = venue.HostPersonas
	}
	a.Host, a.Hype = personas[rand.IntN(len(personas))], hypeStart
	if err := tx.Model(a).Select("host", "hype").Updates(a).Error; err != nil {
		return err
	}
	if err := tx.Create(&HypeEvent{MatchID: a.MatchID, LotNo: lot.LotNo, Delta: hypeStart, Hype: hypeStart}).Error; err != nil {
		return err
	}
	_, err := giveHint(tx, a, lot, items, lot.FirstRound, nil)
	return err
}

// recordHype 记录当前拍品上热度的一次变动, hype 为变动后的热度; round 为结束的轮次, 拍品开场(热度回落)时为 0
func (st *state) recordHype(tx *gorm.DB, round, raises, hype int32) error {
	a := &st.auction
	event := HypeEvent{MatchID: a.MatchID, LotNo: st.lot.LotNo, Round: round, Raises: raises, Delta: hype - a.Hype, Hype: hype}
	if err := tx.Create(&event).Error; err != nil {
		return err
	}
	a.Hype = hype
	return tx.Model(a).Select("hype").Updates(a).Error
}

// hint 给出当前拍品在 s 的当前轮次的提示(见 makeHint), 并交给 s, 这一轮人机决策时也看得到
func (st *state) hint(tx *gorm.DB, s *warehouse) error {
	a, lot := &st.auction, st.lot
	items, err := loadItems(tx, a.MatchID, lot.LotNo)
	if err != nil {
		return err
	}
	given, err := loadHints(tx, a.MatchID, lot.LotNo)
	if err != nil {
		return err
	}
	hint, err := giveHint(tx, a, lot, items, s.Round, given)
	if err != nil || hint == nil {
		return err
	}
	s.addHint(publicHint(hint))
	return nil
}

// giveHint 生成并保存拍品 lot 在仓库第 round 轮的提示, given 为这个拍品已经给过的; 没有能给的提示时返回 nil
func giveHint(tx *gorm.DB, a *Match, lot *Lot, items []DepotItem, round int32, given []HostHint) (*HostHint, error) {
	in := &hostInput{
		Round:        lot.Round(round),
		Items:        items,
		Value:        lot.Value,
		Expected:     expectedValue(lot.Odds),
		Qualities:    hostQualities(lot.Odds, items),
		CountQuality: hintCountQuality(lot.Odds),
		Rule:         &a.Venue.Host,
		Persona:      a.Host,
		Hype:         a.Hype,
		Given:        given,
	}
	hint, ok := makeHint(in, rand.Int64N)
	if !ok {
		return nil, nil
	}
	hint.MatchID, hint.LotNo, hint.Round = a.MatchID, lot.LotNo, round
	if err := tx.Create(&hint).Error; err != nil {
		return nil, err
	}
	return &hint, nil
}

// loadHints 拍品 lotNo 已经给过的提示, 按轮次
func loadHints(tx *gorm.DB, matchID uint64, lotNo int32) ([]HostHint, error) {
	var hints []HostHint
	if err := tx.Where("match_id = ? AND lot_no = ?", matchID, lotNo).Order("round").Find(&hints).Error; err != nil {
		return nil, err
	}
	return hints, nil
}

// loadHost 一局的提示和热度变动, 按时间
func loadHost(tx *gorm.DB, matchID uint64) ([]HostHint, []HypeEvent, error) {
	var hints []HostHint
	if err := tx.Where("match_id = ?", matchID).Order("round").Find(&hints).Error; err != nil {
		return nil, nil, err
	}
	var hypes []HypeEvent
	if err := tx.Where("match_id = ?", matchID).Order("id").Find(&hypes).Error; err != nil {
		return nil, nil, err
	}
	return hints, hypes, nil
}

// HintView 公开的一条提示, 字段见 HostHint
type HintView struct {
	LotNo    int32  `json:"lot_no"`
	Round    int32  `json:"round"` // 仓库内的轮次
	Kind     string `json:"kind"`  // 见 constants.go 的 HostHint
	Quality  int16  `json:"quality"`
	Count    int32  `json:"count"`
	X        int16  `json:"x"` // 指认的藏品的位置, 是给出提示时已经扫出轮廓的一件
	Y        int16  `json:"y"`
	Above    bool   `json:"above"`
	Truthful *bool  `json:"truthful"` // 是否属实, 仓库结束后才公开
}

// HostView 公开的主持人状态, 没有主持人提示的对局为空
type HostView struct {
	Persona string      `json:"persona"` // 见 venue.HostSteady 等, 仓库结束后才公开, 之前为空
	Hype    int32       `json:"hype"`
	Hints   []HintView  `json:"hints"` // 按轮次
	Hypes   []HypeEvent `json:"hypes"` // 热度的每次变动, 按时间
}

// publicHint 一条提示在仓库结束前公开的部分, 不含是否属实
func publicHint(h *HostHint) HintView {
	return HintView{LotNo: h.LotNo, Round: h.Round, Kind: h.Kind, Quality: h.Quality, Count: h.Count, X: h.X, Y: h.Y, Above: h.Above}
}

// publicHints 见 publicHint
func publicHints(hints []HostHint) []HintView {
	views := make([]HintView, len(hints))
	for i := range hints {
		views[i] = publicHint(&hints[i])
	}
	return views
}

// hostView 主持人的公开状态; reveal 为是否公开性格和提示是否属实
func hostView(a *Match, hints []HostHint, hypes []HypeEvent, reveal bool) *HostView {
	if a.Host == "" {
		return nil
	}
	view := &HostView{Hype: a.Hype, Hints: publicHints(hints), Hypes: hypes}
	if reveal {
		view.Persona = a.Host
	}
	if view.Hypes == nil {
		view.Hypes = []HypeEvent{}
	}
	if reveal {
		for i := range hints {
			view.Hints[i].Truthful = &hints[i].Truthful
		}
	}
	return view
}
