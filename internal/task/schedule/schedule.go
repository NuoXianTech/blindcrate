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

package schedule

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/task"
	"github.com/hibiken/asynq"
)

var (
	// AsynqClient 投递任务的客户端, 业务代码用它把任务放进队列
	AsynqClient *asynq.Client

	// Location 定时任务和按天统计使用的时区, 来自 schedule.time_zone; 按天划分的数据都用它计算日期, 与定时任务一致
	Location *time.Location

	// inspector 查询和删除队列中的任务, 用于取消定时器
	inspector *asynq.Inspector

	scheduler     *asynq.Scheduler
	schedulerOnce sync.Once
)

func init() {
	location, err := time.LoadLocation(config.Config.Schedule.TimeZone)
	if err != nil {
		log.Fatalf("[Scheduler] load location %q failed: %v\n", config.Config.Schedule.TimeZone, err)
	}
	Location = location
	AsynqClient = asynq.NewClient(task.RedisOpt)
	inspector = asynq.NewInspector(task.RedisOpt)
}

// StartScheduler 启动定时任务调度, 按 cron 把任务投递到队列, 阻塞到收到退出信号为止
//
// 只能运行一个实例, 多个实例会重复投递同一个定时任务。
func StartScheduler() error {
	var err error
	schedulerOnce.Do(func() {
		scheduler = asynq.NewScheduler(
			task.RedisOpt,
			&asynq.SchedulerOpts{
				Location: Location,
				// asynq 投递失败时不打日志, 这里补上, 避免定时任务漏跑没人发现
				EnqueueErrorHandler: func(t *asynq.Task, _ []asynq.Option, enqueueErr error) {
					logger.ErrorF(context.Background(), "[Scheduler] enqueue %s failed: %v", t.Type(), enqueueErr)
				},
			},
		)

		// 巡检到期未推进的对局和排队, 失败时等下一次巡检, 不重试
		if _, err = scheduler.Register(config.Config.Schedule.MatchSweep, asynq.NewTask(task.TypeMatchSweep, nil), asynq.MaxRetry(0)); err != nil {
			err = fmt.Errorf("register %s failed: %w", task.TypeMatchSweep, err)
			return
		}

		// 清理过期的按天统计, 可以重复执行, 失败时重试几次; 未配置时不清理
		if cron := config.Config.Schedule.StatsCleanup; cron == "" {
			logger.WarnF(context.Background(), "[Scheduler] schedule.stats_cleanup is empty, day stats will not be cleaned up")
		} else if _, err = scheduler.Register(cron, asynq.NewTask(task.TypeStatsCleanup, nil), asynq.Queue(task.QueueLow), asynq.MaxRetry(3)); err != nil {
			err = fmt.Errorf("register %s failed: %w", task.TypeStatsCleanup, err)
			return
		}

		// 发放赛季奖励, 可以重复执行, 失败时重试几次; 未配置时不发放
		if cron := config.Config.Schedule.SeasonSettle; cron == "" {
			logger.WarnF(context.Background(), "[Scheduler] schedule.season_settle is empty, season rewards will not be granted")
		} else if _, err = scheduler.Register(cron, asynq.NewTask(task.TypeSeasonSettle, nil), asynq.Queue(task.QueueLow), asynq.MaxRetry(3)); err != nil {
			err = fmt.Errorf("register %s failed: %w", task.TypeSeasonSettle, err)
			return
		}

		// 启动调度器
		err = scheduler.Run()
	})
	return err
}
