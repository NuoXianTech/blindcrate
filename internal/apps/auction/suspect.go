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
	"context"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// suspectPairsSQL 最近若干天已结束的排位中每一对真人的同局统计, 只统计暗拍的座位
//
// 成交者是两人之一时算作其中一人成交, 成交价低于仓库价值的 @rate 基点时算作低价成交。
const suspectPairsSQL = `
WITH humans AS (
	SELECT ap.match_id, ap.user_id, ap.seat
	FROM auction_players ap
	JOIN matches m ON m.id = ap.match_id
	WHERE m.mode LIKE @mode AND m.status = @status AND m.created_at >= @since AND ap.user_id <> 0
), pairs AS (
	SELECT a.user_id AS user_a, b.user_id AS user_b,
		COUNT(*) AS matches,
		COUNT(*) FILTER (WHERE am.outcome = @sold AND am.winner_seat IN (a.seat, b.seat)) AS wins,
		COUNT(*) FILTER (WHERE am.outcome = @sold AND am.winner_seat IN (a.seat, b.seat) AND am.price * 10000 < am.value * @rate) AS low_wins,
		COALESCE(AVG(am.price * 10000 / NULLIF(am.value, 0)) FILTER (WHERE am.outcome = @sold AND am.winner_seat IN (a.seat, b.seat)), 0)::BIGINT AS price_rate,
		MAX(a.match_id) AS last_match_id
	FROM humans a
	JOIN humans b ON b.match_id = a.match_id AND b.user_id > a.user_id
	JOIN auction_matches am ON am.match_id = a.match_id
	GROUP BY a.user_id, b.user_id
)
SELECT p.*, m.created_at AS last_match_at
FROM pairs p
JOIN matches m ON m.id = p.last_match_id
LEFT JOIN auction_suspect_reviews r ON r.user_a = p.user_a AND r.user_b = p.user_b
WHERE (p.matches >= @min_matches OR p.low_wins >= @min_low_wins)`

// SuspectUser 可疑名单中的一个账号
type SuspectUser struct {
	ID          uint64     `json:"id"`
	Username    string     `json:"username"`
	Nickname    string     `json:"nickname"`
	IsDisabled  bool       `json:"is_disabled"`
	BannedUntil *time.Time `json:"banned_until"`
}

// Suspect 可疑名单中的一对账号: 最近若干天一起打的排位较多, 或同局时其中一人多次低价成交, 天数和次数见排位设置
type Suspect struct {
	UserA       SuspectUser    `json:"user_a"` // 用户 ID 较小的一个
	UserB       SuspectUser    `json:"user_b"`
	Matches     int64          `json:"matches"`    // 一起打的排位局数
	Wins        int64          `json:"wins"`       // 其中一人成交的局数
	LowWins     int64          `json:"low_wins"`   // 其中一人低价成交的局数
	PriceRate   int64          `json:"price_rate"` // 其中一人成交时成交价占仓库价值的平均比例, 基点; 没有成交时为 0
	LastMatchID uint64         `json:"last_match_id"`
	LastMatchAt time.Time      `json:"last_match_at"`
	Review      *SuspectReview `json:"review"` // 最近一次处理记录, 没处理过时为空
}

// suspectRow 可疑名单查询结果的一行
type suspectRow struct {
	UserA       uint64
	UserB       uint64
	Matches     int64
	Wins        int64
	LowWins     int64
	PriceRate   int64
	LastMatchID uint64
	LastMatchAt time.Time
}

// QuerySuspects 分页获取可疑名单, 其中一人低价成交多的排在前面
//
// reviewed 为空时返回全部; false 只返回待处理的, 即没处理过或处理后又一起打过排位; true 只返回其余的。
func QuerySuspects(ctx context.Context, reviewed *bool, offset, limit int) (int64, []Suspect, error) {
	cfg, err := settings.Ranked.Get(ctx)
	if err != nil {
		return 0, nil, err
	}
	query := suspectPairsSQL
	if reviewed != nil {
		if *reviewed {
			query += " AND r.last_match_id >= p.last_match_id"
		} else {
			query += " AND (r.last_match_id IS NULL OR r.last_match_id < p.last_match_id)"
		}
	}
	args := map[string]any{
		"mode":         venue.ModeRanked + ":%",
		"status":       match.StatusFinished,
		"since":        time.Now().AddDate(0, 0, -cfg.WatchDays),
		"sold":         OutcomeSold,
		"rate":         cfg.WatchLowRate,
		"min_matches":  cfg.WatchMatches,
		"min_low_wins": cfg.WatchLowWins,
		"offset":       offset,
		"limit":        limit,
	}

	tx := db.DB(ctx)
	var total int64
	if err := tx.Raw("SELECT COUNT(*) FROM ("+query+") suspects", args).Scan(&total).Error; err != nil {
		return 0, nil, err
	}
	var rows []suspectRow
	if err := tx.Raw(query+" ORDER BY p.low_wins DESC, p.matches DESC, p.last_match_id DESC LIMIT @limit OFFSET @offset", args).
		Scan(&rows).Error; err != nil {
		return 0, nil, err
	}
	suspects, err := loadSuspects(tx, rows)
	if err != nil {
		return 0, nil, err
	}
	return total, suspects, nil
}

// loadSuspects 补上账号信息和处理记录
func loadSuspects(tx *gorm.DB, rows []suspectRow) ([]Suspect, error) {
	suspects := make([]Suspect, len(rows))
	if len(rows) == 0 {
		return suspects, nil
	}
	userIDs := make([]uint64, 0, len(rows)*2)
	pairs := make([][]any, len(rows))
	for i, row := range rows {
		userIDs = append(userIDs, row.UserA, row.UserB)
		pairs[i] = []any{row.UserA, row.UserB}
	}
	var users []auth.User
	if err := tx.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	byID := make(map[uint64]SuspectUser, len(users))
	for _, u := range users {
		byID[u.ID] = SuspectUser{ID: u.ID, Username: u.Username, Nickname: u.Nickname, IsDisabled: u.IsDisabled, BannedUntil: u.BannedUntil}
	}
	var reviews []SuspectReview
	if err := tx.Where("(user_a, user_b) IN ?", pairs).Find(&reviews).Error; err != nil {
		return nil, err
	}
	type pair struct{ a, b uint64 }
	reviewOf := make(map[pair]*SuspectReview, len(reviews))
	for i := range reviews {
		reviewOf[pair{reviews[i].UserA, reviews[i].UserB}] = &reviews[i]
	}

	for i, row := range rows {
		suspects[i] = Suspect{
			UserA:       byID[row.UserA],
			UserB:       byID[row.UserB],
			Matches:     row.Matches,
			Wins:        row.Wins,
			LowWins:     row.LowWins,
			PriceRate:   row.PriceRate,
			LastMatchID: row.LastMatchID,
			LastMatchAt: row.LastMatchAt,
			Review:      reviewOf[pair{row.UserA, row.UserB}],
		}
	}
	return suspects, nil
}

// ReviewSuspect 把一对可疑账号标记为已处理, 处理到 lastMatchID 这一局; 已处理过时覆盖上一次的记录
//
// userA 必须小于 userB; 之后两人又一起打了排位时重新进入待处理。
func ReviewSuspect(ctx context.Context, operatorID, userA, userB, lastMatchID uint64, note string) (*SuspectReview, error) {
	review := &SuspectReview{UserA: userA, UserB: userB, LastMatchID: lastMatchID, Note: note, ReviewedBy: operatorID}
	if err := db.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_a"}, {Name: "user_b"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_match_id", "note", "reviewed_by", "updated_at"}),
	}, clause.Returning{}).Create(review).Error; err != nil {
		return nil, err
	}
	return review, nil
}
