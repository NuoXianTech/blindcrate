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
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/ws"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// joinScript 用户不在任何队列中时加入队列
//
// KEYS[1] 队列, KEYS[2] 用户所在的队列; ARGV[1] 用户 ID, ARGV[2] 玩法, ARGV[3] 当前毫秒时间戳。
// 返回 {0, 加入时间}: 新加入; {1, 加入时间}: 已在同一个队列中; {2, 玩法}: 已在其他队列中。
var joinScript = redis.NewScript(`
local current = redis.call('HGET', KEYS[2], ARGV[1])
if current then
	if current ~= ARGV[2] then
		return {2, current}
	end
	return {1, redis.call('ZSCORE', KEYS[1], ARGV[1])}
end
redis.call('HSET', KEYS[2], ARGV[1], ARGV[2])
redis.call('ZADD', KEYS[1], ARGV[3], ARGV[1])
return {0, ARGV[3]}
`)

// leaveScript 用户仍在这个队列中时移出
//
// KEYS[1] 队列, KEYS[2] 用户所在的队列; ARGV[1] 用户 ID, ARGV[2] 玩法。
var leaveScript = redis.NewScript(`
if redis.call('HGET', KEYS[2], ARGV[1]) == ARGV[2] then
	redis.call('HDEL', KEYS[2], ARGV[1])
	redis.call('ZREM', KEYS[1], ARGV[1])
	return 1
end
return 0
`)

// popScript 凑够一局时取出最早加入的 ARGV[1] 人; 不够一局但至少有 ARGV[2] 人且最早加入的人已等到补位时间时全部取出, 由人机补齐
//
// KEYS[1] 队列, KEYS[2] 用户所在的队列; ARGV[1] 每局人数, ARGV[2] 开局至少需要的真人数,
// ARGV[3] 加入时间不晚于它的人可以补位, 为空表示不补位, ARGV[4] 玩法。
// 返回 {用户 ID, 加入时间, ...}。取出和移出队列是原子的, 多个实例同时组局也不会把同一个人放进两局。
var popScript = redis.NewScript(`
local count = tonumber(ARGV[1])
local size = redis.call('ZCARD', KEYS[1])
if size == 0 then
	return {}
end
if size < count then
	if ARGV[3] == '' or size < tonumber(ARGV[2]) then
		return {}
	end
	local oldest = redis.call('ZRANGE', KEYS[1], 0, 0, 'WITHSCORES')
	if tonumber(oldest[2]) > tonumber(ARGV[3]) then
		return {}
	end
end
local popped = redis.call('ZRANGE', KEYS[1], 0, count - 1, 'WITHSCORES')
for i = 1, #popped, 2 do
	redis.call('ZREM', KEYS[1], popped[i])
	if redis.call('HGET', KEYS[2], popped[i]) == ARGV[4] then
		redis.call('HDEL', KEYS[2], popped[i])
	end
end
return popped
`)

// requeueScript 把取出后没能开局的用户按原来的加入时间放回队列, 已在其他队列中的用户跳过
//
// KEYS[1] 队列, KEYS[2] 用户所在的队列; ARGV[1] 玩法, 之后为 {用户 ID, 加入时间, ...}。
var requeueScript = redis.NewScript(`
for i = 2, #ARGV, 2 do
	if redis.call('HSETNX', KEYS[2], ARGV[i], ARGV[1]) == 1 then
		redis.call('ZADD', KEYS[1], ARGV[i + 1], ARGV[i])
	end
end
return 0
`)

// dropScript 取出队列中的所有用户, 用于玩法已不存在的队列
//
// KEYS[1] 队列, KEYS[2] 用户所在的队列; ARGV[1] 玩法。返回取出的用户 ID。
var dropScript = redis.NewScript(`
local popped = redis.call('ZRANGE', KEYS[1], 0, -1)
for _, member in ipairs(popped) do
	redis.call('ZREM', KEYS[1], member)
	if redis.call('HGET', KEYS[2], member) == ARGV[1] then
		redis.call('HDEL', KEYS[2], member)
	end
end
return popped
`)

// queuedUser 从队列中取出的用户
type queuedUser struct {
	UserID   uint64
	JoinedAt int64 // 加入队列的毫秒时间戳
}

// QueueStatus 排队状态
type QueueStatus struct {
	Mode     string    `json:"mode"`
	JoinedAt time.Time `json:"joined_at"`
}

// MatchmakingStatus 用户的匹配状态, 排队和对局最多只有一个
type MatchmakingStatus struct {
	Queue   *QueueStatus `json:"queue"`    // 正在排队的玩法, 没有排队时为空
	MatchID uint64       `json:"match_id"` // 正在进行中的对局, 没有时为 0
}

// MatchmakingFailedMessage MessageMatchmakingFailed 的 data
type MatchmakingFailedMessage struct {
	Mode string `json:"mode"`
}

// MatchmakingRejectedMessage MessageMatchmakingRejected 的 data
type MatchmakingRejectedMessage struct {
	Mode   string `json:"mode"`
	Reason string `json:"reason"` // 不满足的条件, 如余额不足
}

// Join 加入玩法的匹配队列, 凑够一局时立即创建对局, 返回加入后的匹配状态
//
// 已在同一个队列中时不重复加入; 已在其他队列中返回 AlreadyQueued, 已在对局中返回 InMatch;
// 玩法实现了 JoinChecker 且拒绝时返回 *RejectedError。
func Join(ctx context.Context, userID uint64, modeCode string) (*MatchmakingStatus, error) {
	mode, err := lookupMode(ctx, modeCode)
	if err != nil {
		return nil, err
	}
	matchID, err := activeMatchID(db.DB(ctx), userID)
	if err != nil {
		return nil, err
	}
	if matchID != 0 {
		return nil, errors.New(InMatch)
	}
	if checker, ok := mode.(JoinChecker); ok {
		if err := checker.CheckJoin(ctx, userID); err != nil {
			return nil, err
		}
	}

	now := time.Now().UnixMilli()
	result, err := joinScript.Run(ctx, db.Redis, []string{queueKeyPrefix + modeCode, queueUsersKey}, userID, modeCode, now).Slice()
	if err != nil {
		return nil, err
	}
	switch result[0].(int64) {
	case 2:
		return nil, errors.New(AlreadyQueued)
	case 0:
		// 立即补齐时下面的 Form 就会开局, 不需要补位任务
		if fillsWithBots(mode) && mode.BotFillAfter() > 0 {
			if err := enqueueBotFill(ctx, modeCode, userID, now, mode.BotFillAfter()); err != nil {
				logger.ErrorF(ctx, "[Match] enqueue bot fill of %s failed: %v", modeCode, err)
			}
		}
		if err := Form(ctx, modeCode); err != nil {
			logger.ErrorF(ctx, "[Match] form %s failed: %v", modeCode, err)
		}
	}
	return QueryMatchmakingStatus(ctx, userID)
}

// Leave 离开匹配队列, 不在队列中时什么都不做
func Leave(ctx context.Context, userID uint64) error {
	modeCode, err := db.Redis.HGet(ctx, queueUsersKey, strconv.FormatUint(userID, 10)).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	return leaveScript.Run(ctx, db.Redis, []string{queueKeyPrefix + modeCode, queueUsersKey}, userID, modeCode).Err()
}

// QueryMatchmakingStatus 获取用户的匹配状态
func QueryMatchmakingStatus(ctx context.Context, userID uint64) (*MatchmakingStatus, error) {
	status := &MatchmakingStatus{}
	member := strconv.FormatUint(userID, 10)
	modeCode, err := db.Redis.HGet(ctx, queueUsersKey, member).Result()
	switch {
	case errors.Is(err, redis.Nil):
	case err != nil:
		return nil, err
	default:
		score, err := db.Redis.ZScore(ctx, queueKeyPrefix+modeCode, member).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
		if err == nil {
			status.Queue = &QueueStatus{Mode: modeCode, JoinedAt: time.UnixMilli(int64(score))}
		}
	}

	if status.MatchID, err = activeMatchID(db.DB(ctx), userID); err != nil {
		return nil, err
	}
	return status, nil
}

// Form 从玩法的队列中取出玩家创建对局, 直到不够一局且没有人等到补位时间
//
// 加入队列、补位任务到期和巡检时调用, 多个实例同时调用是安全的。
// 创建失败时这一批玩家已被移出队列, 推送 MessageMatchmakingFailed 让他们重新匹配, 避免同一个错误反复发生。
func Form(ctx context.Context, modeCode string) error {
	mode, err := lookupMode(ctx, modeCode)
	if err != nil {
		return err
	}
	count := mode.PlayerCount()
	keys := []string{queueKeyPrefix + modeCode, queueUsersKey}
	for {
		cutoff := ""
		if fillsWithBots(mode) {
			cutoff = strconv.FormatInt(time.Now().Add(-mode.BotFillAfter()).UnixMilli(), 10)
		}
		popped, err := popScript.Run(ctx, db.Redis, keys, count, mode.MinHumans(), cutoff, modeCode).Int64Slice()
		if err != nil {
			return err
		}
		if len(popped) == 0 {
			return nil
		}
		queued := make([]queuedUser, 0, len(popped)/2)
		for i := 0; i < len(popped); i += 2 {
			queued = append(queued, queuedUser{UserID: uint64(popped[i]), JoinedAt: popped[i+1]})
		}

		result, err := createMatch(ctx, mode, queued)
		if err != nil {
			pushMatchmakingFailed(ctx, modeCode, queued)
			return err
		}
		for _, rejected := range result.Rejected {
			if err := ws.SendToUser(ctx, rejected.UserID, MessageMatchmakingRejected, MatchmakingRejectedMessage{Mode: modeCode, Reason: rejected.Reason}); err != nil {
				logger.ErrorF(ctx, "[Match] push matchmaking rejected to %d failed: %v", rejected.UserID, err)
			}
		}
		if result.Match == nil {
			// 剩下的人不够开局, 放回队列等下一次组局; 现在再取一次还是同一批人, 直接返回
			if len(result.Requeue) > 0 {
				if err := requeue(ctx, modeCode, result.Requeue); err != nil {
					pushMatchmakingFailed(ctx, modeCode, result.Requeue)
					return err
				}
			}
			return nil
		}
		afterChange(ctx, nil, result.Match, result.Players)
		if len(queued) < count {
			return nil
		}
	}
}

// dropQueue 玩法已不存在(如地图被删除)时清空它的队列, 推送 MessageMatchmakingFailed 让这些用户重新匹配
func dropQueue(ctx context.Context, modeCode string) error {
	popped, err := dropScript.Run(ctx, db.Redis, []string{queueKeyPrefix + modeCode, queueUsersKey}, modeCode).Int64Slice()
	if err != nil || len(popped) == 0 {
		return err
	}
	users := make([]queuedUser, len(popped))
	for i, userID := range popped {
		users[i] = queuedUser{UserID: uint64(userID)}
	}
	pushMatchmakingFailed(ctx, modeCode, users)
	return nil
}

// fillsWithBots 玩法是否用人机补齐空位
func fillsWithBots(mode Mode) bool {
	return mode.MinHumans() < mode.PlayerCount()
}

// requeue 把用户按原来的加入时间放回队列
//
// 用户在取出后到放回前离开队列时, 离开不会生效, 仍会被放回队列。
func requeue(ctx context.Context, modeCode string, users []queuedUser) error {
	args := make([]interface{}, 0, 1+len(users)*2)
	args = append(args, modeCode)
	for _, user := range users {
		args = append(args, user.UserID, user.JoinedAt)
	}
	return requeueScript.Run(ctx, db.Redis, []string{queueKeyPrefix + modeCode, queueUsersKey}, args...).Err()
}

// pushMatchmakingFailed 通知被移出队列的用户重新匹配
func pushMatchmakingFailed(ctx context.Context, modeCode string, users []queuedUser) {
	userIDs := make([]uint64, len(users))
	for i, user := range users {
		userIDs[i] = user.UserID
	}
	if err := ws.SendToUsers(ctx, userIDs, MessageMatchmakingFailed, MatchmakingFailedMessage{Mode: modeCode}); err != nil {
		logger.ErrorF(ctx, "[Match] push matchmaking failed to %v failed: %v", userIDs, err)
	}
}

// rejectedUser 开局时被玩法拒绝的用户
type rejectedUser struct {
	UserID uint64
	Reason string
}

// formResult 一次组局的结果
type formResult struct {
	Match    *Match // 没有开局时为 nil
	Players  []Player
	Rejected []rejectedUser // 被 Admit 拒绝的用户, 已移出队列
	Requeue  []queuedUser   // 没有开局时需要放回队列的用户
}

// errNotEnough 准入后真人不够 MinHumans, 回滚已创建的对局和准入时的写入
var errNotEnough = errors.New("match: not enough humans")

// createMatch 为取出的玩家创建对局, 不够的座位由人机补齐, 座位随机分配
//
// 已在对局中的玩家被跳过; 每个真人通过 Admit 后才入座, 通过的真人不够 MinHumans 时不创建对局, 需要把这些人放回队列。
func createMatch(ctx context.Context, mode Mode, queued []queuedUser) (*formResult, error) {
	result := &formResult{}
	var admitted []queuedUser
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		userIDs := make([]uint64, len(queued))
		for i, user := range queued {
			userIDs[i] = user.UserID
		}
		// 加入队列时检查过, 这里兜底: 刚被组进一局还没提交时, 同一个人可能又加入了队列
		var busy []uint64
		if err := tx.Model(&Player{}).
			Joins("JOIN matches ON matches.id = match_players.match_id").
			Where("match_players.user_id IN ? AND matches.status = ?", userIDs, StatusPlaying).
			Pluck("match_players.user_id", &busy).Error; err != nil {
			return err
		}
		candidates := make([]queuedUser, 0, len(queued))
		for _, user := range queued {
			if !slices.Contains(busy, user.UserID) {
				candidates = append(candidates, user)
			}
		}
		if len(candidates) < mode.MinHumans() {
			admitted = candidates
			return errNotEnough
		}

		m := &Match{Mode: mode.Code(), Status: StatusPlaying}
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		for _, user := range candidates {
			// 每个人在自己的保存点中准入, 被拒绝时只撤销该玩家的写入
			err := tx.Transaction(func(tx *gorm.DB) error {
				return mode.Admit(tx, m, user.UserID)
			})
			var rejected *RejectedError
			switch {
			case errors.As(err, &rejected):
				result.Rejected = append(result.Rejected, rejectedUser{UserID: user.UserID, Reason: rejected.Reason})
			case err != nil:
				return err
			default:
				admitted = append(admitted, user)
			}
		}
		if len(admitted) < mode.MinHumans() {
			return errNotEnough
		}

		humans := make([]uint64, len(admitted))
		for i, user := range admitted {
			humans[i] = user.UserID
		}
		players, err := seatPlayers(m.ID, humans, mode)
		if err != nil {
			return err
		}
		if err := tx.Create(&players).Error; err != nil {
			return err
		}

		phase, err := mode.Start(tx, m, players)
		if err != nil {
			return err
		}
		if phase == nil {
			return errors.New("match: Start returned nil phase")
		}
		if err := enterPhase(tx, m, phase); err != nil {
			return err
		}
		if err := tx.Where("match_id = ?", m.ID).Order("seat").Find(&players).Error; err != nil {
			return err
		}
		result.Match, result.Players = m, players
		return nil
	})
	if errors.Is(err, errNotEnough) {
		return &formResult{Rejected: result.Rejected, Requeue: admitted}, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// seatPlayers 随机分配座位, 真人之外的座位由随机昵称的人机补齐, 人机等级由玩法决定, 按座位号排序
func seatPlayers(matchID uint64, humans []uint64, mode Mode) ([]Player, error) {
	count := mode.PlayerCount()
	seats := rand.Perm(count)
	names := rand.Perm(len(botNames))
	players := make([]Player, count)
	for i := range count {
		player := Player{MatchID: matchID, Seat: int16(seats[i] + 1)}
		if i < len(humans) {
			player.UserID = humans[i]
		} else {
			player.BotName = botNames[names[(i-len(humans))%len(names)]]
			player.BotLevel = mode.BotLevel()
			if player.BotLevel < BotLevelLow || player.BotLevel > BotLevelHigh {
				return nil, fmt.Errorf("match: invalid bot level %d of mode %s", player.BotLevel, mode.Code())
			}
		}
		players[player.Seat-1] = player
	}
	return players, nil
}
