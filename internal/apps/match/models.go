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

package match

import (
	"time"

	"gorm.io/gorm"
)

// Match 一局对战, 由匹配创建, 创建时即开始
//
// 索引 idx_matches_status_phase_ends_at 用于巡检到期仍未推进的对局。
type Match struct {
	ID          uint64     `gorm:"primaryKey" json:"id"`
	Mode        string     `gorm:"size:32;not null" json:"mode"`                                                     // 玩法代码
	Status      string     `gorm:"size:16;not null;index:idx_matches_status_phase_ends_at,priority:1" json:"status"` // playing、finished 或 cancelled
	Phase       string     `gorm:"size:32;not null;default:''" json:"phase"`                                         // 当前阶段, 由玩法定义; 结束后保留最后一个阶段
	Round       int32      `gorm:"not null;default:0" json:"round"`                                                  // 当前轮次, 由玩法定义
	PhaseEndsAt *time.Time `gorm:"index:idx_matches_status_phase_ends_at,priority:2" json:"phase_ends_at"`           // 当前阶段的截止时间, 不限时或已结束时为空
	Version     int64      `gorm:"not null;default:0" json:"version"`                                                // 每次切换阶段加 1, 用于丢弃过期的超时任务和推送
	ReleasedAt  *time.Time `json:"released_at"`                                                                      // 玩家可以离开的时间, 之后不再算作进行中的对局; 为空表示还没释放
	CreatedAt   time.Time  `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
	FinishedAt  *time.Time `json:"finished_at"` // 结束或取消的时间
}

// TableName 自定义表名
func (Match) TableName() string { return "matches" }

// Exact 按 ID 获取对局
func (m *Match) Exact(tx *gorm.DB, id uint64) error {
	if err := tx.Where("id = ?", id).First(m).Error; err != nil {
		return err
	}
	return nil
}

// Player 对局中的一个座位, 坐的是真人或人机
//
// 唯一索引 uniq_match_players_match_id_seat 保证同一局的座位不重复。
type Player struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	MatchID   uint64    `gorm:"not null;uniqueIndex:uniq_match_players_match_id_seat,priority:1" json:"match_id"`
	Seat      int16     `gorm:"not null;uniqueIndex:uniq_match_players_match_id_seat,priority:2" json:"seat"`                                                  // 座位号, 从 1 开始, 随机分配
	UserID    uint64    `gorm:"not null;default:0;index:idx_match_players_user_id;check:chk_match_players_bot,(user_id = 0) = (bot_level > 0)" json:"user_id"` // 人机为 0
	BotName   string    `gorm:"size:32;not null;default:''" json:"bot_name"`                                                                                   // 人机的昵称
	BotLevel  int16     `gorm:"not null;default:0" json:"bot_level"`                                                                                           // 人机等级, 真人为 0
	Rank      int16     `gorm:"not null;default:0" json:"rank"`                                                                                                // 结束时的名次, 从 1 开始, 0 表示没有名次
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (Player) TableName() string { return "match_players" }

// IsBot 是否为人机
func (p *Player) IsBot() bool {
	return p.UserID == 0
}
