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

package stats

import "time"

// Stat 用户在某个统计范围和周期内的一个指标, 首次写入时创建
type Stat struct {
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
	Scope     string    `gorm:"primaryKey;size:64;index:idx_stats_board,priority:1" json:"scope"`           // 统计范围, 如 all、mode:ranked、venue:dock
	Period    string    `gorm:"primaryKey;size:32;index:idx_stats_board,priority:2" json:"period"`          // 统计周期, 如 total、day:2026-10-28、week:2026-W44
	Metric    string    `gorm:"primaryKey;size:64;index:idx_stats_board,priority:3" json:"metric"`          // 指标, 如 matches、profit:chip
	Value     int64     `gorm:"not null;default:0;index:idx_stats_board,priority:4,sort:desc" json:"value"` // 累计值, 金额按最小单位计, 可以为负
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Stat) TableName() string { return "stats" }

// Metric 可以在条件中使用的计数指标
type Metric struct {
	Key   string `json:"key"`
	Label string `json:"label"` // 后台和条件描述中展示的名称, 如 "对局数"
}
