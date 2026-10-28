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

// Package task 异步任务, 基于 asynq, 任务存放在 Redis 队列中
//
// 新增一个任务:
//   - 在这里定义任务类型, 格式为 <app>:<动作>, 如 dashboard:update_stats
//   - 处理函数放在对应 app 的 tasks.go, 并在 worker.StartWorker 中注册
//   - 定时任务在 schedule.StartScheduler 中注册, cron 表达式放在配置的 schedule 下
//   - 对战阶段超时等一次性的定时器用 schedule.EnqueueOnce 投递, 提前结束时用 schedule.Cancel 取消
//
// 任务至少会执行一次: 失败重试或 worker 中途退出后, 同一个任务可能再次执行, 处理函数需要保证幂等。
package task

// 队列名称, 投递时未指定队列的任务进 default
//
// worker 严格按优先级处理: critical 为空时才处理 default, default 为空时才处理 low。
const (
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"
)
