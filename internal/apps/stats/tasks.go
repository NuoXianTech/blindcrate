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
	"fmt"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/task"
	"github.com/NuoXianTech/blindcrate/internal/task/schedule"
	"github.com/hibiken/asynq"
)

func init() {
	// 清理过期的按天和按周统计及登记的表, 可以重复执行, 失败时重试几次; 未配置时不清理
	task.Register(task.Job{Type: task.TypeStatsCleanup, Handler: HandleCleanup, Cron: config.Config.Schedule.StatsCleanup, Periodic: true, Queue: task.QueueLow, MaxRetry: 3})
}

// cleanupTarget 随统计一起清理的表, column 为其中的统计周期列
type cleanupTarget struct {
	table, column string
}

// cleanupTargets 需要清理的表, 统计本身排在第一个
var cleanupTargets = []cleanupTarget{{"stats", "period"}}

// RegisterCleanup 登记一张按统计周期记录的表, 过了保留天数的按天、按周周期随统计一起删除, 累计周期不受影响;
// 需要在启动前调用。用于只在当前周期有意义的记录, 如任务的领取记录
func RegisterCleanup(table, column string) {
	cleanupTargets = append(cleanupTargets, cleanupTarget{table: table, column: column})
}

// HandleCleanup 清理保留天数之前的按天统计和按周统计, 以及登记的表中同样过期的记录; 累计统计不受影响, 保留天数见 settings.StatsSettings
//
// 按周统计保留到包含最早保留日的那一周。周期中的日期为 YYYY-MM-DD、周为 YYYY-Www, 按字符串比较即为先后;
// 分批删除, 避免一次删除大量行长时间占用锁。可以重复执行, 中途失败时已删除的行不会恢复, 重试时继续删除。
func HandleCleanup(ctx context.Context, _ *asynq.Task) error {
	cfg, err := settings.Stats.Get(ctx)
	if err != nil {
		return err
	}
	// 保留含今天在内的 DayRetention 天
	earliest := time.Now().In(schedule.Location).AddDate(0, 0, 1-cfg.DayRetention)
	dayCutoff, weekCutoff := PeriodDay(earliest), PeriodWeek(earliest)
	for _, target := range cleanupTargets {
		deleted, err := cleanupBefore(ctx, target, dayCutoff, weekCutoff)
		if err != nil {
			return err
		}
		if deleted > 0 {
			logger.InfoF(ctx, "[Stats] cleaned up %d rows of %s before %s", deleted, target.table, dayCutoff)
		}
	}
	return nil
}

// cleanupBefore 分批删除按天周期早于 dayCutoff、按周周期早于 weekCutoff 的记录, 返回删除的行数; 两种周期在同一次扫描中删除
func cleanupBefore(ctx context.Context, target cleanupTarget, dayCutoff, weekCutoff string) (int64, error) {
	// 表名和列名来自代码中的登记, 不是用户输入
	query := fmt.Sprintf(
		"DELETE FROM %[1]s WHERE ctid IN (SELECT ctid FROM %[1]s WHERE (%[2]s LIKE ? AND %[2]s < ?) OR (%[2]s LIKE ? AND %[2]s < ?) LIMIT ?)",
		target.table, target.column,
	)
	var deleted int64
	for {
		result := db.DB(ctx).Exec(query, periodDayPrefix+"%", dayCutoff, periodWeekPrefix+"%", weekCutoff, cleanupBatch)
		if result.Error != nil {
			return deleted, result.Error
		}
		deleted += result.RowsAffected
		if result.RowsAffected < cleanupBatch {
			return deleted, nil
		}
	}
}
