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

package checkin

import (
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
)

// Day 签到周期中某一天的奖励, 天数从 1 开始连续
type Day struct {
	Day       int16         `gorm:"primaryKey;autoIncrement:false;check:chk_checkin_days_day,day BETWEEN 1 AND 31" json:"day"`
	Reward    reward.Bundle `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"reward"` // 数量为正数, 不能为空
	CreatedAt time.Time     `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt time.Time     `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Day) TableName() string { return "checkin_days" }

// Record 一次签到
type Record struct {
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
	Period    string    `gorm:"primaryKey;size:16" json:"period"` // 签到的日期, 与统计的按天周期相同, 如 day:2026-10-30
	Streak    int32     `gorm:"not null" json:"streak"`           // 含这一天在内的连续签到天数
	CycleDay  int16     `gorm:"not null" json:"cycle_day"`        // 这次领到的是第几天的奖励
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (Record) TableName() string { return "checkins" }
