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

package match

import (
	"context"
	"errors"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdvanceFunc 在锁定对局的事务内执行, 返回非 nil 的阶段时切换到该阶段
type AdvanceFunc func(tx *gorm.DB, m *Match, players []Player) (*Phase, error)

// Advance 锁定进行中的对局后执行 fn, 玩家操作和阶段超时都通过它推进对局
//
// fn 返回错误时整个事务回滚; 返回 nil 阶段时只提交 fn 自己的写入, 不切换阶段。
// 对局不存在时返回 MatchNotFound, 不在进行中时返回 MatchNotPlaying, 都不调用 fn。
// 同一局的 Advance 在对局行锁上排队, fn 中看到的总是最新状态。
// 切换阶段后, 提交后设置新阶段的超时任务并推送 MessageUpdated; 推送和设置超时失败只记录日志, 超时任务由巡检补上。
func Advance(ctx context.Context, matchID uint64, fn AdvanceFunc) (*Match, error) {
	var (
		m        Match
		players  []Player
		previous Match
		changed  bool
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockMatch(tx, &m, matchID); err != nil {
			return err
		}
		if m.Status != StatusPlaying {
			return errors.New(MatchNotPlaying)
		}
		if err := tx.Where("match_id = ?", matchID).Order("seat").Find(&players).Error; err != nil {
			return err
		}

		previous = m
		phase, err := fn(tx, &m, players)
		if err != nil {
			return err
		}
		if phase == nil {
			return nil
		}
		changed = true
		if err := enterPhase(tx, &m, phase); err != nil {
			return err
		}
		// 钩子可能写入了名次, 推送前重新读取
		return tx.Where("match_id = ?", matchID).Order("seat").Find(&players).Error
	})
	if err != nil {
		return nil, err
	}
	if changed {
		afterChange(ctx, &previous, &m, players)
	}
	return &m, nil
}

// Cancel 取消进行中的对局, 用于对局卡住等异常情况, 返回取消后的对局
//
// 玩法已不再注册时不调用 OnCancel, 入场费等需要手动处理。
func Cancel(ctx context.Context, matchID uint64) (*Match, error) {
	var (
		m        Match
		players  []Player
		previous Match
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockMatch(tx, &m, matchID); err != nil {
			return err
		}
		if m.Status != StatusPlaying {
			return errors.New(MatchNotPlaying)
		}
		if err := tx.Where("match_id = ?", matchID).Order("seat").Find(&players).Error; err != nil {
			return err
		}
		mode, err := lookupMode(ctx, m.Mode)
		switch {
		case err != nil && err.Error() == ModeNotFound:
			logger.WarnF(ctx, "[Match] mode %s of match %d is not registered, cancel without OnCancel", m.Mode, m.ID)
		case err != nil:
			return err
		default:
			if err := mode.OnCancel(tx, &m, players); err != nil {
				return err
			}
		}

		previous = m
		now := time.Now()
		m.Status = StatusCancelled
		m.PhaseEndsAt = nil
		m.FinishedAt = &now
		m.Version++
		return tx.Model(&m).Updates(map[string]interface{}{
			"status":        m.Status,
			"phase_ends_at": nil,
			"finished_at":   now,
			"version":       m.Version,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	afterChange(ctx, &previous, &m, players)
	return &m, nil
}

// SetRanks 写入名次, 在钩子中对局结束前调用, ranks 为座位号 -> 名次
func SetRanks(tx *gorm.DB, matchID uint64, ranks map[int16]int16) error {
	for seat, rank := range ranks {
		if err := tx.Model(&Player{}).Where("match_id = ? AND seat = ?", matchID, seat).Update("rank", rank).Error; err != nil {
			return err
		}
	}
	return nil
}

// lockMatch 锁定对局, 同一局的推进、超时和取消互斥
func lockMatch(tx *gorm.DB, m *Match, matchID uint64) error {
	if err := m.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), matchID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(MatchNotFound)
		}
		return err
	}
	return nil
}

// enterPhase 切换到下一个阶段并写库, phase.Finish 时结束对局
func enterPhase(tx *gorm.DB, m *Match, phase *Phase) error {
	now := time.Now()
	m.Version++
	updates := map[string]interface{}{"version": m.Version}
	if phase.Finish {
		m.Status = StatusFinished
		m.PhaseEndsAt = nil
		m.FinishedAt = &now
		updates["status"] = m.Status
		updates["phase_ends_at"] = nil
		updates["finished_at"] = now
	} else {
		if phase.Name == "" || len(phase.Name) > 32 || phase.Duration < 0 {
			return errors.New("match: invalid phase")
		}
		m.Phase = phase.Name
		m.Round = phase.Round
		m.PhaseEndsAt = nil
		if phase.Duration > 0 {
			endsAt := now.Add(phase.Duration)
			m.PhaseEndsAt = &endsAt
		}
		updates["phase"] = m.Phase
		updates["round"] = m.Round
		updates["phase_ends_at"] = m.PhaseEndsAt
	}
	return tx.Model(m).Updates(updates).Error
}

// afterChange 对局状态变化并提交后, 取消上一阶段未到期的超时任务, 设置新阶段的超时任务并推送
func afterChange(ctx context.Context, previous, m *Match, players []Player) {
	// 上一阶段已到期时超时任务就是当前正在执行的任务, 不需要也无法取消
	if previous != nil && previous.PhaseEndsAt != nil && time.Now().Before(*previous.PhaseEndsAt) {
		cancelPhaseTimeout(ctx, previous)
	}
	if m.Status == StatusPlaying && m.PhaseEndsAt != nil {
		if err := enqueuePhaseTimeout(ctx, m); err != nil {
			logger.ErrorF(ctx, "[Match] enqueue phase timeout of match %d failed: %v", m.ID, err)
		}
	}
	pushUpdate(ctx, m, players)
}

// PlayerDetail 管理后台看到的座位, 附带真人的昵称和头像
type PlayerDetail struct {
	Player
	IsBot     bool   `json:"is_bot"`
	Nickname  string `json:"nickname"` // 真人的昵称, 未设置时为用户名; 人机为人机的昵称
	AvatarURL string `json:"avatar_url"`
}

// MatchDetail 对局及其全部座位, 按座位号排序
type MatchDetail struct {
	Match
	Players []PlayerDetail `json:"players"`
}

// LoadDetails 补全对局的座位和真人的昵称, 保持 matches 的顺序
func LoadDetails(tx *gorm.DB, matches []Match) ([]MatchDetail, error) {
	details := make([]MatchDetail, len(matches))
	if len(matches) == 0 {
		return details, nil
	}
	byID := make(map[uint64]*MatchDetail, len(matches))
	ids := make([]uint64, len(matches))
	for i := range matches {
		details[i] = MatchDetail{Match: matches[i], Players: []PlayerDetail{}}
		byID[matches[i].ID] = &details[i]
		ids[i] = matches[i].ID
	}

	var players []Player
	if err := tx.Where("match_id IN ?", ids).Order("match_id, seat").Find(&players).Error; err != nil {
		return nil, err
	}
	profiles, err := loadProfiles(tx, players)
	if err != nil {
		return nil, err
	}
	for _, player := range players {
		detail := byID[player.MatchID]
		detail.Players = append(detail.Players, newPlayerDetail(player, profiles))
	}
	return details, nil
}

// loadProfiles 获取座位上真人的用户信息
func loadProfiles(tx *gorm.DB, players []Player) (map[uint64]auth.User, error) {
	var userIDs []uint64
	for _, player := range players {
		if !player.IsBot() {
			userIDs = append(userIDs, player.UserID)
		}
	}
	profiles := make(map[uint64]auth.User, len(userIDs))
	if len(userIDs) == 0 {
		return profiles, nil
	}
	var users []auth.User
	if err := tx.Select("id", "username", "nickname", "avatar_url").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, user := range users {
		profiles[user.ID] = user
	}
	return profiles, nil
}

// newPlayerDetail 座位附带昵称和头像
func newPlayerDetail(player Player, profiles map[uint64]auth.User) PlayerDetail {
	detail := PlayerDetail{Player: player, IsBot: player.IsBot()}
	if detail.IsBot {
		detail.Nickname = player.BotName
		return detail
	}
	profile := profiles[player.UserID]
	detail.Nickname = profile.Nickname
	if detail.Nickname == "" {
		detail.Nickname = profile.Username
	}
	detail.AvatarURL = profile.AvatarURL
	return detail
}

// PlayerView 玩家看到的座位, 不区分真人和人机, 也不暴露用户 ID
type PlayerView struct {
	Seat      int16  `json:"seat"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Rank      int16  `json:"rank"`    // 结束时的名次, 0 表示没有名次
	IsSelf    bool   `json:"is_self"` // 是否为当前用户
}

// MatchView 玩家看到的对局
type MatchView struct {
	ID          uint64       `json:"id"`
	Mode        string       `json:"mode"`
	Status      string       `json:"status"`
	Phase       string       `json:"phase"`
	Round       int32        `json:"round"`
	PhaseEndsAt *time.Time   `json:"phase_ends_at"` // 当前阶段的截止时间, 不限时或已结束时为空
	Version     int64        `json:"version"`       // 越大越新, 客户端只采用 version 更大的推送
	Players     []PlayerView `json:"players"`       // 按座位号排序
	CreatedAt   time.Time    `json:"created_at"`
	FinishedAt  *time.Time   `json:"finished_at"`
}

// View 以 viewerID 的视角查看对局
func (d *MatchDetail) View(viewerID uint64) MatchView {
	view := MatchView{
		ID:          d.ID,
		Mode:        d.Mode,
		Status:      d.Status,
		Phase:       d.Phase,
		Round:       d.Round,
		PhaseEndsAt: d.PhaseEndsAt,
		Version:     d.Version,
		Players:     make([]PlayerView, len(d.Players)),
		CreatedAt:   d.CreatedAt,
		FinishedAt:  d.FinishedAt,
	}
	for i, player := range d.Players {
		view.Players[i] = PlayerView{
			Seat:      player.Seat,
			Nickname:  player.Nickname,
			AvatarURL: player.AvatarURL,
			Rank:      player.Rank,
			IsSelf:    !player.IsBot && player.UserID == viewerID,
		}
	}
	return view
}

// activeMatchID 用户正在进行中的对局, 没有时为 0
func activeMatchID(tx *gorm.DB, userID uint64) (uint64, error) {
	var ids []uint64
	if err := tx.Model(&Player{}).
		Joins("JOIN matches ON matches.id = match_players.match_id").
		Where("match_players.user_id = ? AND matches.status = ?", userID, StatusPlaying).
		Order("match_players.match_id DESC").Limit(1).
		Pluck("match_players.match_id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	return ids[0], nil
}

// QueryCurrentMatch 获取用户正在进行中的对局, 没有时返回 nil
func QueryCurrentMatch(ctx context.Context, userID uint64) (*MatchView, error) {
	matchID, err := activeMatchID(db.DB(ctx), userID)
	if err != nil || matchID == 0 {
		return nil, err
	}
	return QueryUserMatch(ctx, userID, matchID)
}

// QueryUserMatch 获取用户参与过的一局对局, 不是该局的玩家时返回 MatchNotFound
func QueryUserMatch(ctx context.Context, userID, matchID uint64) (*MatchView, error) {
	var matches []Match
	if err := db.DB(ctx).
		Where("id = ? AND id IN (SELECT match_id FROM match_players WHERE user_id = ?)", matchID, userID).
		Limit(1).Find(&matches).Error; err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, errors.New(MatchNotFound)
	}
	details, err := LoadDetails(db.DB(ctx), matches)
	if err != nil {
		return nil, err
	}
	view := details[0].View(userID)
	return &view, nil
}

// QueryUserMatches 获取用户参与过的对局, 按开始时间从新到旧
func QueryUserMatches(ctx context.Context, userID uint64, req *ListMatchesRequest) (int64, []MatchView, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&Match{}).
		Where("id IN (SELECT match_id FROM match_players WHERE user_id = ?)", userID)
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var matches []Match
	if err := query.Order("id DESC").Offset(offset).Limit(req.Size).Find(&matches).Error; err != nil {
		return 0, nil, err
	}
	details, err := LoadDetails(db.DB(ctx), matches)
	if err != nil {
		return 0, nil, err
	}
	views := make([]MatchView, len(details))
	for i := range details {
		views[i] = details[i].View(userID)
	}
	return total, views, nil
}
