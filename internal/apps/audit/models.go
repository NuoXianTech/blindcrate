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

package audit

import "time"

// Log 操作日志, 记录用户和管理员的操作, 只在管理后台查看
//
// 联合索引：
//   - idx_audit_logs_actor_id_created_at (actor_id, created_at)：按操作人查询
//   - idx_audit_logs_action_created_at   (action, created_at)：按操作类型查询
//   - idx_audit_logs_target              (target_type, target_id, created_at)：按操作对象查询
type Log struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	ActorID    uint64    `gorm:"not null;default:0;index:idx_audit_logs_actor_id_created_at,priority:1" json:"actor_id"`
	Scope      string    `gorm:"size:16;not null;default:''" json:"scope"`
	Action     string    `gorm:"size:64;not null;default:'';index:idx_audit_logs_action_created_at,priority:1" json:"action"`
	TargetType string    `gorm:"size:32;not null;default:'';index:idx_audit_logs_target,priority:1" json:"target_type"`
	TargetID   uint64    `gorm:"not null;default:0;index:idx_audit_logs_target,priority:2" json:"target_id"`
	Detail     string    `gorm:"type:jsonb;not null;default:'{}'" json:"detail"`
	IP         string    `gorm:"size:64;not null;default:''" json:"ip"`
	UserAgent  string    `gorm:"size:512;not null;default:''" json:"user_agent"`
	CreatedAt  time.Time `gorm:"not null;default:now();autoCreateTime;index;index:idx_audit_logs_actor_id_created_at,priority:2;index:idx_audit_logs_action_created_at,priority:2;index:idx_audit_logs_target,priority:3" json:"created_at"`
}

// TableName 自定义表名
func (Log) TableName() string { return "audit_logs" }
