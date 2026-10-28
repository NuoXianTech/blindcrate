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
	"context"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/ws"
)

// pushUpdate 把对局推送给其中的每个真人玩家, 每人看到自己的视角; 需要在事务提交后调用, 失败只记录日志
func pushUpdate(ctx context.Context, m *Match, players []Player) {
	profiles, err := loadProfiles(db.DB(ctx), players)
	if err != nil {
		logger.ErrorF(ctx, "[Match] load profiles of match %d failed: %v", m.ID, err)
		return
	}
	detail := MatchDetail{Match: *m, Players: make([]PlayerDetail, len(players))}
	for i, player := range players {
		detail.Players[i] = newPlayerDetail(player, profiles)
	}

	for _, player := range players {
		if player.IsBot() {
			continue
		}
		if err := ws.SendToUser(ctx, player.UserID, MessageUpdated, detail.View(player.UserID)); err != nil {
			logger.ErrorF(ctx, "[Match] push match %d to %d failed: %v", m.ID, player.UserID, err)
		}
	}
}
