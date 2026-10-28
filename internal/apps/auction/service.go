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
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/ws"
	"gorm.io/gorm"
)

func init() {
	match.RegisterProvider(venue.ModePve, &provider{kind: venue.ModePve})
	match.RegisterProvider(venue.ModeRanked, &provider{kind: venue.ModeRanked})
	ws.Handle(MessageBid, handleBid)
}

// provider 按地图提供一种模式的玩法
type provider struct {
	kind string // venue.ModePve 或 venue.ModeRanked
}

// Mode 按玩法代码中的地图代码查找地图, 已删除的地图也返回, 进行中的对局还要用它打完
func (p *provider) Mode(ctx context.Context, code string) (match.Mode, error) {
	var venues []venue.Venue
	if err := db.DB(ctx).Unscoped().Where("code = ?", strings.TrimPrefix(code, p.kind+":")).Limit(1).Find(&venues).Error; err != nil {
		return nil, err
	}
	if len(venues) == 0 {
		return nil, nil
	}
	return &venueMode{kind: p.kind, venue: &venues[0]}, nil
}

// venueMode 一张地图的一种模式, 每次查找玩法时按地图的当前配置创建
//
// 准入和开局使用地图的当前配置, 开局后只使用 Match.Venue 中的快照。
type venueMode struct {
	kind  string
	venue *venue.Venue
}

func (vm *venueMode) Code() string { return vm.kind + ":" + vm.venue.Code }

func (vm *venueMode) PlayerCount() int { return playerCount }

func (vm *venueMode) MinHumans() int {
	if vm.kind == venue.ModePve {
		return 1
	}
	return rankedMinHumans
}

func (vm *venueMode) MaxHumans() int {
	if vm.kind == venue.ModePve {
		return 1
	}
	return playerCount
}

func (vm *venueMode) BotFillAfter() time.Duration {
	if vm.kind == venue.ModePve {
		return 0
	}
	return rankedBotFillAfter
}

// BotLevel 按地图配置的权重随机选出人机等级
func (vm *venueMode) BotLevel() int16 {
	return pickBotLevel(vm.venue.Bots.LevelWeights, rand.Int64N)
}

// CheckJoin 加入队列前检查地图和模式已开放、满足入场条件; 入场代价和带入资金在准入时检查
func (vm *venueMode) CheckJoin(ctx context.Context, userID uint64) error {
	return vm.checkEntry(db.DB(ctx), userID)
}

// checkEntry 地图和模式已开放且满足入场条件, 否则返回 match.Reject 的错误
func (vm *venueMode) checkEntry(tx *gorm.DB, userID uint64) error {
	v := vm.venue
	if v.DeletedAt.Valid || !v.IsEnabled {
		return match.Reject(venue.VenueDisabled)
	}
	if !v.ModeEnabled(vm.kind) {
		return match.Reject(venue.ModeDisabled)
	}
	unmet, err := condition.FirstUnmet(tx, userID, v.Requirements)
	if err != nil {
		return err
	}
	if unmet != nil {
		return match.Reject(fmt.Sprintf(RequirementUnmet, unmet.Text))
	}
	return nil
}

// Admit 检查入场; 正式局扣入场代价, 再冻结 min(可用余额, 带入资金上限), 不够起拍价时拒绝; 练习局的带入资金为上限且不冻结
func (vm *venueMode) Admit(tx *gorm.DB, m *match.Match, userID uint64) error {
	if err := vm.checkEntry(tx, userID); err != nil {
		return err
	}
	v := vm.venue
	player := &Player{MatchID: m.ID, UserID: userID, Carry: v.Auction.MaxCarry, Feedback: []Feedback{}}
	practice, err := vm.practice(tx, userID)
	if err != nil {
		return err
	}
	if practice {
		player.IsPractice = true
		return tx.Create(player).Error
	}

	bizID := strconv.FormatUint(m.ID, 10)
	remark := remarkOf(v.Name, vm.kind)
	result, replayed, err := reward.Apply(tx, &reward.ApplyRequest{
		UserID:         userID,
		Bundle:         v.EntryCost.Negate(),
		BizType:        wallet.BizMatchEntryFee,
		BizID:          bizID,
		IdempotencyKey: matchKey(m.ID, "entry", userID),
		Remark:         remark,
	})
	if err != nil {
		switch err.Error() {
		case wallet.InsufficientBalance, inventory.InsufficientQuantity:
			return match.Reject(EntryCostShort)
		}
		return err
	}
	pushResult(tx, result, replayed)

	var balance wallet.Balance
	if err := tx.Where("user_id = ? AND currency = ?", userID, v.BidCurrency).Limit(1).Find(&balance).Error; err != nil {
		return err
	}
	carry := min(balance.Balance, v.Auction.MaxCarry)
	if carry < v.Auction.StartPrice {
		return match.Reject(fmt.Sprintf(CarryShort, v.Auction.StartPrice))
	}
	txn, replayed, err := wallet.Change(tx, &wallet.ChangeRequest{
		UserID:         userID,
		Currency:       v.BidCurrency,
		Amount:         -carry,
		FrozenAmount:   carry,
		BizType:        wallet.BizMatchCarryFreeze,
		BizID:          bizID,
		IdempotencyKey: matchKey(m.ID, "freeze", userID),
		Remark:         remark,
	})
	if err != nil {
		// 读余额之后被并发的出售等扣走了
		if err.Error() == wallet.InsufficientBalance {
			return match.Reject(fmt.Sprintf(CarryShort, v.Auction.StartPrice))
		}
		return err
	}
	pushTransaction(tx, txn, replayed)

	player.EntryCost, player.Carry, player.Frozen = v.EntryCost, carry, carry
	return tx.Create(player).Error
}

// practice 是否改为练习局: 人机模式下地图只能练习, 或当天人机模式的净收益达到全局上限, 或在这张地图的净收益达到地图的上限
func (vm *venueMode) practice(tx *gorm.DB, userID uint64) (bool, error) {
	if vm.kind != venue.ModePve {
		return false, nil
	}
	v := vm.venue
	if v.Pve.PracticeOnly {
		return true, nil
	}
	economy, err := settings.Economy(tx.Statement.Context)
	if err != nil {
		return false, err
	}
	caps := []struct {
		scope string
		limit int64
	}{
		{stats.ScopeMode(venue.ModePve), int64(economy.PveDailyProfitCap)},
		{stats.ScopeVenueMode(v.Code, venue.ModePve), v.Pve.DailyProfitCap},
	}
	today := stats.PeriodDay(time.Now())
	for _, c := range caps {
		if c.limit == 0 {
			continue
		}
		stat, err := stats.Get(tx, userID, c.scope, today)
		if err != nil {
			return false, err
		}
		if stat.Profit >= c.limit {
			return true, nil
		}
	}
	return false, nil
}

// Start 快照地图配置, 生成仓库并随机公开顺序, 为人机生成估值, 进入第 1 轮出价
//
// 人机模式按地图的仓库价值系数缩放抽取次数; 掉落池已不可用或摆不下时开局失败。
func (vm *venueMode) Start(tx *gorm.DB, m *match.Match, players []match.Player) (*match.Phase, error) {
	v := vm.venue
	valueFactor := int32(venue.BasisPoints)
	if vm.kind == venue.ModePve {
		valueFactor = v.Pve.ValueFactor
	}
	generator, err := venue.NewGenerator(tx, v.Warehouse, v.BidCurrency)
	if err != nil {
		return nil, fmt.Errorf("auction: load warehouse rule of venue %s failed: %w", v.Code, err)
	}
	warehouse, err := generator.Generate(valueFactor)
	if err != nil {
		return nil, fmt.Errorf("auction: generate warehouse of venue %s failed: %w", v.Code, err)
	}

	a := &Match{
		MatchID: m.ID,
		VenueID: v.ID,
		Mode:    vm.kind,
		Venue: Snapshot{
			Code:        v.Code,
			Name:        v.Name,
			BidCurrency: v.BidCurrency,
			EntryCost:   v.EntryCost,
			Grid:        v.Warehouse.Grid,
			Auction:     v.Auction,
			Bots:        v.Bots,
		},
		Value:      warehouse.Value,
		PieceCount: int32(len(warehouse.Pieces)),
		Contenders: []int16{},
	}
	if err := tx.Create(a).Error; err != nil {
		return nil, err
	}

	itemIDs := make([]uint64, len(warehouse.Pieces))
	for i, piece := range warehouse.Pieces {
		itemIDs[i] = piece.ItemID
	}
	items, err := item.Resolve(tx, itemIDs)
	if err != nil {
		return nil, err
	}
	order := rand.Perm(len(warehouse.Pieces))
	rows := make([]WarehouseItem, len(warehouse.Pieces))
	for i, piece := range warehouse.Pieces {
		target := items[piece.ItemID]
		rows[i] = WarehouseItem{
			MatchID:      m.ID,
			RevealOrder:  int32(order[i] + 1),
			ItemID:       piece.ItemID,
			Name:         target.Name,
			ImageURL:     target.ImageURL,
			Quality:      piece.Quality,
			Width:        piece.Width,
			Height:       piece.Height,
			X:            piece.X,
			Y:            piece.Y,
			SellCurrency: piece.SellCurrency,
			SellPrice:    piece.SellPrice,
		}
	}
	if err := tx.Create(&rows).Error; err != nil {
		return nil, err
	}

	var bots []Player
	for _, player := range players {
		if !player.IsBot() {
			if err := tx.Model(&Player{}).Where("match_id = ? AND user_id = ?", m.ID, player.UserID).Update("seat", player.Seat).Error; err != nil {
				return nil, err
			}
			continue
		}
		bots = append(bots, Player{
			MatchID:  m.ID,
			Seat:     player.Seat,
			BotLevel: player.BotLevel,
			Carry:    v.Auction.MaxCarry,
			Estimate: botEstimate(&v.Bots, player.BotLevel, warehouse.Value, rand.Int64N),
			Feedback: []Feedback{},
		})
	}
	if len(bots) > 0 {
		if err := tx.Create(&bots).Error; err != nil {
			return nil, err
		}
	}

	match.AfterCommit(tx, func(ctx context.Context) { pushViews(ctx, m.ID) })
	return biddingPhase(&v.Auction, 1), nil
}

// OnTimeout 出价阶段到期时没提交的真人视为不加价, 判定本轮; 开箱阶段到期时对局结束
func (vm *venueMode) OnTimeout(tx *gorm.DB, m *match.Match, _ []match.Player) (*match.Phase, error) {
	switch m.Phase {
	case PhaseBidding:
		st, err := loadState(tx, m)
		if err != nil {
			return nil, err
		}
		return st.resolveRound(tx)
	case PhaseReveal:
		return &match.Phase{Finish: true}, nil
	}
	return nil, fmt.Errorf("auction: unknown phase %s of match %d", m.Phase, m.ID)
}

// OnCancel 还没结算时退还正式局真人的入场代价并解冻带入资金; 已结算(开箱阶段)时什么都不做
func (vm *venueMode) OnCancel(tx *gorm.DB, m *match.Match, _ []match.Player) error {
	st, err := loadState(tx, m)
	if err != nil {
		return err
	}
	a := &st.auction
	if a.Outcome != "" {
		return nil
	}
	for i := range st.players {
		p := &st.players[i]
		if p.UserID == 0 || p.IsPractice {
			continue
		}
		if err := st.refund(tx, p); err != nil {
			return err
		}
		if err := st.release(tx, p, p.Frozen); err != nil {
			return err
		}
	}
	now := time.Now()
	a.Outcome, a.SettledAt, a.Contenders = OutcomeCancelled, &now, []int16{}
	return tx.Model(a).Select("outcome", "settled_at", "contenders").Updates(a).Error
}

// biddingPhase 第 round 轮出价
func biddingPhase(rule *venue.AuctionRule, round int32) *match.Phase {
	return &match.Phase{Name: PhaseBidding, Round: round, Duration: time.Duration(rule.RoundSeconds) * time.Second}
}

// state 一局暗拍的当前状态, 在锁定对局的事务中读出后修改
type state struct {
	match     *match.Match
	auction   Match
	players   []Player       // 按座位号排序
	submitted map[int16]bool // 本轮已出价的座位
}

// loadState 读出对局的暗拍状态, 不是暗拍的对局返回 gorm.ErrRecordNotFound
func loadState(tx *gorm.DB, m *match.Match) (*state, error) {
	st := &state{match: m, submitted: map[int16]bool{}}
	if err := tx.Where("match_id = ?", m.ID).First(&st.auction).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("match_id = ?", m.ID).Order("seat").Find(&st.players).Error; err != nil {
		return nil, err
	}
	if m.Phase == PhaseBidding {
		var seats []int16
		if err := tx.Model(&Bid{}).Where("match_id = ? AND round = ?", m.ID, m.Round).Pluck("seat", &seats).Error; err != nil {
			return nil, err
		}
		for _, seat := range seats {
			st.submitted[seat] = true
		}
	}
	return st, nil
}

// player 按座位号获取玩家, 没有时返回 nil
func (st *state) player(seat int16) *Player {
	for i := range st.players {
		if st.players[i].Seat == seat {
			return &st.players[i]
		}
	}
	return nil
}

// bidders 判定用的出价者
func (st *state) bidders() []bidder {
	bidders := make([]bidder, len(st.players))
	for i, p := range st.players {
		bidders[i] = bidder{Seat: p.Seat, Bid: p.Bid}
		if p.BidAt != nil {
			bidders[i].BidAt = *p.BidAt
		}
	}
	return bidders
}

// allSubmitted 本轮可以出价的真人是否都已出价
func (st *state) allSubmitted() bool {
	for _, p := range st.players {
		if p.UserID != 0 && st.auction.CanBid(p.Seat) && !st.submitted[p.Seat] {
			return false
		}
	}
	return true
}

// placeBid 记录 p 本轮的出价, 高于当前出价时更新当前出价和提交时间
func (st *state) placeBid(tx *gorm.DB, p *Player, amount int64, at time.Time) error {
	if err := tx.Create(&Bid{MatchID: st.match.ID, Round: st.match.Round, Seat: p.Seat, Amount: amount}).Error; err != nil {
		return err
	}
	st.submitted[p.Seat] = true
	if amount <= p.Bid {
		return nil
	}
	p.Bid, p.BidAt = amount, &at
	return tx.Model(p).Select("bid", "bid_at").Updates(p).Error
}

// resolveRound 人机出价后判定本轮: 成交或流拍时结算并进入开箱, 否则给真人反馈并进入下一轮
//
// 加轮只有人机并列时没有真人可以出价, 不等待超时, 直接判定下一轮。
func (st *state) resolveRound(tx *gorm.DB) (*match.Phase, error) {
	a := &st.auction
	rule := &a.Venue.Auction
	match.AfterCommit(tx, func(ctx context.Context) { pushViews(ctx, st.match.ID) })
	for {
		round := st.match.Round
		now := time.Now()
		for i := range st.players {
			p := &st.players[i]
			if p.UserID != 0 || !a.CanBid(p.Seat) {
				continue
			}
			var amount int64
			if bid := botBid(&a.Venue.Bots, rule, round, p.Estimate, p.Bid, p.Carry, rand.Int64N); bid > p.Bid {
				amount = bid
			}
			if err := st.placeBid(tx, p, amount, now); err != nil {
				return nil, err
			}
		}

		result := judge(rule, round, a.ExtraRounds, st.bidders())
		if result.Outcome != "" {
			if err := st.settle(tx, result); err != nil {
				return nil, err
			}
			return &match.Phase{Name: PhaseReveal, Round: round, Duration: time.Duration(rule.RevealSeconds) * time.Second}, nil
		}

		bids := make(map[int16]int64, len(st.players))
		for _, p := range st.players {
			bids[p.Seat] = p.Bid
		}
		for i := range st.players {
			p := &st.players[i]
			if p.UserID == 0 {
				continue
			}
			p.Feedback = append(p.Feedback, feedbackFor(round, p.Seat, bids))
			if err := tx.Model(p).Select("feedback").Updates(p).Error; err != nil {
				return nil, err
			}
		}
		if round >= rule.Rounds {
			a.ExtraRounds++
		}
		a.Contenders = result.Contenders
		if a.Contenders == nil {
			a.Contenders = []int16{}
		}
		if err := tx.Model(a).Select("contenders", "extra_rounds").Updates(a).Error; err != nil {
			return nil, err
		}
		if st.humanCanBid() {
			return biddingPhase(rule, round+1), nil
		}
		// 出价记录按 match.Round 写入, 阶段最终由 Advance 按返回值写入
		st.match.Round = round + 1
		clear(st.submitted)
	}
}

// humanCanBid 本轮是否有真人可以出价
func (st *state) humanCanBid() bool {
	for _, p := range st.players {
		if p.UserID != 0 && st.auction.CanBid(p.Seat) {
			return true
		}
	}
	return false
}

// settle 结算: 成交时扣除成交价并发放藏品, 流拍时退还入场代价; 解冻剩余的带入资金, 写入盈亏、名次和统计
//
// 练习局只计算盈亏, 不变动资金也不写统计; 人机只计算盈亏。
func (st *state) settle(tx *gorm.DB, result verdict) error {
	a := &st.auction
	now := time.Now()
	sold := result.Outcome == OutcomeSold
	var ranks map[int16]int16
	if sold {
		ranks = rankSeats(st.bidders(), result.Winner, result.Price, a.Value)
	}

	for i := range st.players {
		p := &st.players[i]
		won := sold && p.Seat == result.Winner
		fee := amountIn(p.EntryCost, a.Venue.BidCurrency)
		switch {
		case won:
			p.Profit = a.Value - result.Price - fee
		case sold:
			p.Profit = -fee
		}
		if err := tx.Model(p).Select("profit").Updates(p).Error; err != nil {
			return err
		}
		if p.UserID == 0 || p.IsPractice {
			continue
		}

		release := p.Frozen
		if won {
			if err := st.changeWallet(tx, p, 0, -result.Price, wallet.BizMatchBidSettle, "settle"); err != nil {
				return err
			}
			release -= result.Price
			if err := st.grantWarehouse(tx, p); err != nil {
				return err
			}
		}
		if !sold {
			if err := st.refund(tx, p); err != nil {
				return err
			}
		}
		if err := st.release(tx, p, release); err != nil {
			return err
		}

		delta := stats.Delta{Matches: 1, Profit: p.Profit}
		if ranks[p.Seat] == 1 {
			delta.Wins = 1
		}
		if won {
			delta.BidWon = 1
		}
		if err := stats.Add(tx, p.UserID, stats.MatchScopes(a.Venue.Code, a.Mode), now, delta); err != nil {
			return err
		}
	}
	if err := match.SetRanks(tx, st.match.ID, ranks); err != nil {
		return err
	}

	a.Outcome, a.WinnerSeat, a.Price, a.SettledAt, a.Contenders = result.Outcome, result.Winner, result.Price, &now, []int16{}
	return tx.Model(a).Select("outcome", "winner_seat", "price", "settled_at", "contenders").Updates(a).Error
}

// grantWarehouse 把仓库里的藏品发给成交者, 已下架或已删除的藏品按开局时的回收价折成货币
func (st *state) grantWarehouse(tx *gorm.DB, p *Player) error {
	var items []WarehouseItem
	if err := tx.Where("match_id = ?", st.match.ID).Find(&items).Error; err != nil {
		return err
	}
	var bundle reward.Bundle
	prices := make(map[uint64]reward.CurrencyLine, len(items))
	for _, it := range items {
		bundle.Items = append(bundle.Items, reward.ItemLine{ItemID: it.ItemID, Quantity: 1})
		prices[it.ItemID] = reward.CurrencyLine{Currency: it.SellCurrency, Amount: it.SellPrice}
	}
	bundle, err := deliverable(tx, bundle, prices)
	if err != nil {
		return err
	}
	return st.applyReward(tx, p, bundle, wallet.BizMatchReward, "reward")
}

// refund 退还 p 的入场代价, 已下架或已删除的物品按当前回收价折成货币
func (st *state) refund(tx *gorm.DB, p *Player) error {
	if p.EntryCost.IsEmpty() {
		return nil
	}
	bundle, err := deliverable(tx, p.EntryCost, nil)
	if err != nil {
		return err
	}
	return st.applyReward(tx, p, bundle, wallet.BizMatchRefund, "refund")
}

// release 解冻 p 剩余的带入资金
func (st *state) release(tx *gorm.DB, p *Player, amount int64) error {
	if amount <= 0 {
		return nil
	}
	return st.changeWallet(tx, p, amount, -amount, wallet.BizMatchCarryRelease, "release")
}

// changeWallet 变动 p 的出价货币, 事务提交后推送
func (st *state) changeWallet(tx *gorm.DB, p *Player, amount, frozen int64, bizType, action string) error {
	txn, replayed, err := wallet.Change(tx, &wallet.ChangeRequest{
		UserID:         p.UserID,
		Currency:       st.auction.Venue.BidCurrency,
		Amount:         amount,
		FrozenAmount:   frozen,
		BizType:        bizType,
		BizID:          strconv.FormatUint(st.match.ID, 10),
		IdempotencyKey: matchKey(st.match.ID, action, p.UserID),
		Remark:         remarkOf(st.auction.Venue.Name, st.auction.Mode),
	})
	if err != nil {
		return err
	}
	pushTransaction(tx, txn, replayed)
	return nil
}

// applyReward 给 p 发放一组货币和物品, 事务提交后推送
func (st *state) applyReward(tx *gorm.DB, p *Player, bundle reward.Bundle, bizType, action string) error {
	result, replayed, err := reward.Apply(tx, &reward.ApplyRequest{
		UserID:         p.UserID,
		Bundle:         bundle,
		BizType:        bizType,
		BizID:          strconv.FormatUint(st.match.ID, 10),
		IdempotencyKey: matchKey(st.match.ID, action, p.UserID),
		Remark:         remarkOf(st.auction.Venue.Name, st.auction.Mode),
	})
	if err != nil {
		return err
	}
	pushResult(tx, result, replayed)
	return nil
}

// deliverable 发放前把已下架或已删除的物品换成货币, 这些物品不能再发放
//
// prices 为物品 ID -> 折算的货币和单价, 没有时按物品当前的回收价; 单价为 0 的物品不折算。
func deliverable(tx *gorm.DB, bundle reward.Bundle, prices map[uint64]reward.CurrencyLine) (reward.Bundle, error) {
	itemIDs := make([]uint64, len(bundle.Items))
	for i, line := range bundle.Items {
		itemIDs[i] = line.ItemID
	}
	items, err := item.Resolve(tx, itemIDs)
	if err != nil {
		return reward.Bundle{}, err
	}
	result := reward.Bundle{Currencies: append([]reward.CurrencyLine{}, bundle.Currencies...)}
	for _, line := range bundle.Items {
		target := items[line.ItemID]
		if !target.IsPlaceholder && target.IsEnabled {
			result.Items = append(result.Items, line)
			continue
		}
		price, ok := prices[line.ItemID]
		if !ok {
			price = reward.CurrencyLine{Currency: target.SellCurrency, Amount: target.SellPrice}
		}
		if price.Amount > 0 {
			result.Currencies = append(result.Currencies, reward.CurrencyLine{Currency: price.Currency, Amount: price.Amount * line.Quantity})
		}
	}
	return result, nil
}

// pushResult 事务提交后推送余额和物品变动, 重放时不推送
func pushResult(tx *gorm.DB, result *reward.Result, replayed bool) {
	if !replayed {
		match.AfterCommit(tx, result.Push)
	}
}

// pushTransaction 事务提交后推送余额变动, 重放时不推送
func pushTransaction(tx *gorm.DB, txn *wallet.Transaction, replayed bool) {
	if !replayed {
		match.AfterCommit(tx, func(ctx context.Context) { wallet.PushChanges(ctx, txn) })
	}
}

// matchKey 对局中资金变动的幂等键, 如 match:42:entry:7
func matchKey(matchID uint64, action string, userID uint64) string {
	return fmt.Sprintf("match:%d:%s:%d", matchID, action, userID)
}

// remarkOf 资金流水的备注, 如 "码头仓库·人机"
func remarkOf(venueName, kind string) string {
	return venueName + "·" + venue.ModeName(kind)
}

// bidError 出价被拒绝, 原因展示给玩家
type bidError struct {
	reason string
}

func (e *bidError) Error() string { return e.reason }

// rejectBid 拒绝出价
func rejectBid(format string, args ...any) error {
	if len(args) == 0 {
		return &bidError{reason: format}
	}
	return &bidError{reason: fmt.Sprintf(format, args...)}
}

// BidRequest 出价
type BidRequest struct {
	MatchID uint64 `json:"match_id"`
	Round   int32  `json:"round"`  // 出价的轮次, 与当前轮次不一致时拒绝, 避免上一轮的出价落到下一轮
	Amount  int64  `json:"amount"` // 0 表示不加价; 否则不低于起拍价和自己之前的出价, 不超过带入资金
}

// PlaceBid 真人出价, 本轮可以出价的真人都提交后立即判定本轮, 返回出价后自己的视角
//
// 不满足出价规则时返回 *bidError, 原因可以直接展示。
func PlaceBid(ctx context.Context, userID uint64, req *BidRequest) (*AuctionView, error) {
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
		rule := &st.auction.Venue.Auction
		switch {
		case !st.auction.CanBid(seat):
			return nil, rejectBid(NotContender)
		case st.submitted[seat]:
			return nil, rejectBid(AlreadyBid)
		case req.Amount == 0:
		case req.Amount < rule.StartPrice:
			return nil, rejectBid(BidTooLow, rule.StartPrice)
		case req.Amount < p.Bid:
			return nil, rejectBid(BidBelowPrevious, p.Bid)
		case req.Amount > p.Carry:
			return nil, rejectBid(BidOverCarry, p.Carry)
		}
		if err := st.placeBid(tx, p, req.Amount, time.Now()); err != nil {
			return nil, err
		}
		if !st.allSubmitted() {
			return nil, nil
		}
		return st.resolveRound(tx)
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

// VenueInfo 对局中展示的地图参数, 来自开局时的快照
type VenueInfo struct {
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	BidCurrency    string     `json:"bid_currency"`
	Grid           venue.Grid `json:"grid"`
	StartPrice     int64      `json:"start_price"`
	Rounds         int32      `json:"rounds"`
	QuickWin       []int32    `json:"quick_win"` // 前 Rounds-1 轮的速胜倍率, 基点
	MaxExtraRounds int32      `json:"max_extra_rounds"`
}

// RevealedPiece 已公开品质的藏品
type RevealedPiece struct {
	Order   int32 `json:"order"` // 公开顺序, 从 1 开始
	Quality int16 `json:"quality"`
}

// SelfView 自己在这局中的状态
type SelfView struct {
	Seat       int16         `json:"seat"`
	IsPractice bool          `json:"is_practice"`
	EntryCost  reward.Bundle `json:"entry_cost"` // 实际扣除的入场代价
	Carry      int64         `json:"carry"`      // 带入资金, 出价的上限
	Bid        int64         `json:"bid"`        // 当前出价, 0 表示还没出过价
	Submitted  bool          `json:"submitted"`  // 本轮是否已出价
	CanBid     bool          `json:"can_bid"`    // 现在是否可以出价
	Feedback   []Feedback    `json:"feedback"`   // 每轮结束后的反馈
}

// SeatResult 开箱后公开的一个座位的出价
type SeatResult struct {
	Seat int16 `json:"seat"`
	Bid  int64 `json:"bid"` // 最后的出价, 0 表示没出过价
}

// PieceView 开箱后公开的一件藏品
type PieceView struct {
	ItemID       uint64 `json:"item_id"`
	Name         string `json:"name"`
	ImageURL     string `json:"image_url"`
	Quality      int16  `json:"quality"`
	Width        int16  `json:"width"`
	Height       int16  `json:"height"`
	X            int16  `json:"x"`
	Y            int16  `json:"y"`
	SellCurrency string `json:"sell_currency"`
	SellPrice    int64  `json:"sell_price"` // 开局时的回收价
}

// ResultView 结算结果
type ResultView struct {
	Outcome    string       `json:"outcome"`     // sold、unsold 或 cancelled
	WinnerSeat int16        `json:"winner_seat"` // 没有成交时为 0
	Price      int64        `json:"price"`
	Value      int64        `json:"value"`  // 仓库价值
	Profit     int64        `json:"profit"` // 自己的盈亏, 出价货币, 已扣除入场代价; 练习局按正式局计算
	Pieces     []PieceView  `json:"pieces"`
	Seats      []SeatResult `json:"seats"` // 按座位号排序
}

// AuctionView 玩家看到的一局暗拍
type AuctionView struct {
	MatchID     uint64          `json:"match_id"`
	Mode        string          `json:"mode"`
	Status      string          `json:"status"`
	Phase       string          `json:"phase"`
	Round       int32           `json:"round"`
	PhaseEndsAt *time.Time      `json:"phase_ends_at"`
	Version     int64           `json:"version"` // 与对局的 version 一致, 客户端只采用 version 更大的推送
	Venue       VenueInfo       `json:"venue"`
	PieceCount  int32           `json:"piece_count"`
	Revealed    []RevealedPiece `json:"revealed"`   // 已公开品质的藏品, 第 N 轮公开前 N 件
	Contenders  []int16         `json:"contenders"` // 加轮时可以出价的座位, 为空表示所有人
	Self        SelfView        `json:"self"`
	Result      *ResultView     `json:"result"` // 结算后才有
}

// view 以 viewerID 的视角查看, 不是这局的真人玩家时返回 nil
func (st *state) view(viewerID uint64, items []WarehouseItem) *AuctionView {
	var self *Player
	for i := range st.players {
		if viewerID != 0 && st.players[i].UserID == viewerID {
			self = &st.players[i]
		}
	}
	if self == nil {
		return nil
	}
	m, a := st.match, &st.auction
	rule := &a.Venue.Auction
	submitted := st.submitted[self.Seat]
	view := &AuctionView{
		MatchID:     m.ID,
		Mode:        m.Mode,
		Status:      m.Status,
		Phase:       m.Phase,
		Round:       m.Round,
		PhaseEndsAt: m.PhaseEndsAt,
		Version:     m.Version,
		Venue: VenueInfo{
			Code:           a.Venue.Code,
			Name:           a.Venue.Name,
			BidCurrency:    a.Venue.BidCurrency,
			Grid:           a.Venue.Grid,
			StartPrice:     rule.StartPrice,
			Rounds:         rule.Rounds,
			QuickWin:       rule.QuickWin,
			MaxExtraRounds: rule.MaxExtraRounds,
		},
		PieceCount: a.PieceCount,
		Revealed:   []RevealedPiece{},
		Contenders: a.Contenders,
		Self: SelfView{
			Seat:       self.Seat,
			IsPractice: self.IsPractice,
			EntryCost:  self.EntryCost,
			Carry:      self.Carry,
			Bid:        self.Bid,
			Submitted:  submitted,
			CanBid:     m.Status == match.StatusPlaying && m.Phase == PhaseBidding && a.CanBid(self.Seat) && !submitted,
			Feedback:   self.Feedback,
		},
	}

	revealed := min(m.Round, a.PieceCount)
	if a.Outcome != "" {
		revealed = a.PieceCount
	}
	for _, it := range items {
		if it.RevealOrder <= revealed {
			view.Revealed = append(view.Revealed, RevealedPiece{Order: it.RevealOrder, Quality: it.Quality})
		}
	}
	if a.Outcome == "" {
		return view
	}

	result := &ResultView{
		Outcome:    a.Outcome,
		WinnerSeat: a.WinnerSeat,
		Price:      a.Price,
		Value:      a.Value,
		Profit:     self.Profit,
		Pieces:     make([]PieceView, len(items)),
		Seats:      make([]SeatResult, len(st.players)),
	}
	for i, it := range items {
		result.Pieces[i] = PieceView{
			ItemID:       it.ItemID,
			Name:         it.Name,
			ImageURL:     it.ImageURL,
			Quality:      it.Quality,
			Width:        it.Width,
			Height:       it.Height,
			X:            it.X,
			Y:            it.Y,
			SellCurrency: it.SellCurrency,
			SellPrice:    it.SellPrice,
		}
	}
	for i, p := range st.players {
		result.Seats[i] = SeatResult{Seat: p.Seat, Bid: p.Bid}
	}
	view.Result = result
	return view
}

// loadItems 仓库里的藏品, 按公开顺序
func loadItems(tx *gorm.DB, matchID uint64) ([]WarehouseItem, error) {
	var items []WarehouseItem
	if err := tx.Where("match_id = ?", matchID).Order("reveal_order").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// QueryAuction 获取用户参与的一局暗拍, 对局不存在、不是暗拍或用户不是该局的真人玩家时返回 match.MatchNotFound
func QueryAuction(ctx context.Context, userID, matchID uint64) (*AuctionView, error) {
	tx := db.DB(ctx)
	var m match.Match
	if err := m.Exact(tx, matchID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(match.MatchNotFound)
		}
		return nil, err
	}
	st, err := loadState(tx, &m)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New(match.MatchNotFound)
	}
	if err != nil {
		return nil, err
	}
	items, err := loadItems(tx, matchID)
	if err != nil {
		return nil, err
	}
	view := st.view(userID, items)
	if view == nil {
		return nil, errors.New(match.MatchNotFound)
	}
	return view, nil
}
