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

package stats

import (
	"context"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/task/schedule"
	"github.com/hibiken/asynq"
)

// HandleCleanup 清理保留天数之前的按天统计, 累计统计不受影响; 保留天数见 settings.StatsSettings
//
// 周期中的日期为 YYYY-MM-DD, 按字符串比较即为日期先后; 分批删除, 避免一次删除大量行长时间占用锁。
// 可以重复执行, 中途失败时已删除的行不会恢复, 重试时继续删除。
func HandleCleanup(ctx context.Context, _ *asynq.Task) error {
	cfg, err := settings.Stats.Get(ctx)
	if err != nil {
		return err
	}
	// 保留含今天在内的 DayRetention 天
	cutoff := PeriodDay(time.Now().In(schedule.Location).AddDate(0, 0, 1-cfg.DayRetention))
	var deleted int64
	for {
		result := db.DB(ctx).Exec(
			"DELETE FROM stats WHERE ctid IN (SELECT ctid FROM stats WHERE period LIKE ? AND period < ? LIMIT ?)",
			periodDayPrefix+"%", cutoff, cleanupBatch,
		)
		if result.Error != nil {
			return result.Error
		}
		deleted += result.RowsAffected
		if result.RowsAffected < cleanupBatch {
			break
		}
	}
	if deleted > 0 {
		logger.InfoF(ctx, "[Stats] cleaned up %d day stats before %s", deleted, cutoff)
	}
	return nil
}
