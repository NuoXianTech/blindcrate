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

package stats

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/fault"
	"github.com/NuoXianTech/blindcrate/internal/apps/params"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/task/schedule"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	params.RegisterSource(SourceMetric, params.Source{Options: metricOptions, First: true})
	params.RegisterSource(SourceScope, params.Source{Options: scopeOptions, Free: true, Default: ScopeAll})
	params.RegisterSource(SourcePeriod, params.Source{Options: periodOptions, Default: PeriodTotal})
}

// metricOptions 可以选的计数指标, 按注册顺序
func metricOptions(_ *gorm.DB) ([]params.Option, error) {
	options := make([]params.Option, len(metrics))
	for i, m := range metrics {
		options[i] = params.Option{Value: m.Key, Label: m.Label}
	}
	return options, nil
}

// ScopeOptions 生成一类统计范围的候选项, 如每种模式、每张地图
type ScopeOptions func(tx *gorm.DB) ([]params.Option, error)

// scopeProviders 统计范围候选项的提供方, 按登记顺序, 只在启动前写入
var scopeProviders []ScopeOptions

// RegisterScopeOptions 登记一类统计范围的候选项, 由定义模式和地图的模块调用, 需要在启动前调用
func RegisterScopeOptions(provider ScopeOptions) {
	scopeProviders = append(scopeProviders, provider)
}

// scopeOptions 统计范围的候选项: 全部对局, 以及各模块登记的; 只是输入建议, 也可以手动填写
func scopeOptions(tx *gorm.DB) ([]params.Option, error) {
	options := []params.Option{{Value: ScopeAll, Label: "全部对局"}}
	for _, provider := range scopeProviders {
		more, err := provider(tx)
		if err != nil {
			return nil, err
		}
		options = append(options, more...)
	}
	return options, nil
}

// ScopeMode 统计范围: 某种模式的对局, 如 mode:ranked
func ScopeMode(mode string) string {
	return "mode:" + mode
}

// ScopeVenue 统计范围: 某张地图的对局, 如 venue:dock
func ScopeVenue(venue string) string {
	return "venue:" + venue
}

// ScopeVenueMode 统计范围: 某张地图某种模式的对局, 如 venue:dock:ranked
func ScopeVenueMode(venue, mode string) string {
	return "venue:" + venue + ":" + mode
}

var (
	// scopePattern 统计范围的格式, 模式和地图代码只能包含小写字母、数字和下划线
	scopePattern = regexp.MustCompile(`^(all|mode:[a-z0-9_]+|venue:[a-z0-9_]+(:[a-z0-9_]+)?)$`)
	// metricPattern 指标的格式, 冒号后为参数, 如 profit:chip
	metricPattern = regexp.MustCompile(`^[a-z0-9_]+(:[a-z0-9_]+)?$`)

	// metrics 已注册的计数指标, 按注册顺序, 只在启动前写入
	metrics = []Metric{
		{Key: MetricMatches, Label: "对局数"},
		{Key: MetricWins, Label: "胜利次数"},
	}
	// periodPattern 周期名称的格式
	periodPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,15}$`)
	// extraPeriods 已注册的额外统计周期, 按注册顺序, 只在启动前写入
	extraPeriods []namedPeriod
)

// namedPeriod 一种注册的额外统计周期
type namedPeriod struct {
	name  string
	label string
	key   PeriodFunc
}

// builtinPeriods 内置周期的名称和展示名称, 按展示的顺序
var builtinPeriods = []params.Option{
	{Value: PeriodTotal, Label: "累计"},
	{Value: PeriodNameDay, Label: "今天"},
	{Value: PeriodNameWeek, Label: "本周"},
}

// periodOptions 可以按名称选择的统计周期: 累计、今天、本周, 以及各模块注册的
func periodOptions(_ *gorm.DB) ([]params.Option, error) {
	options := slices.Clone(builtinPeriods)
	for _, p := range extraPeriods {
		options = append(options, params.Option{Value: p.name, Label: p.label})
	}
	return options, nil
}

// HasPeriod 是否有名称为 name 的周期, 用于校验配置中选择的周期
func HasPeriod(name string) bool {
	if slices.ContainsFunc(builtinPeriods, func(o params.Option) bool { return o.Value == name }) {
		return true
	}
	return slices.ContainsFunc(extraPeriods, func(p namedPeriod) bool { return p.name == name })
}

// PeriodLabel 周期名称对应的展示名称, 如 day 为 "今天"; 名称不存在时返回名称本身
func PeriodLabel(name string) string {
	for _, option := range builtinPeriods {
		if option.Value == name {
			return option.Label
		}
	}
	for _, p := range extraPeriods {
		if p.name == name {
			return p.label
		}
	}
	return name
}

// PeriodKey 名称为 name 的周期在 at 时对应的 stats.period, 如 day 对应 day:2026-10-28, 用于读取配置中按名称选择的周期
//
// 名称不存在时 ok 为 false; 当时没有这种周期(如不在赛季中)时 key 为空, 这时没有任何统计。
func PeriodKey(tx *gorm.DB, name string, at time.Time) (key string, ok bool, err error) {
	switch name {
	case PeriodTotal:
		return PeriodTotal, true, nil
	case PeriodNameDay:
		return PeriodDay(at), true, nil
	case PeriodNameWeek:
		return PeriodWeek(at), true, nil
	}
	for _, p := range extraPeriods {
		if p.name == name {
			key, err = p.key(tx, at)
			return key, true, err
		}
	}
	return "", false, nil
}

// ValidScope 统计范围的格式是否正确, 用于校验配置中填写的范围
func ValidScope(scope string) bool {
	return len(scope) <= maxScopeLength && scopePattern.MatchString(scope)
}

// MatchScopes 一局对局需要累加的全部统计范围
func MatchScopes(venue, mode string) []string {
	return []string{ScopeAll, ScopeMode(mode), ScopeVenue(venue), ScopeVenueMode(venue, mode)}
}

// PeriodDay 统计周期: t 所在的那一天, 如 day:2026-10-28
func PeriodDay(t time.Time) string {
	return periodDayPrefix + t.In(schedule.Location).Format(time.DateOnly)
}

// PeriodWeek 统计周期: t 所在的 ISO 周, 周一开始, 如 week:2026-W44
func PeriodWeek(t time.Time) string {
	year, week := t.In(schedule.Location).ISOWeek()
	return fmt.Sprintf("%s%04d-W%02d", periodWeekPrefix, year, week)
}

// PeriodFunc 返回一种额外的统计周期在 at 时的周期, 如赛季周期 season:s1; 当时没有这种周期(如不在赛季中)时返回空字符串
//
// 结算时在事务中调用, 返回的周期不能超过 stats.period 的长度。
type PeriodFunc func(tx *gorm.DB, at time.Time) (string, error)

// RegisterPeriod 注册一种额外的统计周期, 需要在启动前调用; 名称格式不正确或重复时 panic
//
// Add 时与累计、当天和本周一起写入; 条件等配置中可以按 name 选择它, label 为展示名称(如 "本赛季"), 见 PeriodKey。
func RegisterPeriod(name, label string, fn PeriodFunc) {
	if !periodPattern.MatchString(name) || label == "" || fn == nil {
		panic(fmt.Sprintf("stats: invalid period %q", name))
	}
	if HasPeriod(name) {
		panic(fmt.Sprintf("stats: period %s already registered", name))
	}
	extraPeriods = append(extraPeriods, namedPeriod{name: name, label: label, key: fn})
}

// MetricProfit 指标: 以某种货币计的净收益, 如 profit:chip; 不同货币的收益不能相加, 所以按货币分开
func MetricProfit(currency string) string {
	return profitPrefix + currency
}

// ProfitCurrency 净收益指标的货币代码, 不是净收益指标时返回 false
func ProfitCurrency(metric string) (string, bool) {
	return strings.CutPrefix(metric, profitPrefix)
}

// profitName 净收益指标的展示名称, 如 "净收益(筹码)"
func profitName(currency string) string {
	return "净收益(" + currency + ")"
}

// ProfitMetrics 每种货币的净收益指标, 货币按排序值和代码排列
func ProfitMetrics(tx *gorm.DB) ([]Metric, error) {
	var currencies []wallet.Currency
	if err := tx.Order("sort_order, code").Find(&currencies).Error; err != nil {
		return nil, err
	}
	options := make([]Metric, len(currencies))
	for i, currency := range currencies {
		options[i] = Metric{Key: MetricProfit(currency.Code), Label: profitName(currency.Name)}
	}
	return options, nil
}

// ValueMetrics 可以作为排名或目标的指标: 已注册的计数指标在前, 每种货币的净收益在后
func ValueMetrics(tx *gorm.DB) ([]Metric, error) {
	profits, err := ProfitMetrics(tx)
	if err != nil {
		return nil, err
	}
	return append(Metrics(), profits...), nil
}

// MetricName 计数指标或净收益指标的展示名称; 指标未注册或货币不存在时 ok 为 false
func MetricName(tx *gorm.DB, metric string) (name string, ok bool, err error) {
	if m := FindMetric(metric); m != nil {
		return m.Label, true, nil
	}
	code, isProfit := ProfitCurrency(metric)
	if !isProfit {
		return "", false, nil
	}
	var currency wallet.Currency
	if err := currency.Exact(tx, code); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	return profitName(currency.Name), true, nil
}

// RegisterMetric 注册一个计数指标, 注册后可以在条件中使用, 需要在启动前调用; 格式不正确或重复时 panic
func RegisterMetric(key, label string) {
	if len(key) > maxMetricLength || !metricPattern.MatchString(key) {
		panic(fmt.Sprintf("stats: invalid metric %q", key))
	}
	if FindMetric(key) != nil {
		panic(fmt.Sprintf("stats: metric %s already registered", key))
	}
	metrics = append(metrics, Metric{Key: key, Label: label})
}

// Metrics 全部计数指标, 按注册顺序
func Metrics() []Metric {
	return slices.Clone(metrics)
}

// FindMetric 按 key 查找计数指标, 没有注册时返回 nil
func FindMetric(key string) *Metric {
	for i := range metrics {
		if metrics[i].Key == key {
			return &metrics[i]
		}
	}
	return nil
}

// Values 指标 -> 增量
type Values map[string]int64

// Add 把增量累加到用户在每个统计范围的累计统计、at 当天和当周的统计, 以及注册的额外周期中, 见 RegisterPeriod
//
// 应在结算对局的事务中调用, 与结算一起提交或回滚; 重复的范围只累加一次, 增量为 0 的指标不写入。
func Add(tx *gorm.DB, userID uint64, scopes []string, at time.Time, values Values) error {
	if userID == 0 || len(scopes) == 0 {
		return errors.New("stats: user_id and scopes are required")
	}
	keys := make([]string, 0, len(values))
	for _, key := range slices.Sorted(maps.Keys(values)) {
		if len(key) > maxMetricLength || !metricPattern.MatchString(key) {
			return errors.New("stats: invalid metric " + key)
		}
		if values[key] != 0 {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil
	}

	written := []string{PeriodDay(at), PeriodWeek(at), PeriodTotal}
	for _, p := range extraPeriods {
		period, err := p.key(tx, at)
		if err != nil {
			return err
		}
		if len(period) > maxPeriodLength {
			return errors.New("stats: invalid period " + period)
		}
		if period != "" {
			written = append(written, period)
		}
	}

	// 按主键顺序写入, 同一用户的并发累加按相同顺序加锁, 不会互相死锁
	scopes = slices.Compact(slices.Sorted(slices.Values(scopes)))
	written = slices.Compact(slices.Sorted(slices.Values(written)))
	rows := make([]Stat, 0, len(scopes)*len(written)*len(keys))
	for _, scope := range scopes {
		if scope == "" || len(scope) > maxScopeLength {
			return errors.New("stats: invalid scope " + scope)
		}
		for _, period := range written {
			for _, key := range keys {
				rows = append(rows, Stat{UserID: userID, Scope: scope, Period: period, Metric: key, Value: values[key]})
			}
		}
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "scope"}, {Name: "period"}, {Name: "metric"}},
		DoUpdates: clause.Assignments(map[string]any{
			"value":      gorm.Expr("stats.value + EXCLUDED.value"),
			"updated_at": gorm.Expr("NOW()"),
		}),
	}).Create(&rows).Error
}

// Get 获取用户在某个统计范围和周期的一个指标, 没有记录时返回 0
func Get(tx *gorm.DB, userID uint64, scope, period, metric string) (int64, error) {
	var values []int64
	if err := tx.Model(&Stat{}).
		Where("user_id = ? AND scope = ? AND period = ? AND metric = ?", userID, scope, period, metric).
		Limit(1).Pluck("value", &values).Error; err != nil {
		return 0, err
	}
	if len(values) == 0 {
		return 0, nil
	}
	return values[0], nil
}

// DayValues 一天的全部指标
type DayValues struct {
	Day    string `json:"day"`    // 日期, 如 2026-10-28
	Values Values `json:"values"` // 指标 -> 当天的值, 没有记录的指标不返回
}

// UserStats 用户在一个统计范围的累计和最近几天的数据
type UserStats struct {
	Scope   string      `json:"scope"`
	Metrics []Metric    `json:"metrics"` // 全部计数指标的名称, 按注册顺序; 净收益按货币另外命名, 见 MetricProfit
	Total   Values      `json:"total"`   // 指标 -> 累计值, 没有记录的指标不返回
	Days    []DayValues `json:"days"`    // 最近 RecentDays 天, 从早到晚, 最后一天为今天
}

// QueryUserStats 获取用户在一个统计范围的累计数据和截至 now 的最近 RecentDays 天的数据
func QueryUserStats(tx *gorm.DB, userID uint64, scope string, now time.Time) (*UserStats, error) {
	if !ValidScope(scope) {
		return nil, fault.Invalid(ScopeInvalid)
	}
	result := &UserStats{Scope: scope, Metrics: Metrics(), Total: Values{}, Days: make([]DayValues, RecentDays)}
	periods := []string{PeriodTotal}
	days := make(map[string]*DayValues, RecentDays)
	for i := range result.Days {
		at := now.AddDate(0, 0, i-RecentDays+1)
		period := PeriodDay(at)
		result.Days[i] = DayValues{Day: strings.TrimPrefix(period, periodDayPrefix), Values: Values{}}
		days[period] = &result.Days[i]
		periods = append(periods, period)
	}

	var rows []Stat
	if err := tx.Select("period, metric, value").
		Where("user_id = ? AND scope = ? AND period IN ?", userID, scope, periods).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.Period == PeriodTotal {
			result.Total[row.Metric] = row.Value
		} else if day := days[row.Period]; day != nil {
			day.Values[row.Metric] = row.Value
		}
	}
	return result, nil
}
