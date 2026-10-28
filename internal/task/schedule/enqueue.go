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
	"errors"
	"slices"

	"github.com/hibiken/asynq"
)

// EnqueueOnce 以 taskID 投递任务, 同一个 taskID 的任务还在队列中时不重复投递, 直接返回 nil
//
// 用于对战阶段超时等服务端定时器: 用 asynq.ProcessAt 指定到期时间, taskID 带上对象和阶段, 如 match:42:round:3,
// 同一个定时器被重复设置时只生效一次。任务执行成功后同一个 taskID 可以再次投递,
// 但重试中和重试耗尽被归档的任务仍占用 taskID, 此时投递也直接返回 nil, 所以 taskID 需要区分每一次定时, 不要复用。
// 到期的任务可能晚于预期执行, 也可能在状态已经推进后才执行, 处理函数需要先检查状态是否仍是设置定时器时的状态。
func EnqueueOnce(ctx context.Context, t *asynq.Task, taskID string, opts ...asynq.Option) error {
	_, err := AsynqClient.EnqueueContext(ctx, t, append(slices.Clip(opts), asynq.TaskID(taskID))...)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		return nil
	}
	return err
}

// Cancel 删除还没开始执行的任务, 任务不存在或已执行完时返回 nil
//
// queue 为投递时的队列, 未指定队列时为 task.QueueDefault; 正在执行的任务无法删除, 返回错误。
func Cancel(queue, taskID string) error {
	err := inspector.DeleteTask(queue, taskID)
	if errors.Is(err, asynq.ErrTaskNotFound) || errors.Is(err, asynq.ErrQueueNotFound) {
		return nil
	}
	return err
}
