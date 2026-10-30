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

package mailbox

import (
	"context"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/task"
	"github.com/hibiken/asynq"
)

func init() {
	// 每天清理一次, 可以重复执行, 失败时重试几次
	task.Register(task.Job{Type: TypeCleanup, Handler: HandleCleanup, Cron: cleanupCron, Periodic: true, Queue: task.QueueLow, MaxRetry: 3})
}

// HandleCleanup 删除过期超过保留天数的邮件, 并把超出保留封数的旧邮件移出信箱; 设置见 MailboxSettings
//
// 超出封数的邮件为软删除, 来源单号仍然占用, 全服邮件不会再次展开; 附件未领取且未过期的邮件不移除。
// 可以重复执行, 分批删除, 避免一次删除大量行长时间占用锁。
func HandleCleanup(ctx context.Context, _ *asynq.Task) error {
	cfg, err := Settings.Get(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	cutoff := now.Add(-time.Duration(cfg.RetentionDays) * 24 * time.Hour)
	var deleted int64
	for {
		result := db.DB(ctx).Exec("DELETE FROM mails WHERE id IN (SELECT id FROM mails WHERE expires_at < ? LIMIT ?)", cutoff, cleanupBatch)
		if result.Error != nil {
			return result.Error
		}
		deleted += result.RowsAffected
		if result.RowsAffected < cleanupBatch {
			break
		}
	}

	// 每个超出封数的用户按 ID 从新到旧排, 排在保留封数之后的移出信箱
	result := db.DB(ctx).Exec(`
UPDATE mails SET deleted_at = ? WHERE id IN (
    SELECT id FROM (
        SELECT id, claimed_at, attachments, expires_at,
               ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY id DESC) AS rn
        FROM mails
        WHERE deleted_at IS NULL AND expires_at > ? AND user_id IN (
            SELECT user_id FROM mails WHERE deleted_at IS NULL AND expires_at > ? GROUP BY user_id HAVING COUNT(*) > ?
        )
    ) ranked
    WHERE rn > ? AND NOT (attachments <> '{}'::jsonb AND claimed_at IS NULL)
)`, now, now, now, cfg.MaxKept, cfg.MaxKept)
	if result.Error != nil {
		return result.Error
	}
	if deleted > 0 || result.RowsAffected > 0 {
		logger.InfoF(ctx, "[Mailbox] cleaned up %d expired mails before %s, removed %d mails over the limit",
			deleted, cutoff.Format(time.RFC3339), result.RowsAffected)
	}
	return nil
}
