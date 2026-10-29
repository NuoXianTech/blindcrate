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
	"time"

	"gorm.io/gorm"
)

// Upload 上传的图片, 一个文件一行
//
// 同一用途的相同图片(重新编码后的内容相同)只存一份, 再次上传时复用并刷新上传时间, 见 Save;
// 清理时软删除并删除文件, 之后再上传相同图片会重新写入文件并新建一行。
type Upload struct {
	ID          uint64         `gorm:"primaryKey" json:"id"`
	UserID      uint64         `gorm:"not null;index" json:"user_id"`                                                      // 第一次上传的用户
	Purpose     string         `gorm:"size:16;not null" json:"purpose"`                                                    // 用途, 见 Purpose 开头的常量
	Key         string         `gorm:"size:255;not null;uniqueIndex:uniq_uploads_key,where:deleted_at IS NULL" json:"key"` // 存储中的 key, 如 item/ab/abcd….png
	ContentType string         `gorm:"size:32;not null" json:"content_type"`                                               // 保存后的格式
	Size        int64          `gorm:"not null" json:"size"`                                                               // 保存后的字节数
	Width       int32          `gorm:"not null" json:"width"`
	Height      int32          `gorm:"not null" json:"height"`
	SHA256      string         `gorm:"column:sha256;size:64;not null" json:"sha256"`                  // 保存后内容的 sha256
	CreatedAt   time.Time      `gorm:"not null;default:now();autoCreateTime;index" json:"created_at"` // 上传时间, 重复上传时刷新, 清理按它计算保留天数
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 自定义表名
func (Upload) TableName() string { return "uploads" }
