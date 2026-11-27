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

package ranked

import (
	"context"
	"fmt"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/mailbox"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/task"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

func init() {
	// 发放赛季奖励, 可以重复执行, 失败时重试几次; 未配置时不发放
	mailbox.RegisterSource(SourceSeason, "赛季奖励")
	task.Register(task.Job{Type: task.TypeSeasonSettle, Handler: HandleSeasonSettle, Cron: config.Config.Schedule.SeasonSettle, Periodic: true, Queue: task.QueueLow, MaxRetry: 3})
}

// HandleSeasonSettle 给已结束的赛季发放赛季奖励, 发完后写入 settled_at
//
// 赛季结束 settleDelay 之后才发放; 本赛季打过排位的人按最高段位分得到满足条件的最高一档, 停用和封禁中的账号不发。
// 奖励以邮件发放, 玩家在信箱中领取附件; 同一个赛季对同一用户只写一封, 中途失败时重试不会重复, 可以重复执行。
func HandleSeasonSettle(ctx context.Context, _ *asynq.Task) error {
	var seasons []season.Season
	if err := db.DB(ctx).
		Where("settled_at IS NULL AND ends_at <= ?", time.Now().Add(-settleDelay)).
		Order("ends_at").Find(&seasons).Error; err != nil {
		return err
	}
	for i := range seasons {
		if err := settleSeason(ctx, &seasons[i]); err != nil {
			return fmt.Errorf("settle season %s: %w", seasons[i].Code, err)
		}
	}
	return nil
}

// settleSeason 按用户 ID 分批给一个赛季的玩家发放奖励, 发完后标记为已结算
func settleSeason(ctx context.Context, s *season.Season) error {
	var granted int64
	if len(s.Rewards) > 0 {
		var lastUserID uint64
		for {
			var ratings []Rating
			if err := db.DB(ctx).
				Where("season_id = ? AND user_id > ? AND matches > 0 AND peak >= ?", s.ID, lastUserID, s.Rewards[0].MinRating).
				Where("user_id IN (SELECT id FROM users WHERE NOT is_disabled AND (banned_until IS NULL OR banned_until <= ?))", time.Now()).
				Order("user_id").Limit(settleBatch).Find(&ratings).Error; err != nil {
				return err
			}
			if len(ratings) == 0 {
				break
			}
			sent, err := mailSeasonRewards(ctx, s, ratings)
			if err != nil {
				return err
			}
			sent.Push(ctx)
			granted += sent.Count
			lastUserID = ratings[len(ratings)-1].UserID
		}
	}

	// 并发执行时只有一次写入成功
	result := db.DB(ctx).Model(&season.Season{}).
		Where("id = ? AND settled_at IS NULL", s.ID).
		Update("settled_at", time.Now())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		logger.InfoF(ctx, "[Ranked] settled season %s, mailed rewards to %d users", s.Code, granted)
	}
	return nil
}

// mailSeasonRewards 给一批玩家发送赛季奖励邮件, 一批一个事务
//
// 来源单号为赛季代码, 已经收到过的人会被跳过。附件是赛季中配置的内容, 领取时才发放:
// 已下架的物品在领取时按回收价折成货币, 货币已停用时玩家暂时领不了, 不影响结算。
func mailSeasonRewards(ctx context.Context, s *season.Season, ratings []Rating) (*mailbox.Sent, error) {
	mails := make([]mailbox.Mail, 0, len(ratings))
	for _, r := range ratings {
		target := s.RewardOf(r.Peak)
		if target == nil || target.Bundle.IsEmpty() {
			continue
		}
		mails = append(mails, mailbox.Mail{
			UserID:      r.UserID,
			Source:      SourceSeason,
			SourceID:    s.Code,
			Title:       seasonMailTitle,
			Content:     fmt.Sprintf(seasonMailContent, s.Name, r.Peak, target.MinRating),
			Attachments: target.Bundle,
		})
	}
	var sent *mailbox.Sent
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		sent, err = mailbox.Send(tx, mails)
		return err
	})
	if err != nil {
		return nil, err
	}
	return sent, nil
}
