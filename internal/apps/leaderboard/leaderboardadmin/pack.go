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

package leaderboardadmin

import (
	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"gorm.io/gorm"
)

func init() {
	pack.RegisterTable(pack.Table[leaderboard.Board, LeaderboardRequest]{
		Code: pack.PackLeaderboards, Label: "榜单", Hint: "按代码对应", Order: pack.OrderLeaderboards,
		Query: func(tx *gorm.DB) *gorm.DB { return tx.Order("sort_order, id") },
		Pack: func(b *leaderboard.Board, _ func(string) string) LeaderboardRequest {
			return LeaderboardRequest{
				Code:        b.Code,
				Name:        b.Name,
				Description: b.Description,
				Metric:      b.Metric,
				Scope:       b.Scope,
				Period:      b.Period,
				Size:        b.Size,
				MinMatches:  b.MinMatches,
				SortOrder:   b.SortOrder,
				IsEnabled:   b.IsEnabled,
			}
		},
		Key:    func(req *LeaderboardRequest) string { return req.Code },
		Name:   func(req *LeaderboardRequest) string { return req.Name },
		Taken:  leaderboard.CodeTaken,
		Rename: func(req *LeaderboardRequest, code string) { req.Code = code },
		Renamed: func(req *LeaderboardRequest, section, old, next string) {
			if section == pack.PackVenues {
				req.Scope = pack.RenameScope(req.Scope, old, next)
			}
		},
		Find: func(tx *gorm.DB, req *LeaderboardRequest, current *leaderboard.Board) (bool, error) {
			err := tx.Where("code = ?", req.Code).Limit(1).Find(current).Error
			return current.ID != 0, err
		},
		Lock: func(tx *gorm.DB, current *leaderboard.Board) error { return LockLeaderboard(tx, current, current.ID) },
		Save: func(tx *gorm.DB, _ *pack.Run, current *leaderboard.Board, req *LeaderboardRequest) error {
			target := current
			if target == nil {
				target = &leaderboard.Board{}
			}
			if err := ApplyLeaderboard(tx, target, req); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	})
}
