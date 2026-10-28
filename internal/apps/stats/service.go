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
	"time"

	"github.com/NuoXianTech/blindcrate/internal/task/schedule"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
)

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
	return "day:" + t.In(schedule.Location).Format(time.DateOnly)
}

// MetricProfit 指标: 以某种货币计的净收益, 如 profit:chip; 不同货币的收益不能相加, 所以按货币分开
func MetricProfit(currency string) string {
	return "profit:" + currency
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

// Add 把增量累加到用户在每个统计范围的累计统计和 at 当天的统计中
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

	// 按主键顺序写入, 同一用户的并发累加按相同顺序加锁, 不会互相死锁
	scopes = slices.Compact(slices.Sorted(slices.Values(scopes)))
	periods := []string{PeriodDay(at), PeriodTotal}
	rows := make([]Stat, 0, len(scopes)*len(periods)*len(keys))
	for _, scope := range scopes {
		if scope == "" || len(scope) > maxScopeLength {
			return errors.New("stats: invalid scope " + scope)
		}
		for _, period := range periods {
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
