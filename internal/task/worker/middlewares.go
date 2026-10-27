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

	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/otel_trace"
	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// taskLoggingMiddleware 为每个任务创建 Span, 记录处理结果和耗时
func taskLoggingMiddleware(h asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
		// 初始化 Trace
		ctx, span := otel_trace.Start(ctx, "TaskProcess_"+t.Type(), trace.WithSpanKind(trace.SpanKindConsumer))
		defer span.End()

		// 添加任务信息到 Span
		taskID := t.ResultWriter().TaskID()
		retryCount, _ := asynq.GetRetryCount(ctx)
		span.SetAttributes(
			attribute.String("task.type", t.Type()),
			attribute.String("task.id", taskID),
			attribute.Int("task.payload_size", len(t.Payload())),
			attribute.Int("task.retry_count", retryCount),
		)

		// 开始计时
		start := time.Now()

		// 处理任务
		err := h.ProcessTask(ctx, t)

		// 停止计时
		end := time.Now()
		latency := end.Sub(start)

		if err != nil {
			maxRetry, _ := asynq.GetMaxRetry(ctx)
			logger.ErrorF(
				ctx,
				"[TaskMiddleware] %s failed\nTaskID: %s\nRetry: %d/%d\nStartTime: %s\nLatency: %d\nError: %v",
				t.Type(),
				taskID,
				retryCount,
				maxRetry,
				start.Format(time.RFC3339),
				latency.Milliseconds(),
				err,
			)

			// 设置 Span 错误状态
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			return err
		}

		logger.InfoF(
			ctx,
			"[TaskMiddleware] %s done\nTaskID: %s\nStartTime: %s\nEndTime: %s\nLatency: %d",
			t.Type(),
			taskID,
			start.Format(time.RFC3339),
			end.Format(time.RFC3339),
			latency.Milliseconds(),
		)
		return nil
	})
}
