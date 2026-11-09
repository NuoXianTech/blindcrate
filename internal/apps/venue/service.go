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
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/effect"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
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

// Validate 保存地图前校验配置, 并模拟 SimulationRuns 个仓库, 把结果写入 v.Simulation;
// 再按排位模式的价值分布推导拍卖的金额和比例写入 v.Auction, 由注册的 Balancer 验算对局, 结果写入 v.Simulation.Balance
//
// 配置不合法、摆放失败率超过 1% 或启用的地图验算未通过时返回 *InvalidError, 提示可以直接展示; 其他错误为查库失败等内部错误。
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
	if err := validateBots(tx, v); err != nil {
		return err
	}
	if err := validateConsumables(tx, v); err != nil {
		return err
	}
	if err := validateHost(tx, v); err != nil {
		return err
	}

	lineup, err := NewLineup(tx, v)
	if err != nil {
		return err
	}
	v.Simulation = Simulation{
		Runs:        SimulationRuns,
		Ranked:      lineup.Simulate(SimulationRuns, BasisPoints),
		Pve:         lineup.Simulate(SimulationRuns, v.Pve.ValueFactor),
		Tiers:       []TierStats{},
		SimulatedAt: time.Now(),
	}
	if v.Lots.Enabled() {
		for i := range v.Lots.Tiers {
			g := lineup.Tier(i)
			v.Simulation.Tiers = append(v.Simulation.Tiers, TierStats{
				Code:   v.Lots.Tiers[i].Code,
				Ranked: g.Simulate(SimulationRuns, BasisPoints),
				Pve:    g.Simulate(SimulationRuns, v.Pve.ValueFactor),
			})
		}
	}
	rates := []int32{v.Simulation.Ranked.FailureRate, v.Simulation.Pve.FailureRate}
	for _, t := range v.Simulation.Tiers {
		rates = append(rates, t.Ranked.FailureRate, t.Pve.FailureRate)
	}
	if rate := slices.Max(rates); rate > maxFailureRate {
		return invalid(FailureRateTooHigh, float64(rate)/100)
	}
	if v.Simulation.Ranked.Mean <= 0 {
		return invalid(ValueEmpty)
	}
	for _, t := range v.Simulation.Tiers {
		if t.Ranked.Mean <= 0 {
			return invalid(ValueEmpty)
		}
	}

	rule, version, err := settings.Balance.Load(tx)
	if err != nil {
		return err
	}
	v.Calibrate(rule)
	if balancer == nil {
		return nil
	}
	balance, err := balancer(tx, v, lineup, rule)
	if err != nil {
		return err
	}
	balance.judge(v, rule)
	balance.Version = version
	v.Simulation.Balance = balance
	if v.IsEnabled && !balance.Passed {
		return invalid(BalanceFailed, balance.failures())
	}
	return nil
}

// validateEntryCost 入场代价的数量需为正数, 货币需存在且不是出价货币(出价货币的入场费自动计算), 物品需未删除; 校验后合并同类项
func validateEntryCost(tx *gorm.DB, v *Venue) error {
	cost := v.EntryCost
	if len(cost.Currencies)+len(cost.Items) > maxEntryCostLines {
		return invalid(EntryCostTooMany)
	}
	for _, line := range cost.Currencies {
		if line.Amount <= 0 {
			return invalid(EntryCostInvalid)
		}
		if line.Currency == v.BidCurrency {
			return invalid(EntryCostBidMoney)
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
	type section struct {
		name  string
		value any
	}
	// 配置了档次时不用 Warehouse, 不校验
	var sections []section
	if !v.Lots.Enabled() {
		sections = append(sections, section{"仓库规则", &v.Warehouse})
	}
	sections = append(sections, []section{
		{"拍场", &v.Lots},
		{"拍卖参数", &v.Auction},
		{"人机风格", &v.Bots},
		{"人机模式", &v.Pve},
		{"道具规则", &v.Consumables},
		{"主持人", &v.Host},
	}...)
	for _, section := range sections {
		if err := settings.ValidateStruct(section.value); err != nil {
			return invalid(RuleInvalid, section.name, err.Error())
		}
	}

	if !v.Lots.Enabled() {
		if err := validateWarehouse(&v.Warehouse); err != nil {
			return err
		}
	}
	if err := validateLots(&v.Lots); err != nil {
		return err
	}
	if len(v.Auction.QuickWin) != int(v.Auction.Rounds)-1 {
		return invalid(QuickWinInvalid, v.Auction.Rounds-1)
	}
	return nil
}

// validateWarehouse 档位的次数范围需合法, 至少能抽到一件; 旧数据和没有提交时为 null 的数量波动统一写为空列表
func validateWarehouse(w *WarehouseRule) error {
	if w.Swings == nil {
		w.Swings = []Swing{}
	}
	var maxTimes int32
	for i, slot := range w.Slots {
		if slot.Min > slot.Max {
			return invalid(SlotRangeInvalid, i+1)
		}
		maxTimes += slot.Max
	}
	if maxTimes == 0 {
		return invalid(SlotsEmpty)
	}
	return nil
}

// validateLots 配置了档次时拍品数需在 [1, 8] 且最少不超过最多, 档次代码不能重复; 每个档次的仓库规则同 Warehouse 校验
func validateLots(r *LotRule) error {
	if r.Tiers == nil {
		r.Tiers = []Tier{}
	}
	if !r.Enabled() {
		return nil
	}
	if r.Min < 1 || r.Min > r.Max {
		return invalid(LotRangeInvalid)
	}
	codes := map[string]bool{}
	for i := range r.Tiers {
		t := &r.Tiers[i]
		if codes[t.Code] {
			return invalid(TierDuplicate, t.Code)
		}
		codes[t.Code] = true
		if err := validateWarehouse(&t.Warehouse); err != nil {
			return invalid(TierInvalid, t.Name, err.Error())
		}
	}
	return nil
}

// validateBots 开放的模式至少配置一种人机风格, 风格不能重复且需存在; 校验后没有配置的模式写为空列表
func validateBots(tx *gorm.DB, v *Venue) error {
	modes := []struct {
		mode    string
		enabled bool
		weights *[]bot.Weight
	}{
		{ModePve, v.PveEnabled, &v.Bots.Pve},
		{ModeRanked, v.RankedEnabled, &v.Bots.Ranked},
	}
	for _, m := range modes {
		if *m.weights == nil {
			*m.weights = []bot.Weight{}
		}
		if m.enabled && len(*m.weights) == 0 {
			return invalid(BotsEmpty, ModeName(m.mode))
		}
		reason, err := bot.CheckWeights(tx, *m.weights)
		if err != nil {
			return err
		}
		if reason != "" {
			return invalid(BotsInvalid, ModeName(m.mode), reason)
		}
	}
	return nil
}

// UsingPersona 返回一张引用了这个人机风格的未删除地图的代码, 没有时返回空字符串, 删除风格前调用
func UsingPersona(tx *gorm.DB, persona string) (string, error) {
	var codes []string
	err := tx.Model(&Venue{}).
		Where("bots @> ?::jsonb OR bots @> ?::jsonb", personaFilter(ModePve, persona), personaFilter(ModeRanked, persona)).
		Order("id").Limit(1).Pluck("code", &codes).Error
	if err != nil || len(codes) == 0 {
		return "", err
	}
	return codes[0], nil
}

// personaFilter 按模式匹配引用了风格的人机配置, 用于 jsonb 的 @> 查询
func personaFilter(mode, persona string) string {
	raw, _ := json.Marshal(map[string][]map[string]string{mode: {{"persona": persona}}})
	return string(raw)
}

// validateConsumables 可用道具需为未删除且效果属于对局场景的物品; 校验后去掉重复的 ID
func validateConsumables(tx *gorm.DB, v *Venue) error {
	allowed := slices.Compact(slices.Sorted(slices.Values(v.Consumables.Allowed)))
	if allowed == nil {
		allowed = []uint64{}
	}
	v.Consumables.Allowed = allowed
	if len(allowed) == 0 {
		return nil
	}
	var items []item.Item
	if err := tx.Where("id IN ?", allowed).Find(&items).Error; err != nil {
		return err
	}
	found := make(map[uint64]*item.Item, len(items))
	for i := range items {
		found[items[i].ID] = &items[i]
	}
	for _, id := range allowed {
		target, ok := found[id]
		if !ok {
			return invalid(ConsumableNotFound, id)
		}
		if effect.SceneOf(target.Effect) != effect.SceneMatch {
			return invalid(ConsumableUnusable, target.Name)
		}
	}
	return nil
}

// validateHost 高品质需存在, 卖情报时估价情报的误差不能太小; 校验后去掉重复的性格, 没有配置时写为空列表
func validateHost(tx *gorm.DB, v *Venue) error {
	rule := &v.Host
	personas := []string{}
	for _, persona := range HostPersonas {
		if slices.Contains(rule.Personas, persona) {
			personas = append(personas, persona)
		}
	}
	rule.Personas = personas
	if rule.IntelLimit > 0 && rule.IntelPrecision < minIntelPrecision {
		return invalid(IntelPrecisionLow)
	}
	if rule.HighQuality == 0 {
		return nil
	}
	if err := item.CheckQuality(tx, rule.HighQuality); err != nil {
		if err.Error() == item.QualityNotFound {
			return invalid(HostQualityInvalid, rule.HighQuality)
		}
		return err
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
	PracticeOnly bool        `json:"practice_only"`
	ValueFactor  int32       `json:"value_factor"` // 仓库价值系数, 基点
	ValueRange   ValueRange  `json:"value_range"`
	ProfitCaps   []ProfitCap `json:"profit_caps"` // 用户今天受到的每日收益上限, 不限制时为空; 任一上限达到后人机对局转为练习局
}

// ProfitCap 人机模式的一个每日收益上限和用户当天的净收益, 金额为出价货币的最小单位
type ProfitCap struct {
	Scope  string `json:"scope"` // 统计范围: mode:pve 为全局的上限, venue:<地图代码>:pve 为这张地图的上限
	Limit  int64  `json:"limit"`
	Profit int64  `json:"profit"` // 当天的净收益
}

// Reached 是否达到上限, 达到后当天的人机对局转为练习局
func (c ProfitCap) Reached() bool {
	return c.Profit >= c.Limit
}

// PveProfitCaps 用户在 at 当天的人机每日收益上限: 全局的上限和这张地图的上限, 按地图的出价货币计算; 为 0 的上限不限制, 不返回
func PveProfitCaps(tx *gorm.DB, userID uint64, v *Venue, at time.Time) ([]ProfitCap, error) {
	economySettings, err := settings.Economy.Get(tx.Statement.Context)
	if err != nil {
		return nil, err
	}
	limits := []ProfitCap{
		{Scope: stats.ScopeMode(ModePve), Limit: int64(economySettings.PveDailyProfitCap)},
		{Scope: stats.ScopeVenueMode(v.Code, ModePve), Limit: v.Pve.DailyProfitCap},
	}
	day, metric := stats.PeriodDay(at), stats.MetricProfit(v.BidCurrency)
	caps := make([]ProfitCap, 0, len(limits))
	for _, c := range limits {
		if c.Limit == 0 {
			continue
		}
		if c.Profit, err = stats.Get(tx, userID, c.Scope, day, metric); err != nil {
			return nil, err
		}
		caps = append(caps, c)
	}
	return caps, nil
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
	EntryCost     []CostLine           `json:"entry_cost"`   // 开局时扣除的全部代价, 出价货币的入场费在前
	Requirements  []condition.Progress `json:"requirements"` // 每个入场条件的达成情况
	Eligible      bool                 `json:"eligible"`     // 满足全部入场条件
	Grid          Grid                 `json:"grid"`
	Auction       AuctionRule          `json:"auction"`
	Host          HostRule             `json:"host"`        // 主持人的提示和情报
	ValueRange    ValueRange           `json:"value_range"` // 排位模式的仓库价值区间, 保存地图时模拟; 拍场为单个拍品的
	Pve           PveView              `json:"pve"`
	Lots          *LotsView            `json:"lots"` // 拍场的拍品数和格子大小, 没有配置档次(一局一个仓库)时为空
}

// LotsView 玩家看到的拍场: 拍品数和每种格子大小的出现比例、起拍价; 档次不公开, 只按格子大小汇总
type LotsView struct {
	Min        int32     `json:"min"`
	Max        int32     `json:"max"`
	GapSeconds int32     `json:"gap_seconds"`
	Grids      []LotGrid `json:"grids"` // 按第一次出现在档次中的顺序
}

// LotGrid 一种格子大小的拍品
type LotGrid struct {
	Grid       Grid  `json:"grid"`
	Share      int32 `json:"share"` // 拍品是这种格子大小的概率, 基点
	StartPrice int64 `json:"start_price"`
	SpeedFloor int64 `json:"speed_floor"`
	IntelPrice int64 `json:"intel_price"`
}

// newLotsView 按格子大小汇总档次, 没有配置档次时返回空
func newLotsView(v *Venue) *LotsView {
	if !v.Lots.Enabled() {
		return nil
	}
	view := &LotsView{Min: v.Lots.Min, Max: v.Lots.Max, GapSeconds: v.Lots.GapSeconds, Grids: []LotGrid{}}
	var total int64
	for _, t := range v.Lots.Tiers {
		total += int64(t.Weight)
	}
	for _, t := range v.Lots.Tiers {
		i := slices.IndexFunc(view.Grids, func(g LotGrid) bool { return g.Grid == t.Warehouse.Grid })
		if i < 0 {
			view.Grids = append(view.Grids, LotGrid{Grid: t.Warehouse.Grid, StartPrice: t.Auction.StartPrice, SpeedFloor: t.Auction.SpeedFloor, IntelPrice: t.Auction.IntelPrice})
			i = len(view.Grids) - 1
		}
		view.Grids[i].Share += int32(int64(t.Weight) * BasisPoints / max(total, 1))
	}
	return view
}

// SlotOdds 仓库的一个档位及其掉落池的公示概率
type SlotOdds struct {
	Slot
	Odds *loot.PoolOdds `json:"odds"` // 掉落池不可用时为空
}

// VenueDetail 玩家看到的地图详情
type VenueDetail struct {
	VenueView
	Slots    []SlotOdds `json:"slots"`     // 没有配置档次时仓库的档位; 拍场为空, 见 LotSlots
	LotSlots []GridOdds `json:"lot_slots"` // 拍场每种格子大小的仓库, 没有配置档次时为空
}

// GridOdds 拍场中一种格子大小的仓库: 同一格子大小的各档次不公开名称, 按权重给出概率和档位
type GridOdds struct {
	Grid     Grid          `json:"grid"`
	Variants []VariantOdds `json:"variants"`
}

// VariantOdds 一种格子大小中的一个档次, 不含档次的代码和名称
type VariantOdds struct {
	Share int32      `json:"share"` // 这个格子大小的拍品是这一档的概率, 基点
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

	detail := &VenueDetail{VenueView: *view, Slots: []SlotOdds{}, LotSlots: []GridOdds{}}
	if !v.Lots.Enabled() {
		if detail.Slots, err = slotOdds(ctx, v.Warehouse.Slots); err != nil {
			return nil, err
		}
		return detail, nil
	}
	weights := map[Grid]int64{}
	for _, t := range v.Lots.Tiers {
		weights[t.Warehouse.Grid] += int64(t.Weight)
	}
	for _, t := range v.Lots.Tiers {
		grid := t.Warehouse.Grid
		i := slices.IndexFunc(detail.LotSlots, func(g GridOdds) bool { return g.Grid == grid })
		if i < 0 {
			detail.LotSlots = append(detail.LotSlots, GridOdds{Grid: grid, Variants: []VariantOdds{}})
			i = len(detail.LotSlots) - 1
		}
		slots, err := slotOdds(ctx, t.Warehouse.Slots)
		if err != nil {
			return nil, err
		}
		share := int32(int64(t.Weight) * BasisPoints / max(weights[grid], 1))
		detail.LotSlots[i].Variants = append(detail.LotSlots[i].Variants, VariantOdds{Share: share, Slots: slots})
	}
	return detail, nil
}

// slotOdds 档位及其掉落池的公示概率, 掉落池不可用时概率为空
func slotOdds(ctx context.Context, slots []Slot) ([]SlotOdds, error) {
	list := make([]SlotOdds, len(slots))
	for i, slot := range slots {
		list[i] = SlotOdds{Slot: slot}
		odds, err := loot.QueryOdds(ctx, slot.Pool)
		if err != nil {
			switch err.Error() {
			case loot.PoolNotFound, loot.PoolDisabled, loot.PoolEmpty:
				continue
			}
			return nil, err
		}
		list[i].Odds = odds
	}
	return list, nil
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
		for _, line := range v.Entry().Currencies {
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
	caps, err := PveProfitCaps(tx, userID, v, time.Now())
	if err != nil {
		return nil, err
	}
	entry := v.Entry()
	view := &VenueView{
		Code:          v.Code,
		Name:          v.Name,
		Description:   v.Description,
		ImageURL:      v.ImageURL,
		PveEnabled:    v.PveEnabled,
		RankedEnabled: v.RankedEnabled,
		BidCurrency:   v.BidCurrency,
		EntryCost:     make([]CostLine, 0, len(entry.Currencies)+len(entry.Items)),
		Requirements:  progress,
		Eligible:      true,
		Grid:          v.Warehouse.Grid,
		Auction:       v.Auction,
		Host:          v.Host,
		ValueRange:    newValueRange(v.Simulation.Ranked),
		Pve: PveView{
			PracticeOnly: v.Pve.PracticeOnly,
			ValueFactor:  v.Pve.ValueFactor,
			ValueRange:   newValueRange(v.Simulation.Pve),
			ProfitCaps:   caps,
		},
		Lots: newLotsView(v),
	}
	for _, p := range progress {
		if !p.Met {
			view.Eligible = false
		}
	}
	for _, line := range entry.Currencies {
		name := names.currencies[line.Currency]
		if name == "" {
			name = line.Currency
		}
		view.EntryCost = append(view.EntryCost, CostLine{Currency: line.Currency, Name: name, Amount: line.Amount})
	}
	for _, line := range entry.Items {
		target := names.items[line.ItemID]
		view.EntryCost = append(view.EntryCost, CostLine{ItemID: line.ItemID, Name: target.Name, ImageURL: target.ImageURL, Amount: line.Quantity})
	}
	return view, nil
}

// newValueRange 取模拟结果中的价值区间
func newValueRange(stats ValueStats) ValueRange {
	return ValueRange{P10: stats.P10, P50: stats.P50, P90: stats.P90}
}
