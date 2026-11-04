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
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"gorm.io/gorm"
)

// IntelRequest 向主持人买情报的请求
type IntelRequest struct {
	MatchID uint64 `json:"match_id"`
	Round   int32  `json:"round"` // 购买时的轮次, 与当前轮次不一致时拒绝
}

// BuyIntel 真人在出价阶段向主持人买一次情报: 从可用余额扣除地图的情报价格, 按轮次给出情报并保存, 返回购买后自己的视角
//
// 每轮最多买一次, 每局最多 HostRule.IntelLimit 次; 情报的种类随轮次升级, 见 intelFor。带入资金已冻结, 不能用来买情报。
// 对局被取消时退还, 流拍不退; 结算时计入盈亏。不满足规则时返回 *bidError, 原因可以直接展示。
func BuyIntel(ctx context.Context, userID uint64, req *IntelRequest) (*AuctionView, error) {
	_, err := match.Advance(ctx, req.MatchID, func(tx *gorm.DB, m *match.Match, players []match.Player) (*match.Phase, error) {
		var seat int16
		for _, player := range players {
			if !player.IsBot() && player.UserID == userID {
				seat = player.Seat
			}
		}
		if seat == 0 {
			return nil, rejectBid(match.MatchNotFound)
		}
		if m.Phase != PhaseBidding {
			return nil, rejectBid(NotBidding)
		}
		if m.Round != req.Round || (m.PhaseEndsAt != nil && time.Now().After(*m.PhaseEndsAt)) {
			return nil, rejectBid(RoundClosed)
		}
		st, err := loadState(tx, m)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rejectBid(match.MatchNotFound)
		}
		if err != nil {
			return nil, err
		}

		p := st.player(seat)
		a := &st.auction
		price, limit := a.Venue.Auction.IntelPrice, a.Venue.Host.IntelLimit
		switch {
		case p.IsPractice:
			return nil, rejectBid(IntelPractice)
		case !a.CanBid(seat):
			return nil, rejectBid(NotContender)
		case price <= 0 || limit <= 0:
			return nil, rejectBid(IntelDisabled)
		}
		var uses []Use
		if err := tx.Where("match_id = ? AND seat = ?", m.ID, seat).Order("id").Find(&uses).Error; err != nil {
			return nil, err
		}
		var bought int32
		for _, use := range uses {
			if !use.IsIntel() {
				continue
			}
			if use.Round == m.Round {
				return nil, rejectBid(IntelRepeated)
			}
			bought++
		}
		if bought >= limit {
			return nil, rejectBid(IntelLimitReached, limit)
		}

		txn, replayed, err := wallet.Change(tx, &wallet.ChangeRequest{
			UserID:         userID,
			Currency:       a.Venue.BidCurrency,
			Amount:         -price,
			BizType:        wallet.BizMatchIntel,
			BizID:          strconv.FormatUint(m.ID, 10),
			IdempotencyKey: matchKey(m.ID, fmt.Sprintf("intel_%d", m.Round), userID),
			Remark:         remarkOf(a.Venue.Name, st.kind()),
		})
		if err != nil {
			if err.Error() == wallet.InsufficientBalance {
				return nil, rejectBid(IntelBalanceShort, price)
			}
			return nil, err
		}
		pushTransaction(tx, txn, replayed)

		items, err := loadItems(tx, m.ID)
		if err != nil {
			return nil, err
		}
		useCtx := newUseContext(m.Round, a.PieceCount, a.Value, items, uses, rand.Int64N)
		code, intel, err := intelFor(m.Round, a.Venue.Auction.Rounds, a.Venue.Host.IntelPrecision, useCtx)
		if err != nil {
			return nil, err
		}
		return nil, tx.Create(&Use{
			MatchID: m.ID,
			Seat:    seat,
			Round:   m.Round,
			Name:    intelName,
			Effect:  code,
			Params:  map[string]any{},
			Result:  intel,
			Cost:    price,
		}).Error
	})
	if err != nil {
		switch err.Error() {
		case match.MatchNotFound, match.MatchNotPlaying:
			return nil, &bidError{reason: err.Error()}
		}
		return nil, err
	}
	return QueryAuction(ctx, userID, req.MatchID)
}

// intelFor 第 round 轮(共 rounds 轮)买到的情报, 返回效果代码和情报, 随轮次升级:
// 前三分之一的轮次看一件还不知道品质的藏品的品质, 中间三分之一看一件还不知道名称的藏品的名称, 之后为仓库价值的区间(误差 precision);
// 这一档已经没有可以看的藏品时升到下一档。加轮按最后一档。
func intelFor(round, rounds, precision int32, ctx *UseContext) (string, Intel, error) {
	if round*3 <= rounds {
		if intel := randomQuality(ctx); len(intel.Pieces) > 0 {
			return EffectRevealQuality, intel, nil
		}
	}
	if round*3 <= rounds*2 {
		intel, err := revealName(ctx, &RevealNameParams{Count: 1})
		if err != nil {
			return "", Intel{}, err
		}
		if len(intel.Pieces) > 0 {
			return EffectRevealName, intel, nil
		}
	}
	intel, err := valueRange(ctx, &ValueRangeParams{Precision: precision})
	return EffectValueRange, intel, err
}

// randomQuality 随机看一件还不知道品质的藏品的品质; 与鉴定品质不同, 不优先选最晚公开的
func randomQuality(ctx *UseContext) Intel {
	var unknown []WarehouseItem
	for _, it := range ctx.Pieces {
		if !ctx.Known[it.ScanOrder] {
			unknown = append(unknown, it)
		}
	}
	if len(unknown) == 0 {
		return Intel{}
	}
	it := unknown[ctx.Rand(int64(len(unknown)))]
	return Intel{Pieces: []PieceIntel{{ScanOrder: it.ScanOrder, Quality: it.Quality}}}
}

// intelCost p 在这局买情报付的出价货币
func intelCost(uses []Use, seat int16) int64 {
	var total int64
	for _, use := range uses {
		if use.Seat == seat && use.IsIntel() {
			total += use.Cost
		}
	}
	return total
}

// refundIntel 对局被取消时退还 p 买情报付的出价货币
func (st *state) refundIntel(tx *gorm.DB, p *Player, uses []Use) error {
	cost := intelCost(uses, p.Seat)
	if cost <= 0 {
		return nil
	}
	return st.changeWallet(tx, p, cost, 0, wallet.BizMatchRefund, "intel_refund")
}
