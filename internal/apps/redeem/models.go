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

package redeem

import (
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"gorm.io/gorm"
)

// Batch 兑换码批次, 一次生成的兑换码共用名称、奖励、使用条件、可使用次数和到期时间
//
// 删除为软删除, 兑换记录仍然对应得上; 删除后批次中的兑换码不能再兑换, 也不能被新的批次使用。
type Batch struct {
	ID             uint64                `gorm:"primaryKey" json:"id"`
	Name           string                `gorm:"size:64;not null" json:"name"`                                                   // 兑换成功后展示给玩家, 也是奖励邮件的标题
	Reward         reward.Bundle         `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"reward"`                 // 兑换一次得到的奖励, 数量为正数, 不能为空
	Requirements   []condition.Condition `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"requirements"`           // 使用条件, 需要全部满足
	MaxUses        int32                 `gorm:"not null;check:chk_redeem_batches_max_uses,max_uses >= 0" json:"max_uses"`       // 每个兑换码可以被多少个用户兑换, 1 为单次使用, 0 表示不限
	UserLimit      int32                 `gorm:"not null;check:chk_redeem_batches_user_limit,user_limit >= 0" json:"user_limit"` // 每个用户最多兑换这个批次中的几个兑换码, 0 表示不限
	CodeCount      int32                 `gorm:"not null" json:"code_count"`                                                     // 兑换码的个数, 生成后不变
	ExpiresAt      *time.Time            `json:"expires_at"`                                                                     // 到期时间, 不包含, 为空时不过期
	IsEnabled      bool                  `gorm:"not null;default:false" json:"is_enabled"`                                       // 停用后不能兑换
	IdempotencyKey string                `gorm:"size:64;not null;uniqueIndex:uniq_redeem_batches_idempotency_key" json:"-"`      // 重复提交只生成一次
	CreatedBy      uint64                `gorm:"not null;default:0" json:"created_by"`
	CreatedAt      time.Time             `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time             `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt        `gorm:"index" json:"-"`
}

// TableName 自定义表名
func (Batch) TableName() string { return "redeem_batches" }

// Exact 按 ID 获取批次
func (b *Batch) Exact(tx *gorm.DB, id uint64) error {
	return tx.Where("id = ?", id).First(b).Error
}

// Expired 批次在 now 时是否已到期
func (b *Batch) Expired(now time.Time) bool {
	return b.ExpiresAt != nil && !now.Before(*b.ExpiresAt)
}

// Code 一个兑换码, 属于一个批次
type Code struct {
	ID      uint64 `gorm:"primaryKey" json:"id"`
	BatchID uint64 `gorm:"not null;index" json:"batch_id"`
	Code    string `gorm:"size:32;not null;uniqueIndex:uniq_redeem_codes_code" json:"code"` // 规范化后的兑换码, 见 Normalize; 在已删除的批次中也唯一
	Used    int32  `gorm:"not null;default:0" json:"used"`                                  // 已兑换的次数
}

// TableName 自定义表名
func (Code) TableName() string { return "redeem_codes" }

// Record 一次兑换, 同一个兑换码同一用户只有一行
type Record struct {
	ID        uint64        `gorm:"primaryKey;index:idx_redeem_records_user_id,priority:2" json:"id"`
	BatchID   uint64        `gorm:"not null;index:idx_redeem_records_batch_user,priority:1" json:"batch_id"`
	CodeID    uint64        `gorm:"not null;index:uniq_redeem_records_code_user,unique,priority:1" json:"code_id"`
	UserID    uint64        `gorm:"not null;index:idx_redeem_records_user_id,priority:1;index:idx_redeem_records_batch_user,priority:2;index:uniq_redeem_records_code_user,unique,priority:2" json:"user_id"`
	Reward    reward.Bundle `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"reward"` // 兑换时批次的奖励, 以邮件附件发给了用户
	CreatedAt time.Time     `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (Record) TableName() string { return "redeem_records" }
