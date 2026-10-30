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

package leaderboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// codePattern 榜单代码, 小写字母、数字和下划线
var codePattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

// Validate 保存前校验榜单, 名称和描述需已去掉首尾空白
//
// 配置不合法时返回的错误可以直接展示给管理员, 其他错误为查库失败等内部错误。
func Validate(tx *gorm.DB, b *Board) error {
	if !codePattern.MatchString(b.Code) {
		return errors.New(CodeInvalid)
	}
	if n := utf8.RuneCountInString(b.Name); n < 1 || n > 64 {
		return errors.New(NameInvalid)
	}
	if !stats.ValidScope(b.Scope) {
		return errors.New(ScopeInvalid)
	}
	if b.Period != PeriodTotal && b.Period != PeriodDay && b.Period != PeriodSeason {
		return errors.New(PeriodInvalid)
	}
	if b.Size < MinSize || b.Size > MaxSize {
		return errors.New(SizeInvalid)
	}
	if b.MinMatches < 0 {
		return errors.New(MinMatchesInvalid)
	}
	if b.Metric == MetricRating {
		if b.Scope != stats.ScopeAll || b.Period != PeriodSeason {
			return errors.New(RatingBoardInvalid)
		}
		return nil
	}
	_, err := metricName(tx, b.Metric)
	return err
}

// QueryMetrics 可以排名的指标: 已注册的计数指标、段位分和每种货币的净收益
func QueryMetrics(ctx context.Context) ([]stats.Metric, error) {
	profits, err := stats.ProfitMetrics(db.DB(ctx))
	if err != nil {
		return nil, err
	}
	// 段位分排在计数指标之后、净收益之前
	metrics := append(stats.Metrics(), stats.Metric{Key: MetricRating, Label: ratingName})
	return append(metrics, profits...), nil
}

// metricName 指标的展示名称; 指标未注册或货币不存在时返回 MetricInvalid
func metricName(tx *gorm.DB, metric string) (string, error) {
	if metric == MetricRating {
		return ratingName, nil
	}
	name, ok, err := stats.MetricName(tx, metric)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New(MetricInvalid)
	}
	return name, nil
}

// statsPeriod 榜单在 now 时对应的统计周期, 如 total、day:2026-10-28、season:s1, 以及赛季榜单对应的赛季
//
// 还没有赛季时返回的周期为空。
func statsPeriod(tx *gorm.DB, period string, now time.Time) (string, *season.Season, error) {
	switch period {
	case PeriodDay:
		return stats.PeriodDay(now), nil, nil
	case PeriodSeason:
		latest, err := season.Latest(tx, now)
		if err != nil || latest == nil {
			return "", nil, err
		}
		return season.Period(latest.Code), latest, nil
	}
	return stats.PeriodTotal, nil, nil
}

// BoardView 玩家看到的榜单
type BoardView struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Metric      string `json:"metric"`
	MetricName  string `json:"metric_name"` // 指标的展示名称, 如 "胜利次数"
	Currency    string `json:"currency"`    // 指标为某种货币的净收益时为货币代码, 数值按该货币的最小单位计; 否则为空
	Scope       string `json:"scope"`
	ScopeName   string `json:"scope_name"` // 统计范围的展示名称, 全部对局为空
	Period      string `json:"period"`     // total、day 或 season
	Size        int32  `json:"size"`
	MinMatches  int64  `json:"min_matches"`
}

// Entry 榜单上的一个名次
type Entry struct {
	Rank      int64  `json:"rank"`
	Nickname  string `json:"nickname"` // 未设置昵称时为用户名
	AvatarURL string `json:"avatar_url"`
	Value     int64  `json:"value"`
	IsSelf    bool   `json:"is_self"`
}

// Mine 自己在榜单上的情况
type Mine struct {
	Rank  int64 `json:"rank"`  // 名次, 可能在前 N 名之外; 0 表示没有上榜(没有数据、局数不够或账号受限)
	Value int64 `json:"value"` // 自己的数值, 没有上榜也会返回
}

// BoardDetail 榜单的前 N 名和自己的名次
type BoardDetail struct {
	BoardView
	CurrentPeriod string  `json:"current_period"` // 当前的统计周期, 如 total、day:2026-10-28、season:s1; 赛季榜单还没有赛季时为空
	SeasonName    string  `json:"season_name"`    // 赛季榜单对应的赛季名称, 其他榜单为空
	SeasonEnded   bool    `json:"season_ended"`   // 赛季榜单对应的赛季已经结束, 名次不再变化
	Entries       []Entry `json:"entries"`        // 按名次排列, 最多 Size 名
	Mine          Mine    `json:"mine"`
}

// QueryBoards 获取启用的榜单, 按排序值排列
func QueryBoards(ctx context.Context) ([]BoardView, error) {
	tx := db.DB(ctx)
	var boards []Board
	if err := tx.Where("is_enabled = ?", true).Order("sort_order, id").Find(&boards).Error; err != nil {
		return nil, err
	}
	views := make([]BoardView, 0, len(boards))
	for i := range boards {
		view, err := newView(tx, &boards[i])
		if err != nil {
			return nil, err
		}
		views = append(views, *view)
	}
	return views, nil
}

// QueryBoard 获取启用的榜单的前 N 名和用户自己的名次
//
// 前 N 名缓存 CacheTTL, 自己的名次每次查询, 所以两者可能短暂不一致。
func QueryBoard(ctx context.Context, userID uint64, code string) (*BoardDetail, error) {
	tx := db.DB(ctx)
	var board Board
	if err := board.ExactCode(tx, code); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(BoardNotFound)
		}
		return nil, err
	}
	if !board.IsEnabled {
		return nil, errors.New(BoardNotFound)
	}
	view, err := newView(tx, &board)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	period, current, err := statsPeriod(tx, board.Period, now)
	if err != nil {
		return nil, err
	}
	w := window{period: period, season: current}
	detail := &BoardDetail{BoardView: *view, CurrentPeriod: period, Entries: []Entry{}}
	if current != nil {
		detail.SeasonName = current.Name
		detail.SeasonEnded = !now.Before(current.EndsAt)
	}
	if period == "" {
		return detail, nil
	}

	top, err := loadTop(ctx, &board, w)
	if err != nil {
		return nil, err
	}
	detail.Entries = make([]Entry, len(top))
	for i, row := range top {
		detail.Entries[i] = Entry{
			Rank:      int64(i + 1),
			Nickname:  row.Nickname,
			AvatarURL: row.AvatarURL,
			Value:     row.Value,
			IsSelf:    row.UserID == userID,
		}
	}
	if detail.Mine, err = queryMine(tx, &board, w, userID); err != nil {
		return nil, err
	}
	return detail, nil
}

// newView 榜单附带指标和统计范围的展示名称
func newView(tx *gorm.DB, b *Board) (*BoardView, error) {
	name, err := metricName(tx, b.Metric)
	if err != nil {
		// 货币被删除等原因导致指标失效时仍展示榜单, 名称用指标本身
		if err.Error() != MetricInvalid {
			return nil, err
		}
		name = b.Metric
	}
	scopeName, err := condition.ScopeName(tx, b.Scope)
	if err != nil {
		return nil, err
	}
	view := &BoardView{
		Code:        b.Code,
		Name:        b.Name,
		Description: b.Description,
		Metric:      b.Metric,
		MetricName:  name,
		Scope:       b.Scope,
		ScopeName:   scopeName,
		Period:      b.Period,
		Size:        b.Size,
		MinMatches:  b.MinMatches,
	}
	if code, ok := stats.ProfitCurrency(b.Metric); ok {
		view.Currency = code
	}
	return view, nil
}

// rankedRow 能上榜的一行统计, 缓存的也是这个结构
type rankedRow struct {
	UserID    uint64 `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Value     int64  `json:"value"`
}

// window 榜单在某个时刻对应的统计周期
type window struct {
	period string         // stats 的统计周期, 见 statsPeriod
	season *season.Season // 赛季榜单对应的赛季, 段位分榜单按它查询; 其他榜单为空
}

// loadTop 获取前 N 名, 优先读缓存; 缓存读写失败时直接查库, 不影响结果
func loadTop(ctx context.Context, b *Board, w window) ([]rankedRow, error) {
	key := fmt.Sprintf("%s%s:%d:%s", cacheKeyPrefix, b.Code, b.UpdatedAt.UnixMicro(), w.period)
	raw, err := db.Redis.Get(ctx, key).Bytes()
	if err == nil {
		var rows []rankedRow
		if err := json.Unmarshal(raw, &rows); err == nil {
			return rows, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		logger.WarnF(ctx, "[Leaderboard] read cache %s failed: %v", key, err)
	}

	rows, err := queryTop(db.DB(ctx), b, w)
	if err != nil {
		return nil, err
	}
	if raw, err := json.Marshal(rows); err == nil {
		if err := db.Redis.Set(ctx, key, raw, CacheTTL).Err(); err != nil {
			logger.WarnF(ctx, "[Leaderboard] write cache %s failed: %v", key, err)
		}
	}
	return rows, nil
}

// source 榜单的数据, 每个用户一行(user_id、value、updated_at); qualified 为 true 时只保留完成了足够局数的
//
// 段位分榜单读赛季的段位分, 局数为计分的排位局数; 其他榜单读统计范围、周期和指标的统计, 局数为同一范围和周期内的对局数。
func source(tx *gorm.DB, b *Board, w window, qualified bool) *gorm.DB {
	if b.Metric == MetricRating {
		query := tx.Table("ranked_ratings").
			Select("user_id, rating AS value, updated_at").
			Where("season_id = ?", w.season.ID)
		if qualified && b.MinMatches > 0 {
			query = query.Where("matches >= ?", b.MinMatches)
		}
		return query
	}
	query := tx.Table("stats").
		Select("user_id, value, updated_at").
		Where("scope = ? AND period = ? AND metric = ?", b.Scope, w.period, b.Metric)
	if qualified && b.MinMatches > 0 {
		query = query.Where(
			"EXISTS (SELECT 1 FROM stats AS played WHERE played.user_id = stats.user_id AND played.scope = stats.scope "+
				"AND played.period = stats.period AND played.metric = ? AND played.value >= ?)",
			stats.MetricMatches, b.MinMatches,
		)
	}
	return query
}

// rankedQuery 能上榜的行, 别名为 r: 完成了足够的局数, 账号未停用、未封禁
func rankedQuery(tx *gorm.DB, b *Board, w window) *gorm.DB {
	return tx.Table("(?) AS r", source(tx, b, w, true)).
		Joins("JOIN users ON users.id = r.user_id").
		Where("users.is_disabled = FALSE AND (users.banned_until IS NULL OR users.banned_until <= NOW())")
}

// queryTop 查询前 N 名, 按数值从大到小, 同分时先达到的在前
func queryTop(tx *gorm.DB, b *Board, w window) ([]rankedRow, error) {
	var rows []struct {
		UserID    uint64
		Username  string
		Nickname  string
		AvatarURL string
		Value     int64
	}
	if err := rankedQuery(tx, b, w).
		Select("r.user_id, r.value, users.username, users.nickname, users.avatar_url").
		Order("r.value DESC, r.updated_at, r.user_id").
		Limit(int(b.Size)).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]rankedRow, len(rows))
	for i, row := range rows {
		result[i] = rankedRow{UserID: row.UserID, Nickname: row.Nickname, AvatarURL: row.AvatarURL, Value: row.Value}
		if result[i].Nickname == "" {
			result[i].Nickname = row.Username
		}
	}
	return result, nil
}

// queryMine 查询用户自己的数值和名次, 名次为排在自己前面的人数 + 1, 与 queryTop 的排序一致
func queryMine(tx *gorm.DB, b *Board, w window, userID uint64) (Mine, error) {
	var rows []struct {
		Value     int64
		UpdatedAt time.Time
	}
	if err := rankedQuery(tx, b, w).
		Select("r.value, r.updated_at").
		Where("r.user_id = ?", userID).
		Limit(1).Scan(&rows).Error; err != nil {
		return Mine{}, err
	}
	if len(rows) == 0 {
		// 没有上榜时仍返回自己的数值
		err := source(tx, b, w, false).Where("user_id = ?", userID).Limit(1).Scan(&rows).Error
		if err != nil || len(rows) == 0 {
			return Mine{}, err
		}
		return Mine{Value: rows[0].Value}, nil
	}

	mine := rows[0]
	var ahead int64
	if err := rankedQuery(tx, b, w).
		Where("(r.value > ? OR (r.value = ? AND (r.updated_at < ? OR (r.updated_at = ? AND r.user_id < ?))))",
			mine.Value, mine.Value, mine.UpdatedAt, mine.UpdatedAt, userID).
		Count(&ahead).Error; err != nil {
		return Mine{}, err
	}
	return Mine{Rank: ahead + 1, Value: mine.Value}, nil
}
