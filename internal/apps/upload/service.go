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
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/storage"
	"gorm.io/gorm"
)

// tableNamePattern 登记引用时的表名和字段名, 拼进 SQL 中, 只允许小写字母、数字和下划线
var tableNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// RegisterPurpose 登记一种图片用途, 需要在启动前调用; 格式不正确或重复时 panic
//
// adminOnly 为 true 时只有管理员可以上传; square 为 true 时从中间裁成正方形; maxSide 为缩小后的最大边长(像素)。
func RegisterPurpose(code string, adminOnly, square bool, maxSide int) {
	if !tableNamePattern.MatchString(code) || maxSide <= 0 {
		panic(fmt.Sprintf("upload: invalid purpose %q", code))
	}
	if _, ok := purposes[code]; ok {
		panic(fmt.Sprintf("upload: purpose %s already registered", code))
	}
	purposes[code] = purposeSpec{adminOnly: adminOnly, square: square, maxSide: maxSide}
}

// RegisterReference 登记一个保存图片地址的字段, 地址以存储 key 结尾的上传不会被清理; 需要在启动前调用
func RegisterReference(table, column string) {
	references = append(references, urlReference(checkName(table), checkName(column)))
}

// RegisterJSONReference 登记一个 JSON 字段, 内容中出现存储 key 的上传不会被清理, 用于图片地址保存在配置中的情况; 需要在启动前调用
func RegisterJSONReference(table, column string) {
	references = append(references, jsonReference(checkName(table), checkName(column)))
}

// checkName 校验表名或字段名, 不符合时 panic
func checkName(name string) string {
	if !tableNamePattern.MatchString(name) {
		panic(fmt.Sprintf("upload: invalid table or column name %q", name))
	}
	return name
}

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

// Check 按保存的规则检查并处理图片, 不保存, 返回保存后的访问地址, 如预览导入的配置包时; 返回的错误同 Save
//
// 处理的结果是确定的, 同一张原图保存后的地址与这里返回的相同。
func Check(ctx context.Context, req *SaveRequest) (string, error) {
	img, err := prepare(ctx, req)
	if err != nil {
		return "", err
	}
	store, err := storage.Default()
	if err != nil {
		return "", err
	}
	key, _ := img.key(req.Purpose)
	return store.URL(key), nil
}

// Stored 同一用途的这张图片已经保存过时返回访问地址, data 为保存后的文件内容, 如导出后再导入的配置包中的图片
//
// 文件按内容命名, 内容与 key 一致且记录和文件都在时直接复用; 重新编码 JPEG 会得到不同的文件, 因此不能再按原图保存。
func Stored(ctx context.Context, purpose, key string, data []byte) (string, bool, error) {
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if !strings.HasPrefix(key, fmt.Sprintf("%s/%s/%s.", purpose, hash[:2], hash)) {
		return "", false, nil
	}
	var count int64
	if err := db.DB(ctx).Model(&Upload{}).Where("key = ?", key).Count(&count).Error; err != nil {
		return "", false, err
	}
	if count == 0 {
		return "", false, nil
	}
	store, err := storage.Default()
	if err != nil {
		return "", false, err
	}
	rc, err := store.Open(ctx, key)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	_ = rc.Close()
	return store.URL(key), true, nil
}

// Save 检查并保存一张图片, 返回访问地址
//
// 按内容识别格式, 按用途缩小尺寸并重新编码后按内容的 sha256 命名;
// 同一用途的相同图片只存一份, 再次上传时复用原来的记录并刷新上传时间, 避免刚上传就被清理。
// 图片不符合要求时返回 *InvalidError, 没有权限时返回 *ForbiddenError。
func Save(ctx context.Context, req *SaveRequest) (*View, error) {
	img, err := prepare(ctx, req)
	if err != nil {
		return nil, err
	}

	key, hash := img.key(req.Purpose)
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

// prepare 检查用途、大小和格式, 按用途缩小尺寸并重新编码
func prepare(ctx context.Context, req *SaveRequest) (*processed, error) {
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
	return process(req.Data, purposes[req.Purpose])
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
