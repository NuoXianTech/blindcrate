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

package mission

import (
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"gorm.io/gorm"
)

// Mission 任务, 进度为 stats 中 Scope、周期和 Metric 对应的值, 达到 Target 后可以领取 Reward
//
// 删除为软删除, 领取记录仍然对应得上; 代码在已删除的任务中也唯一。
type Mission struct {
	ID           uint64                `gorm:"primaryKey" json:"id"`
	Code         string                `gorm:"size:32;not null;uniqueIndex:uniq_missions_code" json:"code"` // 自然键, 创建后不能修改
	Name         string                `gorm:"size:64;not null" json:"name"`
	Description  string                `gorm:"size:255;not null;default:''" json:"description"`
	Cycle        string                `gorm:"size:16;not null" json:"cycle"`                                        // daily、weekly 或 once
	Scope        string                `gorm:"size:64;not null" json:"scope"`                                        // 统计范围, 如 all、mode:ranked、venue:dock
	Metric       string                `gorm:"size:64;not null" json:"metric"`                                       // 统计指标, 如 matches、profit:chip
	Target       int64                 `gorm:"not null;check:chk_missions_target,target > 0" json:"target"`          // 指标达到这个值时完成
	Reward       reward.Bundle         `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"reward"`       // 数量为正数, 不能为空
	Requirements []condition.Condition `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"requirements"` // 领取条件, 需要全部满足
	SortOrder    int32                 `gorm:"not null;default:0" json:"sort_order"`                                 // 同一周期内从小到大排列
	IsEnabled    bool                  `gorm:"not null;default:false" json:"is_enabled"`                             // 停用后玩家看不到, 也不能领取
	CreatedAt    time.Time             `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time             `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt        `gorm:"index" json:"-"`
}

// TableName 自定义表名
func (Mission) TableName() string { return "missions" }

// Exact 按 ID 获取任务
func (m *Mission) Exact(tx *gorm.DB, id uint64) error {
	return tx.Where("id = ?", id).First(m).Error
}

// ClaimRecord 一次领取, 同一任务同一用户在同一统计周期内只有一行
type ClaimRecord struct {
	MissionID uint64    `gorm:"primaryKey;autoIncrement:false" json:"mission_id"`
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
	Period    string    `gorm:"primaryKey;size:32" json:"period"` // 领取时的统计周期, 如 day:2026-10-30、week:2026-W44、total
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (ClaimRecord) TableName() string { return "mission_claims" }
