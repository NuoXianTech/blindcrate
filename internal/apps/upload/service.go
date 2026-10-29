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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/storage"
	"gorm.io/gorm"
)

// SaveRequest 上传一张图片
type SaveRequest struct {
	UserID  uint64
	IsAdmin bool
	Purpose string
	Data    []byte // 原图, 调用方读取时已按 MaxSize 限制大小
}

// View 上传结果
type View struct {
	ID          uint64 `json:"id"`
	URL         string `json:"url"` // 保存到业务表的地址
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Width       int32  `json:"width"`
	Height      int32  `json:"height"`
}

// MaxSize 原图的大小上限(字节), 见 settings.UploadSettings
func MaxSize(ctx context.Context) (int64, error) {
	cfg, err := settings.Upload.Get(ctx)
	if err != nil {
		return 0, err
	}
	return int64(cfg.MaxSizeKB) << 10, nil
}

// CheckPurpose 检查用途和上传权限, 不符合时返回 *InvalidError 或 *ForbiddenError
func CheckPurpose(ctx context.Context, purpose string, isAdmin bool) error {
	spec, ok := purposes[purpose]
	if !ok {
		return &InvalidError{Reason: PurposeInvalid}
	}
	if isAdmin {
		return nil
	}
	if spec.adminOnly {
		return &ForbiddenError{Reason: AdminRequired}
	}
	if purpose == PurposeAvatar {
		cfg, err := settings.Upload.Get(ctx)
		if err != nil {
			return err
		}
		if !cfg.AvatarEnabled {
			return &ForbiddenError{Reason: AvatarDisabled}
		}
	}
	return nil
}

// Save 检查并保存一张图片, 返回访问地址
//
// 按内容识别格式, 按用途缩小尺寸并重新编码后按内容的 sha256 命名;
// 同一用途的相同图片只存一份, 再次上传时复用原来的记录并刷新上传时间, 避免刚上传就被清理。
// 图片不符合要求时返回 *InvalidError, 没有权限时返回 *ForbiddenError。
func Save(ctx context.Context, req *SaveRequest) (*View, error) {
	if err := CheckPurpose(ctx, req.Purpose, req.IsAdmin); err != nil {
		return nil, err
	}
	cfg, err := settings.Upload.Get(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.Data) == 0 {
		return nil, &InvalidError{Reason: FileRequired}
	}
	if int64(len(req.Data)) > int64(cfg.MaxSizeKB)<<10 {
		return nil, &InvalidError{Reason: fmt.Sprintf(FileTooLarge, cfg.MaxSizeKB)}
	}
	if err := checkFormat(cfg, http.DetectContentType(req.Data)); err != nil {
		return nil, err
	}
	img, err := process(req.Data, purposes[req.Purpose])
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(img.data)
	hash := hex.EncodeToString(sum[:])
	key := fmt.Sprintf("%s/%s/%s%s", req.Purpose, hash[:2], hash, img.ext)
	store, err := storage.Default()
	if err != nil {
		return nil, err
	}
	var upload Upload
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 同一个文件的上传和清理在咨询锁上排队, 不会删除刚上传的文件
		if err := lockKey(tx, key); err != nil {
			return err
		}
		if err := tx.Where("key = ?", key).Limit(1).Find(&upload).Error; err != nil {
			return err
		}
		if upload.ID != 0 {
			upload.CreatedAt = time.Now()
			if err := tx.Model(&upload).Update("created_at", upload.CreatedAt).Error; err != nil {
				return err
			}
		} else {
			upload = Upload{
				UserID:      req.UserID,
				Purpose:     req.Purpose,
				Key:         key,
				ContentType: img.contentType,
				Size:        int64(len(img.data)),
				Width:       int32(img.width),
				Height:      int32(img.height),
				SHA256:      hash,
			}
			if err := tx.Create(&upload).Error; err != nil {
				return err
			}
		}
		// 已有记录时也重新写入, 文件丢失时可以通过重新上传恢复
		return store.Put(ctx, key, img.reader(), img.contentType)
	})
	if err != nil {
		return nil, err
	}
	return &View{
		ID:          upload.ID,
		URL:         store.URL(key),
		ContentType: upload.ContentType,
		Size:        upload.Size,
		Width:       upload.Width,
		Height:      upload.Height,
	}, nil
}

// checkFormat 原图格式是否为设置中允许的格式
func checkFormat(cfg *settings.UploadSettings, contentType string) error {
	allowed := []struct {
		contentType string
		name        string
		enabled     bool
	}{
		{contentTypePNG, "PNG", cfg.AllowPNG},
		{contentTypeJPEG, "JPEG", cfg.AllowJPEG},
		{contentTypeWebP, "WebP", cfg.AllowWebP},
	}
	var names []string
	for _, f := range allowed {
		if !f.enabled {
			continue
		}
		if f.contentType == contentType {
			return nil
		}
		names = append(names, f.name)
	}
	if len(names) == 0 {
		return &InvalidError{Reason: FormatDisabled}
	}
	return &InvalidError{Reason: fmt.Sprintf(FormatNotAllowed, strings.Join(names, "、"))}
}

// lockKey 在事务中按存储 key 加咨询锁, 事务结束时释放
func lockKey(tx *gorm.DB, key string) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "blindcrate:upload:"+key).Error
}
