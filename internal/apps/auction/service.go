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
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/effect"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/apps/ranked"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/ws"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	match.RegisterProvider(venue.ModePve, &provider{kind: venue.ModePve})
	match.RegisterProvider(venue.ModeRanked, &provider{kind: venue.ModeRanked})
	stats.RegisterMetric(MetricBidWon, "成交次数")
	ws.Handle(MessageBid, handleBid)
	ws.Handle(MessageUse, handleUse)
	registerEffects()
	registerStrategies()
	venue.RegisterBalancer(balance)
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

// Summary 对局记录中暗拍的摘要, 金额为出价货币的最小单位
type Summary struct {
	Venue       string `json:"venue"`      // 地图代码
	VenueName   string `json:"venue_name"` // 开局时的地图名称
	BidCurrency string `json:"bid_currency"`
	Outcome     string `json:"outcome"`      // 拍场结算前为空, 之后为 sold(至少成交一个拍品)、unsold 或 cancelled
	LotCount    int32  `json:"lot_count"`    // 拍品数
	WonLots     int32  `json:"won_lots"`     // 自己拿下的拍品数
	Spent       int64  `json:"spent"`        // 自己拿下的拍品的成交价之和
	Profit      int64  `json:"profit"`       // 自己的盈亏, 结算前为 0
	Rank        int16  `json:"rank"`         // 自己的名次, 全部流拍时为 0
	IsPractice  bool   `json:"is_practice"`  // 自己打的是否为练习局
	RatingDelta int32  `json:"rating_delta"` // 自己段位分的变化, 只有排位且至少成交一个拍品时有
}

// Summaries 实现 match.Summarizer, 为对局记录附带地图、拍品数、自己拿下的拍品数和盈亏
func (p *provider) Summaries(ctx context.Context, viewerID uint64, matchIDs []uint64) (map[uint64]any, error) {
	tx := db.DB(ctx)
	var matches []Match
	if err := tx.Select("match_id, venue, outcome, lot_count").Where("match_id IN ?", matchIDs).Find(&matches).Error; err != nil {
		return nil, err
	}
	var players []Player
	if err := tx.Where("match_id IN ? AND user_id = ?", matchIDs, viewerID).Find(&players).Error; err != nil {
		return nil, err
	}
	mine := make(map[uint64]*Player, len(players))
	for i := range players {
		mine[players[i].MatchID] = &players[i]
	}
	var won []struct {
		MatchID uint64
		Lots    int32
	}
	if err := tx.Model(&Lot{}).Select("auction_lots.match_id, COUNT(*) AS lots").
		Joins("JOIN auction_players ap ON ap.match_id = auction_lots.match_id AND ap.seat = auction_lots.winner_seat").
		Where("auction_lots.match_id IN ? AND auction_lots.outcome = ? AND ap.user_id = ?", matchIDs, OutcomeSold, viewerID).
		Group("auction_lots.match_id").Scan(&won).Error; err != nil {
		return nil, err
	}
	wonLots := make(map[uint64]int32, len(won))
	for _, w := range won {
		wonLots[w.MatchID] = w.Lots
	}
	summaries := make(map[uint64]any, len(matches))
	for _, m := range matches {
		summary := Summary{
			Venue:       m.Venue.Code,
			VenueName:   m.Venue.Name,
			BidCurrency: m.Venue.BidCurrency,
			Outcome:     m.Outcome,
			LotCount:    m.LotCount,
		}
		if player := mine[m.MatchID]; player != nil {
			summary.WonLots = wonLots[m.MatchID]
			summary.Spent = player.Spent
			summary.Profit = player.Profit
			summary.Rank = player.Rank
			summary.IsPractice = player.IsPractice
			summary.RatingDelta = player.RatingDelta
		}
		summaries[m.MatchID] = summary
	}
	return summaries, nil
}

// venueMode 一张地图的一种模式, 每次查找玩法时按地图的当前配置创建
//
// 准入和开局使用地图的当前配置, 开局后只使用 Match.Venue 和 Lot 中的快照。
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

// CheckJoin 加入队列前检查排位没有因挂机禁赛, 地图和模式已开放、满足入场条件; 入场代价和带入资金在准入时检查
func (vm *venueMode) CheckJoin(ctx context.Context, userID uint64) error {
	if vm.kind == venue.ModeRanked {
		reason, err := ranked.AfkBanReason(ctx, userID)
		if err != nil {
			return err
		}
		if reason != "" {
			return match.Reject(reason)
		}
	}
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

// Admit 检查入场; 带入资金为玩家在这张地图选的金额(见 SetCarry), 没选过时为上限, 限制在 [起拍价, 上限] 内;
// 正式局扣入场费和入场代价, 再冻结带入资金, 可用余额不够时拒绝; 练习局不冻结
func (vm *venueMode) Admit(tx *gorm.DB, m *match.Match, userID uint64) error {
	if err := vm.checkEntry(tx, userID); err != nil {
		return err
	}
	v := vm.venue
	carry, err := carryOf(tx, userID, v)
	if err != nil {
		return err
	}
	player := &Player{MatchID: m.ID, UserID: userID, Carry: carry}
	practice, err := vm.practice(tx, userID)
	if err != nil {
		return err
	}
	if practice {
		player.IsPractice = true
		return tx.Create(player).Error
	}

	var balance wallet.Balance
	if err := tx.Where("user_id = ? AND currency = ?", userID, v.BidCurrency).Limit(1).Find(&balance).Error; err != nil {
		return err
	}
	// 先检查带入资金, 不够时不扣入场费
	if balance.Balance < carry+v.Auction.EntryFee {
		return match.Reject(fmt.Sprintf(CarryShort, carry))
	}
	bizID := strconv.FormatUint(m.ID, 10)
	remark := remarkOf(v.Name, vm.kind)
	entry := v.Entry()
	result, replayed, err := reward.Apply(tx, &reward.ApplyRequest{
		UserID:         userID,
		Bundle:         entry.Negate(),
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
			return match.Reject(fmt.Sprintf(CarryShort, carry))
		}
		return err
	}
	pushTransaction(tx, txn, replayed)

	player.EntryCost, player.Frozen = entry, carry
	return tx.Create(player).Error
}

// carryOf 玩家在地图 v 的带入资金: 选过的金额, 没选过时为上限, 限制在 [起拍价, 上限] 内
func carryOf(tx *gorm.DB, userID uint64, v *venue.Venue) (int64, error) {
	rule := &v.Auction
	var prefs []CarryPreference
	if err := tx.Where("user_id = ? AND venue = ?", userID, v.Code).Limit(1).Find(&prefs).Error; err != nil {
		return 0, err
	}
	if len(prefs) == 0 {
		return rule.MaxCarry, nil
	}
	return min(max(prefs[0].Amount, rule.StartPrice), rule.MaxCarry), nil
}

// CarryRequest 选择在一张地图的带入金额
type CarryRequest struct {
	Venue  string `json:"venue" binding:"required,max=32"` // 地图代码
	Amount int64  `json:"amount" binding:"min=0"`          // 超出 [起拍价, 带入资金上限] 时按边界
}

// SetCarry 保存玩家在一张地图的带入金额, 之后准入时按它冻结; 地图之后调整了上限时按新的上限截断
//
// 地图不存在或已关闭时返回 venue.VenueDisabled。
func SetCarry(ctx context.Context, userID uint64, req *CarryRequest) (*CarryPreference, error) {
	tx := db.DB(ctx)
	var v venue.Venue
	if err := v.ExactCode(tx, req.Venue); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(venue.VenueDisabled)
		}
		return nil, err
	}
	if !v.IsEnabled {
		return nil, errors.New(venue.VenueDisabled)
	}
	pref := &CarryPreference{UserID: userID, Venue: v.Code, Amount: min(max(req.Amount, v.Auction.StartPrice), v.Auction.MaxCarry)}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "venue"}},
		DoUpdates: clause.AssignmentColumns([]string{"amount", "updated_at"}),
	}).Create(pref).Error; err != nil {
		return nil, err
	}
	return pref, nil
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
	caps, err := venue.PveProfitCaps(tx, userID, v, time.Now())
	if err != nil {
		return false, err
	}
	for _, c := range caps {
		if c.Reached() {
			return true, nil
		}
	}
	return false, nil
}

// Start 快照地图配置, 抽出拍品数并逐个生成拍品(见 newLots), 按地图配置的权重为人机选出风格, 进入第 1 个拍品的第 1 轮出价
//
// 人机模式按地图的仓库价值系数缩放抽取次数; 多拍品时人机的带入资金在 BotCarry × [人机带入下限, 100%] 内随机。
// 掉落池已不可用或摆不下、人机风格已不可用时开局失败。
func (vm *venueMode) Start(tx *gorm.DB, m *match.Match, players []match.Player) (*match.Phase, error) {
	v := vm.venue
	balance, err := settings.Balance.Get(tx.Statement.Context)
	if err != nil {
		return nil, err
	}
	a := &Match{
		MatchID: m.ID,
		VenueID: v.ID,
		Venue: Snapshot{
			Code:        v.Code,
			Name:        v.Name,
			BidCurrency: v.BidCurrency,
			EntryCost:   v.Entry(),
			Auction:     v.Auction,
			Consumables: v.Consumables,
			Host:        v.Host,
			GapSeconds:  v.Lots.GapSeconds,
			HypeDecay:   int32(balance.HypeLotDecay),
			TierNames:   map[string]string{},
		},
		Lot: 1,
	}
	for _, t := range v.Lots.Tiers {
		a.Venue.TierNames[t.Code] = t.Name
	}
	lots, items, err := vm.newLots(tx, m.ID)
	if err != nil {
		return nil, err
	}
	a.LotCount = int32(len(lots))
	if err := tx.Create(a).Error; err != nil {
		return nil, err
	}
	if err := tx.Create(&lots).Error; err != nil {
		return nil, err
	}
	if err := tx.Create(&items).Error; err != nil {
		return nil, err
	}
	if err := startHost(tx, a, &lots[0], lotItems(items, 1)); err != nil {
		return nil, err
	}

	botCount := 0
	for _, player := range players {
		if player.IsBot() {
			botCount++
		}
	}
	personas, err := bot.Assign(tx, v.Bots.Weights(vm.kind), botCount)
	if err != nil {
		return nil, fmt.Errorf("auction: assign bot personas of venue %s failed: %w", v.Code, err)
	}
	var bots []Player
	for _, player := range players {
		if !player.IsBot() {
			if err := tx.Model(&Player{}).Where("match_id = ? AND user_id = ?", m.ID, player.UserID).Update("seat", player.Seat).Error; err != nil {
				return nil, err
			}
			continue
		}
		carry := v.Auction.BotBudget()
		if !v.Lots.Single() {
			floor := int64(balance.BotCarryFloor)
			carry = max(carry*(floor+rand.Int64N(venue.BasisPoints-floor+1))/venue.BasisPoints, v.Auction.StartPrice)
		}
		bots = append(bots, Player{
			MatchID: m.ID,
			Seat:    player.Seat,
			Persona: personas[len(bots)],
			Seed:    rand.Int64(),
			Carry:   carry,
		})
	}
	if len(bots) > 0 {
		if err := tx.Create(&bots).Error; err != nil {
			return nil, err
		}
	}

	match.AfterCommit(tx, func(ctx context.Context) { pushViews(ctx, m.ID) })
	rule := lots[0].Rule(&v.Auction)
	return biddingPhase(&rule, 1), nil
}

// newLots 生成拍场的全部拍品和藏品: 抽出拍品数, 每个拍品按权重抽一种档次生成仓库, 公示概率按格子大小推断(见 publicLots),
// 金额取档次按格子大小推导的; 每个拍品按首轮扫描比例抽出第 1 轮扫出轮廓的藏品(见 previewPieces), 按位置排出扫描顺序并快照扫描时长
func (vm *venueMode) newLots(tx *gorm.DB, matchID uint64) ([]Lot, []WarehouseItem, error) {
	v := vm.venue
	valueFactor := int32(venue.BasisPoints)
	if vm.kind == venue.ModePve {
		valueFactor = v.Pve.ValueFactor
	}
	lineup, err := venue.NewLineup(tx, v)
	if err != nil {
		return nil, nil, fmt.Errorf("auction: load warehouse rules of venue %s failed: %w", v.Code, err)
	}
	tiers := v.Lots.Tiers
	public := publicLots(tiers, venueLineup{lineup}, valueFactor)
	scanDurations, err := item.ScanDurations(tx)
	if err != nil {
		return nil, nil, err
	}

	count := lineup.Count()
	lots := make([]Lot, count)
	var rows []WarehouseItem
	for i := range lots {
		k := lineup.Pick()
		warehouse, err := lineup.Tier(k).Generate(valueFactor)
		if err != nil {
			return nil, nil, fmt.Errorf("auction: generate warehouse of venue %s failed: %w", v.Code, err)
		}
		t := &tiers[k].Auction
		lots[i] = Lot{
			MatchID:    matchID,
			LotNo:      int32(i + 1),
			Tier:       tiers[k].Code,
			Grid:       tiers[k].Warehouse.Grid,
			Odds:       public[k].odds,
			StartPrice: t.StartPrice,
			SpeedFloor: t.SpeedFloor,
			Value:      warehouse.Value,
			PieceCount: int32(len(warehouse.Pieces)),
			Contenders: []int16{},
		}
		if i == 0 {
			lots[i].FirstRound = 1
		}

		itemIDs := make([]uint64, len(warehouse.Pieces))
		for j, piece := range warehouse.Pieces {
			itemIDs[j] = piece.ItemID
		}
		items, err := item.Resolve(tx, itemIDs)
		if err != nil {
			return nil, nil, err
		}
		preview := previewPieces(len(warehouse.Pieces), v.Auction.PreviewRate, rand.Int64N)
		scanOrder := scanOrders(warehouse.Pieces)
		for j, piece := range warehouse.Pieces {
			target := items[piece.ItemID]
			rows = append(rows, WarehouseItem{
				MatchID:      matchID,
				LotNo:        int32(i + 1),
				Preview:      preview[j],
				ScanOrder:    scanOrder[j],
				ScanMs:       scanDurations[piece.Quality],
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
			})
		}
	}
	return lots, rows, nil
}

// OnTimeout 出价阶段到期时没提交的真人视为不加价, 判定本轮; 开箱阶段到期时拍场已结算则对局结束, 否则开拍下一个拍品
func (vm *venueMode) OnTimeout(tx *gorm.DB, m *match.Match, _ []match.Player) (*match.Phase, error) {
	st, err := loadState(tx, m)
	if err != nil {
		return nil, err
	}
	switch m.Phase {
	case PhaseBidding:
		return st.resolveRound(tx, true)
	case PhaseReveal:
		if st.auction.Outcome != "" {
			return &match.Phase{Finish: true}, nil
		}
		return st.startLot(tx)
	}
	return nil, fmt.Errorf("auction: unknown phase %s of match %d", m.Phase, m.ID)
}

// OnCancel 拍场还没结算时取消: 已结算的拍品不变; 还没有拍品成交时退还正式局真人的入场代价;
// 退还还没结算的拍品上用掉的道具, 解冻剩余的带入资金。拍场已结算(最后的开箱阶段)时什么都不做
func (vm *venueMode) OnCancel(tx *gorm.DB, m *match.Match, _ []match.Player) error {
	st, err := loadState(tx, m)
	if err != nil {
		return err
	}
	a := &st.auction
	if a.Outcome != "" {
		return nil
	}
	uses, err := loadUses(tx, m.ID)
	if err != nil {
		return err
	}
	open := slices.DeleteFunc(slices.Clone(uses), func(u Use) bool { return st.lots[u.LotNo-1].Outcome != "" })
	sold := st.soldLots()
	for i := range st.players {
		p := &st.players[i]
		p.Profit = st.wonProfit(p.Seat)
		if err := tx.Model(p).Select("profit").Updates(p).Error; err != nil {
			return err
		}
		if p.UserID == 0 || p.IsPractice {
			continue
		}
		if sold == 0 {
			if err := st.refund(tx, p); err != nil {
				return err
			}
		}
		if err := st.refundUses(tx, p, open); err != nil {
			return err
		}
		if err := st.release(tx, p, p.Frozen-p.Spent); err != nil {
			return err
		}
	}
	now := time.Now()
	if lot := st.lot; lot.Outcome == "" {
		lot.Outcome, lot.SettledAt, lot.Contenders = OutcomeCancelled, &now, []int16{}
		if err := tx.Model(lot).Select("outcome", "settled_at", "contenders").Updates(lot).Error; err != nil {
			return err
		}
	}
	a.Outcome, a.SettledAt = OutcomeCancelled, &now
	return tx.Model(a).Select("outcome", "settled_at").Updates(a).Error
}

// biddingPhase 第 round 轮出价, round 为拍场内的轮次
func biddingPhase(rule *venue.AuctionRule, round int32) *match.Phase {
	return &match.Phase{Name: PhaseBidding, Round: round, Duration: time.Duration(rule.RoundSeconds) * time.Second}
}

// state 一局暗拍的当前状态, 在锁定对局的事务中读出后修改
type state struct {
	match     *match.Match
	auction   Match
	lots      []Lot          // 按序号排序
	lot       *Lot           // 当前拍品, lots 中的一个
	players   []Player       // 按座位号排序
	submitted map[int16]bool // 本轮已出价的座位
	hints     []HostHint     // 主持人的提示, 只有展示时读出, 见 loadDetails
	hypes     []HypeEvent    // 热度的变动, 同上
	bids      []Bid          // 全部出价记录, 同上
}

// loadState 读出对局的暗拍状态, 不是暗拍的对局返回 gorm.ErrRecordNotFound
func loadState(tx *gorm.DB, m *match.Match) (*state, error) {
	st := &state{match: m, submitted: map[int16]bool{}}
	if err := tx.Where("match_id = ?", m.ID).First(&st.auction).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("match_id = ?", m.ID).Order("lot_no").Find(&st.lots).Error; err != nil {
		return nil, err
	}
	if int(st.auction.Lot) > len(st.lots) || st.auction.Lot < 1 {
		return nil, fmt.Errorf("auction: lot %d of match %d not found", st.auction.Lot, m.ID)
	}
	st.lot = &st.lots[st.auction.Lot-1]
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

// rule 当前拍品的拍卖参数
func (st *state) rule() venue.AuctionRule {
	return st.lot.Rule(&st.auction.Venue.Auction)
}

// canBid p 在本轮是否可以出价: 加轮时只有并列最高的人可以, 剩余资金不够当前拍品的起拍价时不能出价
func (st *state) canBid(p *Player) bool {
	return st.lot.CanBid(p.Seat) && p.Remaining() >= st.lot.StartPrice
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
	for i := range st.players {
		p := &st.players[i]
		if p.UserID != 0 && st.canBid(p) && !st.submitted[p.Seat] {
			return false
		}
	}
	return true
}

// humanCanBid 本轮是否有真人可以出价
func (st *state) humanCanBid() bool {
	for i := range st.players {
		if p := &st.players[i]; p.UserID != 0 && st.canBid(p) {
			return true
		}
	}
	return false
}

// placeBid 记录 p 本轮的出价, 高于当前出价时更新当前出价和提交时间
func (st *state) placeBid(tx *gorm.DB, p *Player, amount int64, at time.Time) error {
	if err := tx.Create(&Bid{MatchID: st.match.ID, LotNo: st.lot.LotNo, Round: st.match.Round, Seat: p.Seat, Amount: amount}).Error; err != nil {
		return err
	}
	st.submitted[p.Seat] = true
	if amount <= p.Bid {
		return nil
	}
	p.Bid, p.BidAt = amount, &at
	return tx.Model(p).Select("bid", "bid_at").Updates(p).Error
}

// botBids 本轮可以出价的人机按风格决策并出价, 同时更新估值
//
// 人机看到的与真人在这一轮出价时相同, 见 Observation; 决策出错时记录日志并视为不加价, 不影响对局。
// waited 为本轮是否等待过真人出价: 是时出价时间见 botBidAt; 否则没有真人可比, 为现在。
func (st *state) botBids(tx *gorm.DB, waited bool) error {
	m, lot := st.match, st.lot
	rule := st.rule()
	now := time.Now()
	var start time.Time
	var length time.Duration
	if waited && m.PhaseEndsAt != nil {
		length = time.Duration(rule.RoundSeconds) * time.Second
		start = m.PhaseEndsAt.Add(-length)
	}
	var future int64
	for _, l := range st.lots[lot.LotNo:] {
		future += expectedValue(l.Odds)
	}
	var prints []Footprint
	var shown map[int16]int64
	for i := range st.players {
		p := &st.players[i]
		if p.UserID != 0 || !st.canBid(p) {
			continue
		}
		if shown == nil {
			var err error
			if shown, err = shownBids(tx, lot, m.Round, rule.Rounds); err != nil {
				return err
			}
			items, err := loadItems(tx, m.ID, lot.LotNo)
			if err != nil {
				return err
			}
			prints = footprints(items)
		}
		others := make([]int64, 0, len(st.players)-1)
		for _, other := range st.players {
			if other.Seat != p.Seat {
				others = append(others, shown[other.Seat])
			}
		}
		obs := Observation{
			Round:      lot.Round(m.Round),
			Rules:      rule,
			Grid:       lot.Grid,
			Footprints: prints,
			Odds:       lot.Odds,
			Carry:      p.Remaining(),
			Bid:        p.Bid,
			Others:     others,
			Future:     future,
			LotsLeft:   st.auction.LotCount - lot.LotNo,
			Seed:       p.Seed,
		}
		var amount int64
		decision, err := bot.Decide[Observation, Decision](&p.Persona, obs, rand.New(rand.NewPCG(uint64(p.Seed), uint64(m.Round))))
		if err != nil {
			logger.ErrorF(tx.Statement.Context, "[Auction] bot of seat %d in match %d failed to decide: %v", p.Seat, m.ID, err)
		} else {
			if amount = min(decision.Bid, p.Remaining()); amount < rule.StartPrice || amount <= p.Bid {
				amount = 0
			}
			if decision.Estimate != p.Estimate {
				p.Estimate = decision.Estimate
				if err := tx.Model(p).Select("estimate").Updates(p).Error; err != nil {
					return err
				}
			}
		}
		if err := st.placeBid(tx, p, amount, botBidAt(start, length, now, rand.Int64N)); err != nil {
			return err
		}
	}
	return nil
}

// shownBids 拍场的第 round 轮出价时已公开的拍品 lot 上每个座位的出价: 拍品的前 N-1 轮(N 为规则的轮数)结束后公开,
// 取 round 之前的公开轮次中每个座位的最高出价
func shownBids(tx *gorm.DB, lot *Lot, round, rounds int32) (map[int16]int64, error) {
	var rows []Bid
	if err := tx.Where("match_id = ? AND lot_no = ? AND round < ? AND round < ?", lot.MatchID, lot.LotNo, round, lot.FirstRound+rounds-1).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	shown := map[int16]int64{}
	for _, b := range rows {
		shown[b.Seat] = max(shown[b.Seat], b.Amount)
	}
	return shown, nil
}

// resolveRound 人机出价后判定本轮: 当前拍品成交或流拍时结算并进入开箱, 否则进入下一轮
//
// 开箱阶段的时长为这个拍品的扫描时长之和加上停留时长, 还有下一个拍品时再加上拍品间隔; 最后一个拍品开箱时拍场已结算, 对局随即释放,
// 玩家可以离开去排下一局。没有真人可以出价的轮次(只有人机并列的加轮、真人的剩余资金都不够起拍价)不等待超时, 直接判定下一轮。
// waited 见 botBids。
func (st *state) resolveRound(tx *gorm.DB, waited bool) (*match.Phase, error) {
	a, lot := &st.auction, st.lot
	rule := st.rule()
	match.AfterCommit(tx, func(ctx context.Context) { pushViews(ctx, st.match.ID) })
	for ; ; waited = false {
		round := st.match.Round
		if err := st.botBids(tx, waited); err != nil {
			return nil, err
		}

		result := judge(&rule, lot.Round(round), lot.ExtraRounds, st.bidders())
		if result.Outcome != "" {
			if err := st.settleLot(tx, result); err != nil {
				return nil, err
			}
			scan, err := scanDuration(tx, st.match.ID, lot.LotNo)
			if err != nil {
				return nil, err
			}
			phase := &match.Phase{Name: PhaseReveal, Round: round, Duration: scan + time.Duration(rule.RevealSeconds)*time.Second}
			if lot.LotNo < a.LotCount {
				phase.Duration += time.Duration(a.Venue.GapSeconds) * time.Second
				return phase, nil
			}
			if err := st.settleSession(tx); err != nil {
				return nil, err
			}
			phase.Release = true
			return phase, nil
		}

		if lot.Round(round) >= rule.Rounds {
			lot.ExtraRounds++
		}
		lot.Contenders = result.Contenders
		if lot.Contenders == nil {
			lot.Contenders = []int16{}
		}
		if err := tx.Model(lot).Select("contenders", "extra_rounds").Updates(lot).Error; err != nil {
			return nil, err
		}
		if err := st.advanceHost(tx, round); err != nil {
			return nil, err
		}
		// 出价记录按 match.Round 写入, 阶段最终由 Advance 按返回值写入
		st.match.Round = round + 1
		clear(st.submitted)
		if st.humanCanBid() {
			return biddingPhase(&rule, round+1), nil
		}
	}
}

// startLot 开拍下一个拍品: 从上一个拍品的最后一轮之后开始, 清空每个人的出价和估值, 热度回落并给出第 1 轮的提示
func (st *state) startLot(tx *gorm.DB) (*match.Phase, error) {
	a := &st.auction
	a.Lot++
	if int(a.Lot) > len(st.lots) {
		return nil, fmt.Errorf("auction: lot %d of match %d not found", a.Lot, a.MatchID)
	}
	st.lot = &st.lots[a.Lot-1]
	round := st.match.Round + 1
	st.lot.FirstRound = round
	if err := tx.Model(st.lot).Select("first_round").Updates(st.lot).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(a).Select("lot").Updates(a).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&Player{}).Where("match_id = ?", a.MatchID).
		Updates(map[string]any{"bid": 0, "bid_at": nil, "estimate": 0}).Error; err != nil {
		return nil, err
	}
	for i := range st.players {
		st.players[i].Bid, st.players[i].BidAt, st.players[i].Estimate = 0, nil, 0
	}
	if err := st.lotHost(tx); err != nil {
		return nil, err
	}

	st.match.Round = round
	clear(st.submitted)
	rule := st.rule()
	if !st.humanCanBid() {
		return st.resolveRound(tx, false)
	}
	match.AfterCommit(tx, func(ctx context.Context) { pushViews(ctx, st.match.ID) })
	return biddingPhase(&rule, round), nil
}

// scanDuration 开箱时扫描一个拍品的全部藏品的时长
func scanDuration(tx *gorm.DB, matchID uint64, lotNo int32) (time.Duration, error) {
	var total int64
	if err := tx.Model(&WarehouseItem{}).Where("match_id = ? AND lot_no = ?", matchID, lotNo).
		Select("COALESCE(SUM(scan_ms), 0)").Scan(&total).Error; err != nil {
		return 0, err
	}
	return time.Duration(total) * time.Millisecond, nil
}

// lotAction 拍品上资金变动的幂等键中的动作, 如 settle_2
func lotAction(action string, lotNo int32) string {
	return fmt.Sprintf("%s_%d", action, lotNo)
}

// settleLot 结算当前拍品: 成交时从成交者冻结的带入资金中扣除成交价并发放藏品, 计入已花的资金; 练习局和人机只计入已花的资金
func (st *state) settleLot(tx *gorm.DB, result verdict) error {
	lot := st.lot
	now := time.Now()
	if result.Outcome == OutcomeSold {
		p := st.player(result.Winner)
		p.Spent += result.Price
		if err := tx.Model(p).Select("spent").Updates(p).Error; err != nil {
			return err
		}
		if p.UserID != 0 && !p.IsPractice {
			if err := st.changeWallet(tx, p, 0, -result.Price, wallet.BizMatchBidSettle, lotAction("settle", lot.LotNo)); err != nil {
				return err
			}
			if err := st.grantWarehouse(tx, p, lot.LotNo); err != nil {
				return err
			}
		}
	}
	lot.Outcome, lot.WinnerSeat, lot.Price, lot.SettledAt, lot.Contenders = result.Outcome, result.Winner, result.Price, &now, []int16{}
	return tx.Model(lot).Select("outcome", "winner_seat", "price", "settled_at", "contenders").Updates(lot).Error
}

// soldLots 已成交的拍品数
func (st *state) soldLots() int32 {
	var n int32
	for _, l := range st.lots {
		if l.Outcome == OutcomeSold {
			n++
		}
	}
	return n
}

// wonProfit 座位 seat 拿下的拍品的价值 - 成交价之和
func (st *state) wonProfit(seat int16) int64 {
	var total int64
	for _, l := range st.lots {
		if l.Outcome == OutcomeSold && l.WinnerSeat == seat {
			total += l.Value - l.Price
		}
	}
	return total
}

// settleSession 最后一个拍品结算后结算拍场: 写入盈亏和名次, 全部流拍时退还入场费和入场代价, 解冻剩余的带入资金, 写入统计;
// 排位至少成交一个拍品时结算段位分
//
// 练习局只计算盈亏和名次, 不变动资金也不写统计; 人机同。排位中挂机的真人计入统计, 提交后记录挂机次数。
func (st *state) settleSession(tx *gorm.DB) error {
	a := &st.auction
	now := time.Now()
	sold := st.soldLots()
	afk, err := st.afkUsers(tx)
	if err != nil {
		return err
	}
	var finals []struct {
		LotNo  int32
		Seat   int16
		Amount int64
	}
	if err := tx.Model(&Bid{}).Where("match_id = ?", st.match.ID).
		Select("lot_no, seat, MAX(amount) AS amount").Group("lot_no, seat").Scan(&finals).Error; err != nil {
		return err
	}

	totals := make([]seatTotal, len(st.players))
	for i := range st.players {
		p := &st.players[i]
		// 入场费和入场代价在准入时已扣除, 不计入盈亏
		p.Profit = st.wonProfit(p.Seat)
		totals[i] = seatTotal{Seat: p.Seat, Profit: p.Profit, Gap: -1}
		var gap float64
		var n int
		for _, f := range finals {
			if value := st.lots[f.LotNo-1].Value; f.Seat == p.Seat && f.Amount > 0 && value > 0 {
				gap += math.Abs(float64(f.Amount-value)) / float64(value)
				n++
			}
		}
		if n > 0 {
			totals[i].Gap = gap / float64(n)
		}
	}
	var ranks map[int16]int16
	if sold > 0 {
		ranks = rankSession(totals)
	}

	for i := range st.players {
		p := &st.players[i]
		p.Rank = ranks[p.Seat]
		if err := tx.Model(p).Select("profit", "rank").Updates(p).Error; err != nil {
			return err
		}
		if p.UserID == 0 || p.IsPractice {
			continue
		}
		if sold == 0 {
			if err := st.refund(tx, p); err != nil {
				return err
			}
		}
		if err := st.release(tx, p, p.Frozen-p.Spent); err != nil {
			return err
		}

		values := stats.Values{stats.MetricMatches: 1, stats.MetricProfit(a.Venue.BidCurrency): p.Profit}
		if p.Rank == 1 {
			values[stats.MetricWins] = 1
		}
		var won int64
		for _, l := range st.lots {
			if l.Outcome == OutcomeSold && l.WinnerSeat == p.Seat {
				won++
			}
		}
		if won > 0 {
			values[MetricBidWon] = won
		}
		if slices.Contains(afk, p.UserID) {
			values[ranked.MetricAfk] = 1
		}
		if err := stats.Add(tx, p.UserID, stats.MatchScopes(a.Venue.Code, st.kind()), now, values); err != nil {
			return err
		}
	}
	if sold > 0 && st.kind() == venue.ModeRanked {
		if err := st.settleRatings(tx, ranks, now); err != nil {
			return err
		}
	}
	if len(afk) > 0 {
		matchID := st.match.ID
		match.AfterCommit(tx, func(ctx context.Context) { recordAfk(ctx, matchID, afk) })
	}
	if err := match.SetRanks(tx, st.match.ID, ranks); err != nil {
		return err
	}

	a.Outcome, a.SettledAt = OutcomeUnsold, &now
	if sold > 0 {
		a.Outcome = OutcomeSold
	}
	return tx.Model(a).Select("outcome", "settled_at").Updates(a).Error
}

// afkUsers 排位中整个拍场都没有提交出价的真人, 练习局和其他模式不算
//
// 每轮都可以提交"不加价", 一次都没有提交说明人不在; 轮次到期时没提交的人没有出价记录。
func (st *state) afkUsers(tx *gorm.DB) ([]uint64, error) {
	if st.kind() != venue.ModeRanked {
		return nil, nil
	}
	var seats []int16
	if err := tx.Model(&Bid{}).Where("match_id = ?", st.match.ID).Distinct("seat").Pluck("seat", &seats).Error; err != nil {
		return nil, err
	}
	var afk []uint64
	for _, p := range st.players {
		if p.UserID != 0 && !p.IsPractice && !slices.Contains(seats, p.Seat) {
			afk = append(afk, p.UserID)
		}
	}
	return afk, nil
}

// recordAfk 记录挂机次数, 达到次数时禁赛; 失败只记录日志, 只是少算一次
func recordAfk(ctx context.Context, matchID uint64, userIDs []uint64) {
	for _, userID := range userIDs {
		ban, err := ranked.RecordAfk(ctx, userID, matchID)
		if err != nil {
			logger.ErrorF(ctx, "[Auction] record afk of user %d in match %d failed: %v", userID, matchID, err)
			continue
		}
		if ban > 0 {
			logger.InfoF(ctx, "[Auction] user %d is banned from ranked for %s after afk in match %d", userID, ban, matchID)
		}
	}
}

// settleRatings 按名次结算真人的段位分, 练习局不计分
func (st *state) settleRatings(tx *gorm.DB, ranks map[int16]int16, now time.Time) error {
	var results []ranked.Result
	for _, p := range st.players {
		if p.UserID != 0 && !p.IsPractice {
			results = append(results, ranked.Result{UserID: p.UserID, Rank: ranks[p.Seat]})
		}
	}
	changes, err := ranked.Settle(tx, now, results)
	if err != nil {
		return err
	}
	for i := range st.players {
		p := &st.players[i]
		change, ok := changes[p.UserID]
		if p.UserID == 0 || !ok {
			continue
		}
		p.RatingDelta, p.Rating = change.Delta, change.Rating
		if err := tx.Model(p).Select("rating_delta", "rating").Updates(p).Error; err != nil {
			return err
		}
	}
	return nil
}

// grantWarehouse 把拍品 lotNo 里的藏品发给成交者, 已下架或已删除的藏品按开局时的回收价折成货币
func (st *state) grantWarehouse(tx *gorm.DB, p *Player, lotNo int32) error {
	items, err := loadItems(tx, st.match.ID, lotNo)
	if err != nil {
		return err
	}
	var bundle reward.Bundle
	prices := make(map[uint64]reward.CurrencyLine, len(items))
	for _, it := range items {
		bundle.Items = append(bundle.Items, reward.ItemLine{ItemID: it.ItemID, Quantity: 1})
		prices[it.ItemID] = reward.CurrencyLine{Currency: it.SellCurrency, Amount: it.SellPrice}
	}
	bundle, err = reward.Deliverable(tx, bundle, prices)
	if err != nil {
		return err
	}
	return st.applyReward(tx, p, bundle, wallet.BizMatchReward, lotAction("reward", lotNo))
}

// refund 退还 p 的入场代价, 已下架或已删除的物品按当前回收价折成货币
func (st *state) refund(tx *gorm.DB, p *Player) error {
	if p.EntryCost.IsEmpty() {
		return nil
	}
	bundle, err := reward.Deliverable(tx, p.EntryCost, nil)
	if err != nil {
		return err
	}
	return st.applyReward(tx, p, bundle, wallet.BizMatchRefund, "refund")
}

// refundUses 退还 p 在 uses 中用掉的道具, 已下架或已删除的按当前回收价折成货币
func (st *state) refundUses(tx *gorm.DB, p *Player, uses []Use) error {
	items := map[uint64]int64{}
	for _, use := range uses {
		if use.Seat == p.Seat {
			items[use.ItemID]++
		}
	}
	if len(items) == 0 {
		return nil
	}
	result, replayed, err := effect.Refund(tx, &effect.RefundRequest{
		UserID:         p.UserID,
		Items:          items,
		BizID:          strconv.FormatUint(st.match.ID, 10),
		IdempotencyKey: matchKey(st.match.ID, "use_refund", p.UserID),
		Remark:         remarkOf(st.auction.Venue.Name, st.kind()),
	})
	if err != nil {
		return err
	}
	pushResult(tx, result, replayed)
	return nil
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
		Remark:         remarkOf(st.auction.Venue.Name, st.kind()),
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
		Remark:         remarkOf(st.auction.Venue.Name, st.kind()),
	})
	if err != nil {
		return err
	}
	pushResult(tx, result, replayed)
	return nil
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

// kind 这局的模式, venue.ModePve 或 venue.ModeRanked, 即玩法代码中冒号之前的部分
func (st *state) kind() string {
	kind, _, _ := strings.Cut(st.match.Mode, ":")
	return kind
}

// remarkOf 资金流水的备注, 如 "码头仓库·人机"
func remarkOf(venueName, kind string) string {
	return venueName + "·" + venue.ModeName(kind)
}

// bidError 出价或使用道具被拒绝, 原因展示给玩家
type bidError struct {
	reason string
}

func (e *bidError) Error() string { return e.reason }

// rejectBid 拒绝出价或使用道具
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
	Amount  int64  `json:"amount"` // 0 表示不加价; 否则不低于当前拍品的起拍价和自己之前的出价, 不超过剩余资金
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
		lot := st.lot
		switch {
		case !lot.CanBid(seat):
			return nil, rejectBid(NotContender)
		case p.Remaining() < lot.StartPrice:
			return nil, rejectBid(RemainingShort, lot.StartPrice)
		case st.submitted[seat]:
			return nil, rejectBid(AlreadyBid)
		case req.Amount == 0:
		case req.Amount < lot.StartPrice:
			return nil, rejectBid(BidTooLow, lot.StartPrice)
		case req.Amount < p.Bid:
			return nil, rejectBid(BidBelowPrevious, p.Bid)
		case req.Amount > p.Remaining():
			return nil, rejectBid(BidOverCarry, p.Remaining())
		}
		if err := st.placeBid(tx, p, req.Amount, time.Now()); err != nil {
			return nil, err
		}
		if !st.allSubmitted() {
			return nil, nil
		}
		return st.resolveRound(tx, true)
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

// UseRequest 使用道具
type UseRequest struct {
	MatchID uint64 `json:"match_id"`
	Round   int32  `json:"round"`   // 使用时的轮次, 与当前轮次不一致时拒绝
	ItemID  uint64 `json:"item_id"` // 效果属于对局场景的物品
}

// UseItem 真人在出价阶段对当前拍品使用一个道具: 扣除一个, 按效果算出情报并保存, 返回使用后自己的视角; 不影响出价和本轮的判定
//
// 本轮已出过价也可以用, 情报留到这个拍品之后几轮; 次数上限按整个拍场计。不满足使用规则时返回 *bidError, 原因可以直接展示。
func UseItem(ctx context.Context, userID uint64, req *UseRequest) (*AuctionView, error) {
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
		rule := &a.Venue.Consumables
		limit := rule.MaxPerMatch(st.kind())
		switch {
		case p.IsPractice:
			return nil, rejectBid(UsePractice)
		case !st.lot.CanBid(seat):
			return nil, rejectBid(NotContender)
		case limit == 0:
			return nil, rejectBid(UseDisabled)
		case !rule.Allows(req.ItemID):
			return nil, rejectBid(UseNotAllowed)
		}
		var target item.Item
		if err := tx.Where("id = ?", req.ItemID).Limit(1).Find(&target).Error; err != nil {
			return nil, err
		}
		if target.ID == 0 || effect.SceneOf(target.Effect) != effect.SceneMatch {
			return nil, rejectBid(UseUnusable)
		}
		var uses []Use
		if err := tx.Where("match_id = ? AND seat = ?", m.ID, seat).Order("id").Find(&uses).Error; err != nil {
			return nil, err
		}
		if int32(len(uses)) >= limit {
			return nil, rejectBid(UseLimitReached, limit)
		}
		for _, use := range uses {
			if use.ItemID == req.ItemID && use.Round == m.Round {
				return nil, rejectBid(UseRepeated)
			}
		}

		result, replayed, err := effect.Consume(tx, &effect.UseRequest{
			UserID:         userID,
			ItemID:         req.ItemID,
			BizID:          strconv.FormatUint(m.ID, 10),
			IdempotencyKey: fmt.Sprintf("%s:%d", matchKey(m.ID, "use", userID), len(uses)+1),
			Remark:         remarkOf(a.Venue.Name, st.kind()),
			Scopes:         stats.MatchScopes(a.Venue.Code, st.kind()),
		})
		if err != nil {
			if err.Error() == inventory.InsufficientQuantity {
				return nil, rejectBid(UseQuantityShort)
			}
			return nil, err
		}
		pushResult(tx, result, replayed)

		items, err := loadItems(tx, m.ID, st.lot.LotNo)
		if err != nil {
			return nil, err
		}
		useCtx := newUseContext(st.lot.Value, items, lotUses(uses, st.lot.LotNo), rand.Int64N)
		intel, err := effect.Apply[*UseContext, Intel](effect.SceneMatch, target.Effect, useCtx, target.EffectParams)
		if err != nil {
			return nil, err
		}
		return nil, tx.Create(&Use{
			MatchID: m.ID,
			LotNo:   st.lot.LotNo,
			Seat:    seat,
			ItemID:  target.ID,
			Round:   m.Round,
			Name:    target.Name,
			Effect:  target.Effect,
			Params:  target.EffectParams,
			Result:  intel,
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

// loadUses 一局中用掉的全部道具, 按使用顺序
func loadUses(tx *gorm.DB, matchID uint64) ([]Use, error) {
	var uses []Use
	if err := tx.Where("match_id = ?", matchID).Order("id").Find(&uses).Error; err != nil {
		return nil, err
	}
	return uses, nil
}

// UsableItem 背包中这局可以用的一种道具
type UsableItem struct {
	ItemID      uint64 `json:"item_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Quality     int16  `json:"quality"`
	ImageURL    string `json:"image_url"`
	Effect      string `json:"effect"`
	Quantity    int64  `json:"quantity"`
}

// QueryUsableItems 用户背包中这局可以用的道具: 效果属于对局场景且地图的道具规则允许, 按品质从高到低
//
// 只看地图规则和背包, 不检查阶段、次数和练习局; 这个模式不能用道具时为空。
// 对局不存在、不是暗拍或用户不是该局的真人玩家时返回 match.MatchNotFound。
func QueryUsableItems(ctx context.Context, userID, matchID uint64) ([]UsableItem, error) {
	tx := db.DB(ctx)
	var m match.Match
	if err := m.Exact(tx, matchID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(match.MatchNotFound)
		}
		return nil, err
	}
	var a Match
	if err := tx.Where("match_id = ?", matchID).Limit(1).Find(&a).Error; err != nil {
		return nil, err
	}
	var joined int64
	if err := tx.Model(&Player{}).Where("match_id = ? AND user_id = ?", matchID, userID).Count(&joined).Error; err != nil {
		return nil, err
	}
	if a.MatchID == 0 || userID == 0 || joined == 0 {
		return nil, errors.New(match.MatchNotFound)
	}

	usable := []UsableItem{}
	kind, _, _ := strings.Cut(m.Mode, ":")
	rule := &a.Venue.Consumables
	if rule.MaxPerMatch(kind) == 0 {
		return usable, nil
	}
	var rows []UsableItem
	if err := tx.Model(&inventory.UserItem{}).
		Select("items.id AS item_id, items.name, items.description, items.quality, items.image_url, items.effect, inventory_items.quantity").
		Joins("JOIN items ON items.id = inventory_items.item_id AND items.deleted_at IS NULL").
		Where("inventory_items.user_id = ? AND inventory_items.quantity > 0 AND items.effect <> ''", userID).
		Order("items.quality DESC, items.id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if effect.SceneOf(row.Effect) == effect.SceneMatch && rule.Allows(row.ItemID) {
			usable = append(usable, row)
		}
	}
	return usable, nil
}

// QueryCarry 用户在各地图选的带入金额, 按地图代码
func QueryCarry(ctx context.Context, userID uint64) ([]CarryPreference, error) {
	prefs := []CarryPreference{}
	if err := db.DB(ctx).Where("user_id = ?", userID).Order("venue").Find(&prefs).Error; err != nil {
		return nil, err
	}
	return prefs, nil
}
