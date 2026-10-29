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

package matchadmin

import (
	"context"

	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/db"
)

// QueryMatches 获取对局列表
func QueryMatches(ctx context.Context, req *ListMatchesRequest) (int64, []match.MatchDetail, error) {
	offset := (req.Current - 1) * req.Size

	query := match.FilterMode(db.DB(ctx).Model(&match.Match{}), req.Prefix, req.Suffix)
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.UserID != nil {
		query = query.Where("id IN (SELECT match_id FROM match_players WHERE user_id = ?)", *req.UserID)
	}
	if req.OtherID != nil {
		query = query.Where("id IN (SELECT match_id FROM match_players WHERE user_id = ?)", *req.OtherID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var matches []match.Match
	if err := query.Order("id DESC").Offset(offset).Limit(req.Size).Find(&matches).Error; err != nil {
		return 0, nil, err
	}
	details, err := match.LoadDetails(db.DB(ctx), matches)
	if err != nil {
		return 0, nil, err
	}
	return total, details, nil
}
