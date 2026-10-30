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

		// 按登记的任务定时投递, 见 task.Register
		for _, job := range task.Jobs() {
			if err = registerJob(job); err != nil {
				return
			}
		}

		// 启动调度器
		err = scheduler.Run()
	})
	return err
}

// registerJob 按 cron 定时投递一种任务; 没有 cron 时跳过, 必需的任务返回错误
func registerJob(job task.Job) error {
	spec := job.CronSpec()
	if spec == "" {
		if job.Required {
			return fmt.Errorf("cron of %s is required", job.Type)
		}
		if job.Periodic {
			logger.WarnF(context.Background(), "[Scheduler] cron of %s is empty, it will not be scheduled", job.Type)
		}
		return nil
	}
	opts := []asynq.Option{asynq.MaxRetry(job.MaxRetry)}
	if job.Queue != "" {
		opts = append(opts, asynq.Queue(job.Queue))
	}
	if _, err := scheduler.Register(spec, asynq.NewTask(job.Type, nil), opts...); err != nil {
		return fmt.Errorf("register %s failed: %w", job.Type, err)
	}
	return nil
}
