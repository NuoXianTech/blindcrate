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

// UserStat 用户在某个统计范围和周期内的对局统计, 首次写入时创建
type UserStat struct {
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
	Scope     string    `gorm:"primaryKey;size:64" json:"scope"`   // 统计范围, 如 all、mode:ranked、venue:dock
	Period    string    `gorm:"primaryKey;size:32" json:"period"`  // 统计周期, 如 total、day:2026-10-28
	Matches   int64     `gorm:"not null;default:0" json:"matches"` // 完成的对局数
	Wins      int64     `gorm:"not null;default:0" json:"wins"`    // 胜利次数, 名次为第 1 的对局数
	Profit    int64     `gorm:"not null;default:0" json:"profit"`  // 净收益, 按最小单位计, 可以为负
	BidWon    int64     `gorm:"not null;default:0" json:"bid_won"` // 成交次数
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (UserStat) TableName() string { return "user_stats" }
