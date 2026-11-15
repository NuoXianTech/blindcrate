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

package crate

import (
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
)

// Record 一次开启, 记录开启时箱子的名称、掉落池和抽中的内容, 之后修改箱子和掉落池不影响已有记录
//
// 扣除箱子和发放内容的流水, biz_id 都是开启记录 ID。
//
// 联合索引：
//   - uniq_crate_records_idempotency_key (user_id, idempotency_key)：幂等键在同一用户内唯一
type Record struct {
	ID             uint64        `gorm:"primaryKey" json:"id"`
	UserID         uint64        `gorm:"not null;index:uniq_crate_records_idempotency_key,unique,priority:1" json:"user_id"`
	IdempotencyKey string        `gorm:"size:64;not null;index:uniq_crate_records_idempotency_key,unique,priority:2" json:"-"` // 同一用户内唯一
	ItemID         uint64        `gorm:"not null" json:"item_id"`                                                              // 箱子的物品 ID
	ItemName       string        `gorm:"size:64;not null" json:"item_name"`                                                    // 开启时箱子的名称
	Pool           string        `gorm:"size:64;not null" json:"pool"`                                                         // 开启时箱子的掉落池代码
	Reward         reward.Bundle `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"reward"`                       // 抽中并发放的内容, 只有一项
	CreatedAt      time.Time     `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (Record) TableName() string { return "crate_records" }
