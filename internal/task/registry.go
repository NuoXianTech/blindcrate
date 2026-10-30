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

package task

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/hibiken/asynq"
)

// Handler 任务的处理函数, 返回错误时按重试次数重新执行
type Handler func(ctx context.Context, t *asynq.Task) error

// Job 一种任务, 由所属模块在 init 中用 Register 登记; worker 处理全部登记的任务, 调度器按 cron 定时投递
type Job struct {
	Type    string  // 任务类型, 格式为 <app>:<动作>, 如 mailbox:cleanup
	Handler Handler // 处理函数, 需要保证幂等
	// Cron 定时投递的默认 cron 表达式, 为空时只由业务代码投递
	//
	// 配置 schedule.crons 中有这个类型时以配置为准, 配置为空字符串时不定时投递。
	Cron     string
	Periodic bool   // 是定时任务, 没有 cron 时调度器记录警告
	Required bool   // 是必需的定时任务, 没有 cron 时调度器启动失败
	Queue    string // 定时投递进入的队列, 为空时进 default
	MaxRetry int    // 定时投递的任务失败后的重试次数
}

// typePattern 任务类型的格式
var typePattern = regexp.MustCompile(`^[a-z0-9_]+:[a-z0-9_]+$`)

// jobs 已登记的任务, 按登记顺序, 只在启动前写入
var jobs []Job

// Register 登记一种任务, 需要在启动前调用; 格式不正确或重复时 panic
func Register(job Job) {
	if !typePattern.MatchString(job.Type) || job.Handler == nil {
		panic(fmt.Sprintf("task: invalid job %q", job.Type))
	}
	if slices.ContainsFunc(jobs, func(j Job) bool { return j.Type == job.Type }) {
		panic(fmt.Sprintf("task: job %s already registered", job.Type))
	}
	jobs = append(jobs, job)
}

// Jobs 全部登记的任务, 按登记顺序
func Jobs() []Job {
	return slices.Clone(jobs)
}

// CronSpec 定时投递使用的 cron 表达式, 配置 schedule.crons 优先; 为空时不定时投递
func (j Job) CronSpec() string {
	if spec, ok := config.Config.Schedule.Crons[j.Type]; ok {
		return spec
	}
	return j.Cron
}
