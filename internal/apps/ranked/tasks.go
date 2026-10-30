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

	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/task"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

func init() {
	// 发放赛季奖励, 可以重复执行, 失败时重试几次; 未配置时不发放
	task.Register(task.Job{Type: task.TypeSeasonSettle, Handler: HandleSeasonSettle, Cron: config.Config.Schedule.SeasonSettle, Periodic: true, Queue: task.QueueLow, MaxRetry: 3})
}

// HandleSeasonSettle 给已结束的赛季发放赛季奖励, 发完后写入 settled_at
//
// 赛季结束 settleDelay 之后才发放; 本赛季打过排位的人按最高段位分领取满足条件的最高一档, 停用和封禁中的账号不发。
// 每人一个事务, 中途失败时重试会跳过已发放的人, 可以重复执行。
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
	granted := 0
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
			n, err := grantSeasonRewards(ctx, s, ratings)
			if err != nil {
				return err
			}
			granted += n
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
		logger.InfoF(ctx, "[Ranked] settled season %s, granted rewards to %d users", s.Code, granted)
	}
	return nil
}

// grantSeasonRewards 给一批玩家发放赛季奖励, 返回本次发放的人数
//
// 已有赛季奖励流水的人视为已发放, 不再发放: 重试时已下架的物品会折成货币, 内容变了, 幂等键拦不住。
func grantSeasonRewards(ctx context.Context, s *season.Season, ratings []Rating) (int, error) {
	userIDs := make([]uint64, len(ratings))
	for i, r := range ratings {
		userIDs[i] = r.UserID
	}
	done, err := seasonRewarded(db.DB(ctx), s.Code, userIDs)
	if err != nil {
		return 0, err
	}
	granted := 0
	for _, r := range ratings {
		target := s.RewardOf(r.Peak)
		if target == nil || done[r.UserID] {
			continue
		}
		var (
			result   *reward.Result
			replayed bool
		)
		err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
			bundle, err := reward.Deliverable(tx, target.Bundle, nil)
			if err != nil {
				return err
			}
			result, replayed, err = reward.Apply(tx, &reward.ApplyRequest{
				UserID:         r.UserID,
				Bundle:         bundle,
				BizType:        wallet.BizSeasonReward,
				BizID:          s.Code,
				IdempotencyKey: fmt.Sprintf("season:%s:reward:%d", s.Code, r.UserID),
				Remark:         s.Name,
			})
			return err
		})
		if err != nil {
			return granted, fmt.Errorf("grant user %d: %w", r.UserID, err)
		}
		if !replayed {
			result.Push(ctx)
			granted++
		}
	}
	return granted, nil
}

// seasonRewarded 这批用户中已经领过赛季奖励的人, 余额和背包的流水都要查, 奖励可能只有其中一种
func seasonRewarded(tx *gorm.DB, code string, userIDs []uint64) (map[uint64]bool, error) {
	var walletUsers, inventoryUsers []uint64
	if err := tx.Model(&wallet.Transaction{}).
		Where("biz_type = ? AND biz_id = ? AND user_id IN ?", wallet.BizSeasonReward, code, userIDs).
		Distinct().Pluck("user_id", &walletUsers).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&inventory.Transaction{}).
		Where("biz_type = ? AND biz_id = ? AND user_id IN ?", inventory.BizSeasonReward, code, userIDs).
		Distinct().Pluck("user_id", &inventoryUsers).Error; err != nil {
		return nil, err
	}
	done := make(map[uint64]bool, len(walletUsers)+len(inventoryUsers))
	for _, id := range walletUsers {
		done[id] = true
	}
	for _, id := range inventoryUsers {
		done[id] = true
	}
	return done, nil
}
