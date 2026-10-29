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

package auction

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/ws"
)

// handleBid 处理客户端的 auction:bid 消息, 回复出价后自己的视角
func handleBid(ctx context.Context, client *ws.Client, data json.RawMessage) (interface{}, error) {
	var req BidRequest
	if err := json.Unmarshal(data, &req); err != nil || req.MatchID == 0 || req.Round < 1 || req.Amount < 0 {
		return nil, errors.New(auth.InvalidParams)
	}
	view, err := PlaceBid(ctx, client.UserID(), &req)
	if err != nil {
		var rejected *bidError
		if errors.As(err, &rejected) {
			return nil, err
		}
		logger.ErrorF(ctx, "[Auction] bid of %d in match %d failed: %v", client.UserID(), req.MatchID, err)
		return nil, errors.New(auth.InternalError)
	}
	return view, nil
}

// handleUse 处理客户端的 auction:use 消息, 回复使用后自己的视角
func handleUse(ctx context.Context, client *ws.Client, data json.RawMessage) (interface{}, error) {
	var req UseRequest
	if err := json.Unmarshal(data, &req); err != nil || req.MatchID == 0 || req.Round < 1 || req.ItemID == 0 {
		return nil, errors.New(auth.InvalidParams)
	}
	view, err := UseItem(ctx, client.UserID(), &req)
	if err != nil {
		var rejected *bidError
		if errors.As(err, &rejected) {
			return nil, err
		}
		logger.ErrorF(ctx, "[Auction] use item %d of %d in match %d failed: %v", req.ItemID, client.UserID(), req.MatchID, err)
		return nil, errors.New(auth.InternalError)
	}
	return view, nil
}

// pushViews 把暗拍推送给对局中的每个真人, 每人看到自己的视角; 需要在事务提交后调用, 失败只记录日志
func pushViews(ctx context.Context, matchID uint64) {
	tx := db.DB(ctx)
	var m match.Match
	if err := m.Exact(tx, matchID); err != nil {
		logger.ErrorF(ctx, "[Auction] load match %d failed: %v", matchID, err)
		return
	}
	st, err := loadState(tx, &m)
	if err != nil {
		logger.ErrorF(ctx, "[Auction] load auction of match %d failed: %v", matchID, err)
		return
	}
	items, err := loadItems(tx, matchID)
	if err != nil {
		logger.ErrorF(ctx, "[Auction] load warehouse of match %d failed: %v", matchID, err)
		return
	}
	uses, err := loadUses(tx, matchID)
	if err != nil {
		logger.ErrorF(ctx, "[Auction] load uses of match %d failed: %v", matchID, err)
		return
	}

	for _, p := range st.players {
		if p.UserID == 0 {
			continue
		}
		if err := ws.SendToUser(ctx, p.UserID, MessageUpdated, st.view(p.UserID, items, uses)); err != nil {
			logger.ErrorF(ctx, "[Auction] push match %d to %d failed: %v", matchID, p.UserID, err)
		}
	}
}
