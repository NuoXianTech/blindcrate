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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/task"
	"github.com/NuoXianTech/blindcrate/internal/task/schedule"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

func init() {
	task.Register(task.Job{Type: task.TypeMatchPhaseTimeout, Handler: HandlePhaseTimeout})
	task.Register(task.Job{Type: task.TypeMatchBotFill, Handler: HandleBotFill})
	// 巡检到期未推进的对局和排队, 失败时等下一次巡检, 不重试
	task.Register(task.Job{Type: task.TypeMatchSweep, Handler: HandleSweep, Cron: config.Config.Schedule.MatchSweep, Periodic: true, Required: true})
}

// phaseTimeoutPayload 阶段超时任务的参数
type phaseTimeoutPayload struct {
	MatchID uint64 `json:"match_id"`
	Version int64  `json:"version"` // 设置超时时对局的版本, 对局已切换阶段时任务作废
}

// botFillPayload 人机补位任务的参数
type botFillPayload struct {
	Mode string `json:"mode"`
}

// phaseTimeoutTaskID 每个阶段一个超时任务, 版本区分同一局的不同阶段
func phaseTimeoutTaskID(m *Match) string {
	return fmt.Sprintf("match:timeout:%d:%d", m.ID, m.Version)
}

// ceilSecond asynq 按秒记录定时任务的执行时间并向下取整, 向上取整到秒才能保证任务不会提前执行
func ceilSecond(t time.Time) time.Time {
	ceil := t.Truncate(time.Second)
	if ceil.Before(t) {
		ceil = ceil.Add(time.Second)
	}
	return ceil
}

// enqueuePhaseTimeout 投递当前阶段的超时任务, 到期时执行; 已到期时立即执行
func enqueuePhaseTimeout(ctx context.Context, m *Match) error {
	payload, err := json.Marshal(phaseTimeoutPayload{MatchID: m.ID, Version: m.Version})
	if err != nil {
		return err
	}
	return schedule.EnqueueOnce(
		ctx,
		asynq.NewTask(task.TypeMatchPhaseTimeout, payload),
		phaseTimeoutTaskID(m),
		asynq.Queue(task.QueueCritical),
		asynq.ProcessAt(ceilSecond(*m.PhaseEndsAt)),
	)
}

// cancelPhaseTimeout 阶段提前结束时删除它的超时任务, 失败只记录日志, 任务执行时会因版本不符而作废
func cancelPhaseTimeout(ctx context.Context, m *Match) {
	if err := schedule.Cancel(task.QueueCritical, phaseTimeoutTaskID(m)); err != nil {
		logger.WarnF(ctx, "[Match] cancel phase timeout of match %d failed: %v", m.ID, err)
	}
}

// enqueueBotFill 投递补位任务, 在用户等满 delay 时组局; 每次加入队列一个任务, joinedAt 为加入队列的毫秒时间戳
func enqueueBotFill(ctx context.Context, modeCode string, userID uint64, joinedAt int64, delay time.Duration) error {
	payload, err := json.Marshal(botFillPayload{Mode: modeCode})
	if err != nil {
		return err
	}
	return schedule.EnqueueOnce(
		ctx,
		asynq.NewTask(task.TypeMatchBotFill, payload),
		fmt.Sprintf("match:bot_fill:%s:%d:%d", modeCode, userID, joinedAt),
		asynq.ProcessAt(ceilSecond(time.UnixMilli(joinedAt).Add(delay))),
	)
}

// HandlePhaseTimeout 阶段到期, 调用玩法的 OnTimeout 进入下一个阶段; 对局已切换阶段或已结束时什么都不做
func HandlePhaseTimeout(ctx context.Context, t *asynq.Task) error {
	var payload phaseTimeoutPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("decode payload failed: %v: %w", err, asynq.SkipRetry)
	}
	_, err := Advance(ctx, payload.MatchID, func(tx *gorm.DB, m *Match, players []Player) (*Phase, error) {
		if m.Version != payload.Version {
			return nil, nil
		}
		mode, err := lookupMode(ctx, m.Mode)
		if err != nil {
			return nil, fmt.Errorf("lookup mode %s failed: %w", m.Mode, err)
		}
		phase, err := mode.OnTimeout(tx, m, players)
		if err != nil {
			return nil, err
		}
		if phase == nil {
			return nil, errors.New("OnTimeout returned nil phase")
		}
		return phase, nil
	})
	if err != nil && (err.Error() == MatchNotFound || err.Error() == MatchNotPlaying) {
		return nil
	}
	return err
}

// HandleBotFill 补位时间到, 为等满时间的玩家组局
func HandleBotFill(ctx context.Context, t *asynq.Task) error {
	var payload botFillPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("decode payload failed: %v: %w", err, asynq.SkipRetry)
	}
	return Form(ctx, payload.Mode)
}

// HandleSweep 巡检: 为到期仍在进行中的对局重新投递超时任务, 并为每个有人排队的玩法组局
//
// 兜底投递失败、Redis 故障等原因丢失的超时和补位任务; 超时任务还在队列中时 EnqueueOnce 不会重复投递。
// 队列从 Redis 中扫描, 动态玩法不需要预先知道代码; 玩法已不存在的队列被清空。
func HandleSweep(ctx context.Context, _ *asynq.Task) error {
	var overdue []Match
	if err := db.DB(ctx).
		Where("status = ? AND phase_ends_at < ?", StatusPlaying, time.Now().Add(-timeoutGrace)).
		Order("phase_ends_at").Limit(sweepBatch).
		Find(&overdue).Error; err != nil {
		return err
	}

	var errs []error
	for i := range overdue {
		if err := enqueuePhaseTimeout(ctx, &overdue[i]); err != nil {
			errs = append(errs, fmt.Errorf("enqueue phase timeout of match %d failed: %w", overdue[i].ID, err))
		}
	}
	codes, err := queuedModeCodes(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("scan queues failed: %w", err))
	}
	for _, code := range codes {
		err := Form(ctx, code)
		if err != nil && err.Error() == ModeNotFound {
			err = dropQueue(ctx, code)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("form %s failed: %w", code, err))
		}
	}
	return errors.Join(errs...)
}

// queuedModeCodes 扫描有人排队的玩法代码; 队列清空后 Redis 会删除它的键, 扫描到的都是非空队列
func queuedModeCodes(ctx context.Context) ([]string, error) {
	var codes []string
	iter := db.Redis.Scan(ctx, 0, queueKeyPrefix+"*", sweepScanCount).Iterator()
	for iter.Next(ctx) {
		codes = append(codes, strings.TrimPrefix(iter.Val(), queueKeyPrefix))
	}
	return codes, iter.Err()
}
