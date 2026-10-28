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

	// inspector 查询和删除队列中的任务, 用于取消定时器
	inspector *asynq.Inspector

	scheduler     *asynq.Scheduler
	schedulerOnce sync.Once
)

func init() {
	AsynqClient = asynq.NewClient(task.RedisOpt)
	inspector = asynq.NewInspector(task.RedisOpt)
}

// StartScheduler 启动定时任务调度, 按 cron 把任务投递到队列, 阻塞到收到退出信号为止
//
// 只能运行一个实例, 多个实例会重复投递同一个定时任务。
func StartScheduler() error {
	var err error
	schedulerOnce.Do(func() {
		location, locErr := time.LoadLocation(config.Config.Schedule.TimeZone)
		if locErr != nil {
			err = fmt.Errorf("load location %q failed: %w", config.Config.Schedule.TimeZone, locErr)
			return
		}
		scheduler = asynq.NewScheduler(
			task.RedisOpt,
			&asynq.SchedulerOpts{
				Location: location,
				// asynq 投递失败时不打日志, 这里补上, 避免定时任务漏跑没人发现
				EnqueueErrorHandler: func(t *asynq.Task, _ []asynq.Option, enqueueErr error) {
					logger.ErrorF(context.Background(), "[Scheduler] enqueue %s failed: %v", t.Type(), enqueueErr)
				},
			},
		)

		// 启动调度器
		err = scheduler.Run()
	})
	return err
}
