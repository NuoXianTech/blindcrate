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

// MatchScopes 一局对局需要累加的全部统计范围
func MatchScopes(venue, mode string) []string {
	return []string{ScopeAll, ScopeMode(mode), ScopeVenue(venue), ScopeVenueMode(venue, mode)}
}

// PeriodDay 统计周期: t 所在的那一天, 如 day:2026-10-28
func PeriodDay(t time.Time) string {
	return "day:" + t.In(schedule.Location).Format(time.DateOnly)
}

// Delta 一局对局给一个玩家带来的统计增量
type Delta struct {
	Matches int64
	Wins    int64
	Profit  int64
	BidWon  int64
}

// Add 把增量累加到用户在每个统计范围的累计统计和 at 当天的统计中
//
// 应在结算对局的事务中调用, 与结算一起提交或回滚; 重复的范围只累加一次, 增量全为 0 时什么都不做。
func Add(tx *gorm.DB, userID uint64, scopes []string, at time.Time, delta Delta) error {
	if userID == 0 || len(scopes) == 0 {
		return errors.New("stats: user_id and scopes are required")
	}
	if delta == (Delta{}) {
		return nil
	}
	// 按主键顺序写入, 同一用户的并发累加按相同顺序加锁, 不会互相死锁
	scopes = slices.Compact(slices.Sorted(slices.Values(scopes)))
	periods := []string{PeriodDay(at), PeriodTotal}
	rows := make([]UserStat, 0, len(scopes)*len(periods))
	for _, scope := range scopes {
		if scope == "" || len(scope) > maxScopeLength {
			return errors.New("stats: invalid scope " + scope)
		}
		for _, period := range periods {
			rows = append(rows, UserStat{
				UserID:  userID,
				Scope:   scope,
				Period:  period,
				Matches: delta.Matches,
				Wins:    delta.Wins,
				Profit:  delta.Profit,
				BidWon:  delta.BidWon,
			})
		}
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "scope"}, {Name: "period"}},
		DoUpdates: clause.Assignments(map[string]any{
			"matches":    gorm.Expr("user_stats.matches + EXCLUDED.matches"),
			"wins":       gorm.Expr("user_stats.wins + EXCLUDED.wins"),
			"profit":     gorm.Expr("user_stats.profit + EXCLUDED.profit"),
			"bid_won":    gorm.Expr("user_stats.bid_won + EXCLUDED.bid_won"),
			"updated_at": gorm.Expr("NOW()"),
		}),
	}).Create(&rows).Error
}

// Get 获取用户在某个统计范围和周期的统计, 没有记录时返回全为 0 的统计
func Get(tx *gorm.DB, userID uint64, scope, period string) (*UserStat, error) {
	var stat UserStat
	// 没有记录是常态, 用 Find 避免每次都打 record not found 日志
	result := tx.Where("user_id = ? AND scope = ? AND period = ?", userID, scope, period).Limit(1).Find(&stat)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		stat = UserStat{UserID: userID, Scope: scope, Period: period}
	}
	return &stat, nil
}
