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
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/storage"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

// references 保存图片地址的位置, 地址以存储 key 结尾的上传视为被引用; 新增保存图片地址的字段时在这里补上
//
// 已软删除的行也算引用, 恢复后图片仍可用; 设置组按整个 JSON 查找, 如站点 Logo。
var references = []string{
	"SELECT 1 FROM users WHERE RIGHT(avatar_url, LENGTH(uploads.key)) = uploads.key",
	"SELECT 1 FROM items WHERE RIGHT(image_url, LENGTH(uploads.key)) = uploads.key",
	"SELECT 1 FROM venues WHERE RIGHT(image_url, LENGTH(uploads.key)) = uploads.key",
	"SELECT 1 FROM shop_items WHERE RIGHT(image_url, LENGTH(uploads.key)) = uploads.key",
	"SELECT 1 FROM settings WHERE STRPOS(value::text, uploads.key) > 0",
}

// unreferenced 没有被引用的条件
var unreferenced = func() string {
	conds := make([]string, 0, len(references))
	for _, ref := range references {
		conds = append(conds, "NOT EXISTS ("+ref+")")
	}
	return strings.Join(conds, " AND ")
}()

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
		if err := db.DB(ctx).Where("id > ? AND created_at < ? AND "+unreferenced, lastID, cutoff).
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
		result := tx.Where("id = ? AND created_at < ? AND "+unreferenced, upload.ID, cutoff).Delete(&Upload{})
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
