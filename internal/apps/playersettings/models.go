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
// Package playersettings 保存账号自己的游戏偏好, 与管理员系统设置分开。
package playersettings

import "time"

type Settings struct {
	UserID             uint64     `gorm:"primaryKey;autoIncrement:false" json:"-"`
	KeepOneCollectible bool       `gorm:"not null;default:false" json:"keep_one_collectible"`
	Version            int64      `gorm:"not null;default:0;check:chk_player_game_settings_version,version >= 0" json:"version"`
	UpdatedAt          *time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

func (Settings) TableName() string { return "player_game_settings" }

const (
	Changed        = "游戏设置已在其他页面修改, 请确认最新设置后重试"
	MessageChanged = "player_settings:changed"
	ActionUpdate   = "player_settings.update"
)
