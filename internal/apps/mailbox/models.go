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
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"gorm.io/gorm"
)

// Letter 邮件的内容, 发给指定用户和全服邮件共用
type Letter struct {
	Sender      string        `json:"sender"` // 为空时为 "系统"
	Title       string        `json:"title"`
	Content     string        `json:"content"`
	Attachments reward.Bundle `json:"attachments"` // 数量为正数, 为空时没有附件
}

// Mail 一封邮件, 每个收件人一行
//
// 玩家删除和超出保留封数时为软删除, 来源单号仍然占用, 全服邮件不会再次展开; 过期很久后由清理任务真正删除。
type Mail struct {
	ID          uint64         `gorm:"primaryKey;index:idx_mails_user_id,priority:2" json:"id"`
	UserID      uint64         `gorm:"not null;index:idx_mails_user_id,priority:1;index:uniq_mails_source,unique,priority:3" json:"user_id"`
	Source      string         `gorm:"size:16;not null;index:uniq_mails_source,unique,priority:1" json:"source"`    // 见 RegisterSource
	SourceID    string         `gorm:"size:64;not null;index:uniq_mails_source,unique,priority:2" json:"source_id"` // 来源单号, 同一来源和用户内唯一
	Sender      string         `gorm:"size:32;not null" json:"sender"`
	Title       string         `gorm:"size:64;not null" json:"title"`
	Content     string         `gorm:"size:2000;not null;default:''" json:"content"`
	Attachments reward.Bundle  `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"attachments"`
	ReadAt      *time.Time     `json:"read_at"`
	ClaimedAt   *time.Time     `json:"claimed_at"` // 领取附件的时间, 没有附件时始终为空
	ExpiresAt   time.Time      `gorm:"not null;index" json:"expires_at"`
	CreatedAt   time.Time      `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 自定义表名
func (Mail) TableName() string { return "mails" }

// Claimable 附件在 now 时是否可以领取
func (m *Mail) Claimable(now time.Time) bool {
	return !m.Attachments.IsEmpty() && m.ClaimedAt == nil && now.Before(m.ExpiresAt)
}

// Broadcast 全服邮件, 玩家打开信箱时展开成自己的邮件, 发送后不能修改, 只能撤回
type Broadcast struct {
	ID               uint64                `gorm:"primaryKey" json:"id"`
	Sender           string                `gorm:"size:32;not null" json:"sender"`
	Title            string                `gorm:"size:64;not null" json:"title"`
	Content          string                `gorm:"size:2000;not null;default:''" json:"content"`
	Attachments      reward.Bundle         `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"attachments"`
	Requirements     []condition.Condition `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"requirements"`       // 收件条件, 需要全部满足, 展开时检查
	RegisteredBefore *time.Time            `json:"registered_before"`                                                          // 只发给这个时间之前注册的用户, 为空时之后注册的用户也能收到
	ExpiresAt        time.Time             `gorm:"not null;index" json:"expires_at"`                                           // 展开的邮件同时过期
	IdempotencyKey   string                `gorm:"size:64;not null;uniqueIndex:uniq_mail_broadcasts_idempotency_key" json:"-"` // 重复提交只发送一次
	CreatedBy        uint64                `gorm:"not null;default:0" json:"created_by"`
	RevokedAt        *time.Time            `json:"revoked_at"`
	RevokedBy        uint64                `gorm:"not null;default:0" json:"revoked_by"`
	CreatedAt        time.Time             `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (Broadcast) TableName() string { return "mail_broadcasts" }
