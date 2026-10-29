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

package pack

import "time"

// Import 一次导入配置包的记录, 只增不改
type Import struct {
	ID        uint64            `gorm:"primaryKey" json:"id"`
	UserID    uint64            `gorm:"not null;index:idx_imports_user_id" json:"user_id"` // 导入的管理员
	Name      string            `gorm:"size:64;not null" json:"name"`                      // 配置包的名称
	Author    string            `gorm:"size:64;not null;default:''" json:"author"`
	Version   int32             `gorm:"not null" json:"version"`        // 配置包的格式版本
	Digest    string            `gorm:"size:64;not null" json:"digest"` // 文件的 sha256
	Size      int64             `gorm:"not null" json:"size"`           // 文件大小, 字节
	Created   int32             `gorm:"not null;default:0" json:"created"`
	Updated   int32             `gorm:"not null;default:0" json:"updated"`
	Skipped   int32             `gorm:"not null;default:0" json:"skipped"`                                         // 跳过的项, 包括与已有配置相同的项
	Entries   []PackEntryResult `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"entries,omitempty"` // 每一项的处理结果, 列表中不返回
	CreatedAt time.Time         `gorm:"not null;default:now();autoCreateTime;index:idx_imports_created_at" json:"created_at"`
}

// TableName 自定义表名
func (Import) TableName() string { return "imports" }
