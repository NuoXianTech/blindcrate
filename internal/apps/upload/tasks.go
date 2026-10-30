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

package upload

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/storage"
	"github.com/NuoXianTech/blindcrate/internal/task"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

func init() {
	// 删除没有被引用的上传图片, 可以重复执行, 失败时重试几次; 未配置时不清理
	task.Register(task.Job{Type: task.TypeUploadCleanup, Handler: HandleCleanup, Cron: config.Config.Schedule.UploadCleanup, Periodic: true, Queue: task.QueueLow, MaxRetry: 3})
}

// references 保存图片地址的位置, 见 RegisterReference 和 RegisterJSONReference; 只在启动前写入
//
// 已软删除的行也算引用, 恢复后图片仍可用; 设置组按整个 JSON 查找, 如站点 Logo。
var references = []string{
	urlReference("users", "avatar_url"),
	urlReference("items", "image_url"),
	urlReference("venues", "image_url"),
	urlReference("shop_items", "image_url"),
	jsonReference("settings", "value"),
}

// urlReference 字段保存一个图片地址, 地址以存储 key 结尾的上传视为被引用
func urlReference(table, column string) string {
	return fmt.Sprintf("SELECT 1 FROM %s WHERE RIGHT(%s, LENGTH(uploads.key)) = uploads.key", table, column)
}

// jsonReference 字段为 JSON, 内容中出现存储 key 的上传视为被引用
func jsonReference(table, column string) string {
	return fmt.Sprintf("SELECT 1 FROM %s WHERE STRPOS(%s::text, uploads.key) > 0", table, column)
}

// unreferenced 没有被引用的条件, 在启动后第一次清理时生成
var unreferenced = sync.OnceValue(func() string {
	conds := make([]string, 0, len(references))
	for _, ref := range references {
		conds = append(conds, "NOT EXISTS ("+ref+")")
	}
	return strings.Join(conds, " AND ")
})

// HandleCleanup 删除上传后超过保留天数仍没有被引用的图片, 保留天数见 settings.UploadSettings
//
// 先软删除记录再删除文件, 与上传同一个文件时在咨询锁上排队; 删除文件失败时回滚, 下次重试。
// 可以重复执行, 中途失败时已删除的图片不会恢复。
func HandleCleanup(ctx context.Context, _ *asynq.Task) error {
	cfg, err := settings.Upload.Get(ctx)
	if err != nil {
		return err
	}
	store, err := storage.Default()
	if err != nil {
		return err
	}
	cutoff := time.Now().AddDate(0, 0, -cfg.RetentionDays)
	var deleted int
	var lastID uint64
	for {
		var candidates []Upload
		if err := db.DB(ctx).Where("id > ? AND created_at < ? AND "+unreferenced(), lastID, cutoff).
			Order("id").Limit(cleanupBatch).Find(&candidates).Error; err != nil {
			return err
		}
		for _, candidate := range candidates {
			ok, err := deleteUpload(ctx, store, &candidate, cutoff)
			if err != nil {
				return fmt.Errorf("delete upload %d: %w", candidate.ID, err)
			}
			if ok {
				deleted++
			}
		}
		if len(candidates) < cleanupBatch {
			break
		}
		lastID = candidates[len(candidates)-1].ID
	}
	if deleted > 0 {
		logger.InfoF(ctx, "[Upload] cleaned up %d unused uploads before %s", deleted, cutoff.Format(time.DateOnly))
	}
	return nil
}

// deleteUpload 加锁后再次确认仍未被引用且没有重新上传, 软删除记录并删除文件, 返回是否删除
func deleteUpload(ctx context.Context, store storage.Storage, upload *Upload, cutoff time.Time) (bool, error) {
	var deleted bool
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockKey(tx, upload.Key); err != nil {
			return err
		}
		result := tx.Where("id = ? AND created_at < ? AND "+unreferenced(), upload.ID, cutoff).Delete(&Upload{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		deleted = true
		return store.Delete(ctx, upload.Key)
	})
	return deleted, err
}
