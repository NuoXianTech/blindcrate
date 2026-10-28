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

package worker

import (
	"context"
	"time"

	_ "github.com/NuoXianTech/blindcrate/internal/apps/auction" // 注册暗拍玩法, 阶段超时时调用
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/otel_trace"
	"github.com/NuoXianTech/blindcrate/internal/task"
	"github.com/hibiken/asynq"
)

// StartWorker 启动任务处理服务, 从队列取出任务交给注册的处理函数, 阻塞到收到退出信号为止
//
// 可以运行多个实例; 收到退出信号后最多等 3 分钟让正在处理的任务完成, 没完成的任务会重新入队。
func StartWorker() error {
	defer otel_trace.Shutdown(context.Background())

	asynqServer := asynq.NewServer(
		task.RedisOpt,
		asynq.Config{
			Concurrency:     config.Config.Worker.Concurrency,
			ShutdownTimeout: 3 * time.Minute,
			// 定时任务默认每 5 秒检查一次是否到期, 缩短到 1 秒让对局阶段准时结束
			DelayedTaskCheckInterval: time.Second,
			Queues: map[string]int{
				task.QueueCritical: 10,
				task.QueueDefault:  5,
				task.QueueLow:      1,
			},
			StrictPriority: true,
		},
	)

	// 注册任务处理器
	mux := asynq.NewServeMux()
	mux.Use(taskLoggingMiddleware)
	mux.HandleFunc(task.TypeMatchPhaseTimeout, match.HandlePhaseTimeout)
	mux.HandleFunc(task.TypeMatchBotFill, match.HandleBotFill)
	mux.HandleFunc(task.TypeMatchSweep, match.HandleSweep)

	// 启动服务
	return asynqServer.Run(mux)
}
