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
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
)

func init() {
	condition.RegisterScopeNamer("mode", modeScopeName)
	condition.RegisterScopeNamer("venue", venueScopeName)
}

// codePattern 地图代码, 小写字母、数字和下划线; 玩法代码最长 32, 需要留出 "ranked:" 前缀
var codePattern = regexp.MustCompile(`^[a-z0-9_]{1,24}$`)

// modeNames 模式的展示名称
var modeNames = map[string]string{
	ModePve:    "人机",
	ModeRanked: "排位",
}

// InvalidError 地图配置不合法, Error 返回的提示可以直接展示给管理员
type InvalidError struct {
	Reason string
}

func (e *InvalidError) Error() string { return e.Reason }

// invalid 生成 *InvalidError
func invalid(format string, args ...any) error {
	if len(args) == 0 {
		return &InvalidError{Reason: format}
	}
	return &InvalidError{Reason: fmt.Sprintf(format, args...)}
}

// Validate 保存地图前校验配置, 并模拟 SimulationRuns 个仓库, 把结果写入 v.Simulation
//
// 配置不合法或摆放失败率超过 1% 时返回 *InvalidError, 提示可以直接展示; 其他错误为查库失败等内部错误。
// 名称等文本字段需要调用方先去掉首尾空白。
func Validate(tx *gorm.DB, v *Venue) error {
	if !codePattern.MatchString(v.Code) {
		return invalid(CodeInvalid)
	}
	if n := utf8.RuneCountInString(v.Name); n < 1 || n > 64 {
		return invalid(NameInvalid)
	}
	if err := checkCurrencies(tx, []string{v.BidCurrency}); err != nil {
		return err
	}
	if err := validateEntryCost(tx, v); err != nil {
		return err
	}
	if v.Requirements == nil {
		v.Requirements = []condition.Condition{}
	}
	if err := condition.Validate(tx, v.Requirements); err != nil {
		var conditionErr *condition.InvalidError
		if errors.As(err, &conditionErr) {
			return invalid(RequirementInvalid, conditionErr.Error())
		}
		return err
	}
	if err := validateRules(v); err != nil {
		return err
	}

	generator, err := NewGenerator(tx, v.Warehouse, v.BidCurrency)
	if err != nil {
		return err
	}
	v.Simulation = Simulation{
		Runs:        SimulationRuns,
		Ranked:      generator.Simulate(SimulationRuns, BasisPoints),
		Pve:         generator.Simulate(SimulationRuns, v.Pve.ValueFactor),
		SimulatedAt: time.Now(),
	}
	if rate := max(v.Simulation.Ranked.FailureRate, v.Simulation.Pve.FailureRate); rate > maxFailureRate {
		return invalid(FailureRateTooHigh, float64(rate)/100)
	}
	return nil
}

// validateEntryCost 入场代价的数量需为正数, 货币需存在, 物品需未删除; 校验后合并同类项
func validateEntryCost(tx *gorm.DB, v *Venue) error {
	cost := v.EntryCost
	if len(cost.Currencies)+len(cost.Items) > maxEntryCostLines {
		return invalid(EntryCostTooMany)
	}
	for _, line := range cost.Currencies {
		if line.Amount <= 0 {
			return invalid(EntryCostInvalid)
		}
	}
	for _, line := range cost.Items {
		if line.Quantity <= 0 {
			return invalid(EntryCostInvalid)
		}
	}
	normalized, err := cost.Normalize()
	if err != nil {
		return &InvalidError{Reason: err.Error()}
	}

	codes := make([]string, len(normalized.Currencies))
	for i, line := range normalized.Currencies {
		codes[i] = line.Currency
	}
	if err := checkCurrencies(tx, codes); err != nil {
		return err
	}
	if len(normalized.Items) > 0 {
		itemIDs := make([]uint64, len(normalized.Items))
		for i, line := range normalized.Items {
			itemIDs[i] = line.ItemID
		}
		var count int64
		if err := tx.Model(&item.Item{}).Where("id IN ?", itemIDs).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(itemIDs)) {
			return invalid(item.ItemNotFound)
		}
	}
	v.EntryCost = normalized
	return nil
}

// checkCurrencies 货币需存在
func checkCurrencies(tx *gorm.DB, codes []string) error {
	if len(codes) == 0 {
		return nil
	}
	codes = slices.Compact(slices.Sorted(slices.Values(codes)))
	var count int64
	if err := tx.Model(&wallet.Currency{}).Where("code IN ?", codes).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(codes)) {
		return invalid(wallet.CurrencyNotFound)
	}
	return nil
}

// validateRules 按 binding 规则校验各项参数, 再检查字段之间的关系
func validateRules(v *Venue) error {
	sections := []struct {
		name  string
		value any
	}{
		{"仓库规则", &v.Warehouse},
		{"拍卖参数", &v.Auction},
		{"人机参数", &v.Bots},
		{"人机模式", &v.Pve},
	}
	for _, section := range sections {
		if err := settings.ValidateStruct(section.value); err != nil {
			return invalid(RuleInvalid, section.name, err.Error())
		}
	}

	var maxTimes int32
	for i, slot := range v.Warehouse.Slots {
		if slot.Min > slot.Max {
			return invalid(SlotRangeInvalid, i+1)
		}
		maxTimes += slot.Max
	}
	if maxTimes == 0 {
		return invalid(SlotsEmpty)
	}
	if v.Auction.MaxCarry < v.Auction.StartPrice {
		return invalid(MaxCarryInvalid)
	}
	if len(v.Auction.QuickWin) != int(v.Auction.Rounds)-1 {
		return invalid(QuickWinInvalid, v.Auction.Rounds-1)
	}
	var weights int32
	for _, weight := range v.Bots.LevelWeights {
		weights += weight
	}
	if weights == 0 {
		return invalid(BotWeightsInvalid)
	}
	return nil
}

// ModeName 模式的展示名称, 如 "排位"; 未知的模式返回代码本身
func ModeName(mode string) string {
	if name, ok := modeNames[mode]; ok {
		return name
	}
	return mode
}

// modeScopeName 统计范围 mode:<模式> 的展示名称, 如 "排位"
func modeScopeName(_ *gorm.DB, scope string) (string, error) {
	return modeNames[strings.TrimPrefix(scope, "mode:")], nil
}

// venueScopeName 统计范围 venue:<代码> 和 venue:<代码>:<模式> 的展示名称, 如 "码头仓库"、"码头仓库·排位"
//
// 已删除的地图仍按原名称展示, 地图不存在时展示统计范围本身。
func venueScopeName(tx *gorm.DB, scope string) (string, error) {
	code, mode, hasMode := strings.Cut(strings.TrimPrefix(scope, "venue:"), ":")
	var venues []Venue
	if err := tx.Unscoped().Select("name").Where("code = ?", code).Limit(1).Find(&venues).Error; err != nil {
		return "", err
	}
	if len(venues) == 0 {
		return "", nil
	}
	name := venues[0].Name
	if hasMode {
		name += "·" + ModeName(mode)
	}
	return name, nil
}

// CostLine 入场代价的一行, 附带货币或物品的名称
type CostLine struct {
	Currency string `json:"currency,omitempty"` // 与 item_id 二选一
	ItemID   uint64 `json:"item_id,omitempty"`
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
	Amount   int64  `json:"amount"` // 货币金额或物品数量
}

// ValueRange 玩家看到的仓库价值区间
type ValueRange struct {
	P10 int64 `json:"p10"`
	P50 int64 `json:"p50"`
	P90 int64 `json:"p90"`
}

// PveView 玩家看到的人机模式参数
type PveView struct {
	PracticeOnly bool       `json:"practice_only"`
	ValueFactor  int32      `json:"value_factor"` // 仓库价值系数, 基点
	ValueRange   ValueRange `json:"value_range"`
}

// VenueView 玩家看到的地图
type VenueView struct {
	Code          string               `json:"code"`
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	ImageURL      string               `json:"image_url"`
	PveEnabled    bool                 `json:"pve_enabled"`
	RankedEnabled bool                 `json:"ranked_enabled"`
	BidCurrency   string               `json:"bid_currency"`
	EntryCost     []CostLine           `json:"entry_cost"`
	Requirements  []condition.Progress `json:"requirements"` // 每个入场条件的达成情况
	Eligible      bool                 `json:"eligible"`     // 满足全部入场条件
	Grid          Grid                 `json:"grid"`
	Auction       AuctionRule          `json:"auction"`
	ValueRange    ValueRange           `json:"value_range"` // 排位模式的仓库价值区间, 保存地图时模拟
	Pve           PveView              `json:"pve"`
}

// SlotOdds 仓库的一个档位及其掉落池的公示概率
type SlotOdds struct {
	Slot
	Odds *loot.PoolOdds `json:"odds"` // 掉落池不可用时为空
}

// VenueDetail 玩家看到的地图详情
type VenueDetail struct {
	VenueView
	Slots []SlotOdds `json:"slots"`
}

// QueryVenues 获取启用的地图, 按排序值和 ID 排列, 附带用户对入场条件的达成情况
func QueryVenues(ctx context.Context, userID uint64) ([]VenueView, error) {
	tx := db.DB(ctx)
	var venues []Venue
	if err := tx.Where("is_enabled").Order("sort_order, id").Find(&venues).Error; err != nil {
		return nil, err
	}
	costs, err := resolveCosts(tx, venues)
	if err != nil {
		return nil, err
	}
	views := make([]VenueView, len(venues))
	for i := range venues {
		view, err := newView(tx, userID, &venues[i], costs)
		if err != nil {
			return nil, err
		}
		views[i] = *view
	}
	return views, nil
}

// QueryVenue 获取一张启用的地图的详情, 不存在或已停用时返回 VenueNotFound
func QueryVenue(ctx context.Context, userID uint64, code string) (*VenueDetail, error) {
	tx := db.DB(ctx)
	var v Venue
	if err := v.ExactCode(tx, code); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(VenueNotFound)
		}
		return nil, err
	}
	if !v.IsEnabled {
		return nil, errors.New(VenueNotFound)
	}
	costs, err := resolveCosts(tx, []Venue{v})
	if err != nil {
		return nil, err
	}
	view, err := newView(tx, userID, &v, costs)
	if err != nil {
		return nil, err
	}

	detail := &VenueDetail{VenueView: *view, Slots: make([]SlotOdds, len(v.Warehouse.Slots))}
	for i, slot := range v.Warehouse.Slots {
		detail.Slots[i] = SlotOdds{Slot: slot}
		odds, err := loot.QueryOdds(ctx, slot.Pool)
		if err != nil {
			switch err.Error() {
			case loot.PoolNotFound, loot.PoolDisabled, loot.PoolEmpty:
				continue
			}
			return nil, err
		}
		detail.Slots[i].Odds = odds
	}
	return detail, nil
}

// costNames 入场代价中货币和物品的名称和图片
type costNames struct {
	currencies map[string]string
	items      map[uint64]*item.Item
}

// resolveCosts 批量获取地图入场代价中的货币和物品
func resolveCosts(tx *gorm.DB, venues []Venue) (*costNames, error) {
	codes := map[string]struct{}{}
	itemIDs := map[uint64]struct{}{}
	for _, v := range venues {
		for _, line := range v.EntryCost.Currencies {
			codes[line.Currency] = struct{}{}
		}
		for _, line := range v.EntryCost.Items {
			itemIDs[line.ItemID] = struct{}{}
		}
	}

	names := &costNames{currencies: make(map[string]string, len(codes))}
	if len(codes) > 0 {
		var currencies []wallet.Currency
		if err := tx.Where("code IN ?", slices.Collect(maps.Keys(codes))).Find(&currencies).Error; err != nil {
			return nil, err
		}
		for _, currency := range currencies {
			names.currencies[currency.Code] = currency.Name
		}
	}
	items, err := item.Resolve(tx, slices.Collect(maps.Keys(itemIDs)))
	if err != nil {
		return nil, err
	}
	names.items = items
	return names, nil
}

// newView 生成玩家看到的地图
func newView(tx *gorm.DB, userID uint64, v *Venue, names *costNames) (*VenueView, error) {
	progress, err := condition.Evaluate(tx, userID, v.Requirements)
	if err != nil {
		return nil, err
	}
	view := &VenueView{
		Code:          v.Code,
		Name:          v.Name,
		Description:   v.Description,
		ImageURL:      v.ImageURL,
		PveEnabled:    v.PveEnabled,
		RankedEnabled: v.RankedEnabled,
		BidCurrency:   v.BidCurrency,
		EntryCost:     make([]CostLine, 0, len(v.EntryCost.Currencies)+len(v.EntryCost.Items)),
		Requirements:  progress,
		Eligible:      true,
		Grid:          v.Warehouse.Grid,
		Auction:       v.Auction,
		ValueRange:    newValueRange(v.Simulation.Ranked),
		Pve: PveView{
			PracticeOnly: v.Pve.PracticeOnly,
			ValueFactor:  v.Pve.ValueFactor,
			ValueRange:   newValueRange(v.Simulation.Pve),
		},
	}
	for _, p := range progress {
		if !p.Met {
			view.Eligible = false
		}
	}
	for _, line := range v.EntryCost.Currencies {
		name := names.currencies[line.Currency]
		if name == "" {
			name = line.Currency
		}
		view.EntryCost = append(view.EntryCost, CostLine{Currency: line.Currency, Name: name, Amount: line.Amount})
	}
	for _, line := range v.EntryCost.Items {
		target := names.items[line.ItemID]
		view.EntryCost = append(view.EntryCost, CostLine{ItemID: line.ItemID, Name: target.Name, ImageURL: target.ImageURL, Amount: line.Quantity})
	}
	return view, nil
}

// newValueRange 取模拟结果中的价值区间
func newValueRange(stats ValueStats) ValueRange {
	return ValueRange{P10: stats.P10, P50: stats.P50, P90: stats.P90}
}
