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
	"slices"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
)

// VenueInfo 对局中展示的地图参数, 来自开局时的快照; 每个拍品的起拍价等见 LotView
type VenueInfo struct {
	Code           string   `json:"code"`
	Name           string   `json:"name"`
	BidCurrency    string   `json:"bid_currency"`
	Rounds         int32    `json:"rounds"`    // 每个拍品的轮数
	QuickWin       []int32  `json:"quick_win"` // 前 Rounds-1 轮的速胜倍率, 基点
	MaxExtraRounds int32    `json:"max_extra_rounds"`
	EntryFee       int64    `json:"entry_fee"`     // 入场费等以下字段见 venue.AuctionRule
	MaxCarry       int64    `json:"max_carry"`     // 带入资金上限
	LeadDiscount   int32    `json:"lead_discount"` // 领先折让, 基点
	PreviewRate    int32    `json:"preview_rate"`  // 首轮扫描比例, 基点: 每个拍品的第 1 轮每件藏品按这个概率扫出轮廓, 至少一件
	MaxUses        int32    `json:"max_uses"`      // 这个模式每局(整个拍场)最多使用几个道具, 0 表示不能用
	AllowedItems   []uint64 `json:"allowed_items"` // 可以用的道具, 为空表示所有对局中可用的道具
	GapSeconds     int32    `json:"gap_seconds"`   // 拍品开箱后停留多久进入下一个
}

// Footprint 第 1 轮扫出轮廓的一件藏品占的格子, 不包含品质、名称、回收价和扫描顺序, 以位置(X, Y)区分
type Footprint struct {
	X      int16 `json:"x"`
	Y      int16 `json:"y"`
	Width  int16 `json:"width"`
	Height int16 `json:"height"`
}

// SelfView 自己在这局中的状态
type SelfView struct {
	Seat        int16         `json:"seat"`
	IsPractice  bool          `json:"is_practice"`
	EntryCost   reward.Bundle `json:"entry_cost"`   // 实际扣除的入场代价
	Carry       int64         `json:"carry"`        // 带入资金
	Spent       int64         `json:"spent"`        // 已拿下的拍品的成交价之和
	Remaining   int64         `json:"remaining"`    // 剩余资金, 当前拍品出价的上限
	Bid         int64         `json:"bid"`          // 当前拍品的出价, 0 表示还没出过价
	Submitted   bool          `json:"submitted"`    // 本轮是否已出价
	CanBid      bool          `json:"can_bid"`      // 现在是否可以出价; 剩余资金不够当前拍品的起拍价时不能
	Uses        []UseView     `json:"uses"`         // 自己用过的道具和得到的情报, 按使用顺序, 含之前的拍品
	UsesLeft    int32         `json:"uses_left"`    // 本局还能用几个道具, 练习局为 0; 不考虑阶段和轮次
	Profit      int64         `json:"profit"`       // 拍场结算后的盈亏, 出价货币, 不含准入时扣除的入场代价; 练习局按正式局计算
	Rank        int16         `json:"rank"`         // 拍场结算后的名次, 全部流拍时为 0
	RatingDelta int32         `json:"rating_delta"` // 段位分的变化, 只有排位且至少成交一个拍品时有
	Rating      int32         `json:"rating"`       // 结算后的段位分, 同上
}

// UseView 自己用过的一个道具
type UseView struct {
	LotNo     int32     `json:"lot_no"`
	Round     int32     `json:"round"`
	ItemID    uint64    `json:"item_id"`
	Name      string    `json:"name"`   // 使用时的物品名称
	Effect    string    `json:"effect"` // 效果代码, 前端按它展示 Result
	Result    Intel     `json:"result"`
	CreatedAt time.Time `json:"created_at"`
}

// UseCount 拍品开箱后公开的一个座位在这个拍品上用某种道具的个数
type UseCount struct {
	ItemID uint64 `json:"item_id"`
	Name   string `json:"name"`
	Count  int32  `json:"count"`
}

// SeatResult 拍品开箱后公开的一个座位的出价
type SeatResult struct {
	Seat int16      `json:"seat"`
	Bid  int64      `json:"bid"`  // 最后的出价, 0 表示没出过价
	Uses []UseCount `json:"uses"` // 在这个拍品上用过的道具, 按第一次使用的顺序, 不公开得到的情报
}

// PublicBid 公开的一次出价: 拍品的前 N-1 轮结束后公开, 最后一轮和加轮在拍品结算后公开
type PublicBid struct {
	Round  int32 `json:"round"` // 拍场内的轮次
	Seat   int16 `json:"seat"`
	Amount int64 `json:"amount"` // 0 表示不加价
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
	ScanOrder    int32  `json:"scan_order"` // 开箱时扫描的顺序, 从 1 开始
	ScanMs       int32  `json:"scan_ms"`    // 扫描这件的时长, 毫秒
}

// LotView 公开的一个拍品: 格子大小和金额开局时公开, 第 1 轮的轮廓开拍后公开, 价值、藏品和每个座位的出价开箱后公开, 档次拍场结束后公开
type LotView struct {
	LotNo      int32        `json:"lot_no"`
	Grid       venue.Grid   `json:"grid"`
	StartPrice int64        `json:"start_price"`
	SpeedFloor int64        `json:"speed_floor"` // 速胜下限
	FirstRound int32        `json:"first_round"` // 开拍时的轮次, 还没开拍时为 0
	Footprints []Footprint  `json:"footprints"`  // 第 1 轮扫出轮廓的藏品占的格子, 按位置从上到下、从左到右; 件数开箱前不公开
	Contenders []int16      `json:"contenders"`  // 加轮时可以出价的座位, 为空表示所有人
	Bids       []PublicBid  `json:"bids"`        // 已公开的出价, 按轮次和座位
	Outcome    string       `json:"outcome"`     // 结算前为空, 之后为 sold、unsold 或 cancelled
	WinnerSeat int16        `json:"winner_seat"` // 没有成交时为 0
	Price      int64        `json:"price"`
	Value      int64        `json:"value"`     // 仓库价值, 开箱后公开
	Tier       string       `json:"tier"`      // 档次名称, 拍场结束后公开
	Pieces     []PieceView  `json:"pieces"`    // 开箱后公开, 按扫描顺序
	Seats      []SeatResult `json:"seats"`     // 开箱后公开, 按座位号排序
	ScanFrom   *time.Time   `json:"scan_from"` // 开始扫描的时间, 即结算时间; 按已过去的时间和每件的扫描时长算出扫描进度
}

// SessionSeat 拍场结束后公开的一个座位的成绩
type SessionSeat struct {
	Seat   int16 `json:"seat"`
	Carry  int64 `json:"carry"`
	Spent  int64 `json:"spent"`
	Profit int64 `json:"profit"` // 拿下的拍品的价值 - 成交价
	Rank   int16 `json:"rank"`
}

// SessionView 拍场结束后公开的成绩
type SessionView struct {
	Outcome string        `json:"outcome"` // sold(至少成交一个拍品)、unsold 或 cancelled
	Seats   []SessionSeat `json:"seats"`   // 按座位号排序
}

// PublicView 一局暗拍中所有人都看得到的部分
type PublicView struct {
	MatchID     uint64       `json:"match_id"`
	Mode        string       `json:"mode"`
	Status      string       `json:"status"`
	Phase       string       `json:"phase"`
	Round       int32        `json:"round"`     // 拍场内的轮次, 从 1 开始连续编号
	LotRound    int32        `json:"lot_round"` // 当前拍品的第几轮
	PhaseEndsAt *time.Time   `json:"phase_ends_at"`
	Version     int64        `json:"version"` // 与对局的 version 一致, 客户端只采用 version 更大的推送
	Venue       VenueInfo    `json:"venue"`
	LotCount    int32        `json:"lot_count"`
	Lot         int32        `json:"lot"`     // 当前拍品的序号, 从 1 开始
	Lots        []LotView    `json:"lots"`    // 按序号
	Host        *HostView    `json:"host"`    // 主持人, 没有主持人提示时为空
	Session     *SessionView `json:"session"` // 拍场结束后才有
}

// AuctionView 玩家看到的一局暗拍
type AuctionView struct {
	PublicView
	Self SelfView `json:"self"`
}

// AdminView 管理员看到的一局暗拍: 公开部分和每个拍品、每个座位的暗拍数据
type AdminView struct {
	PublicView
	LotRows []Lot    `json:"lot_rows"` // 全部拍品, 含档次代码、价值和件数
	Players []Player `json:"players"`  // 按座位号排序
	Uses    []Use    `json:"uses"`     // 全部座位用过的道具和得到的情报, 按使用顺序
}

// view 以 viewerID 的视角查看, 不是这局的真人玩家时返回 nil
func (st *state) view(viewerID uint64, items []WarehouseItem, uses []Use) *AuctionView {
	var self *Player
	for i := range st.players {
		if viewerID != 0 && st.players[i].UserID == viewerID {
			self = &st.players[i]
		}
	}
	if self == nil {
		return nil
	}
	m := st.match
	submitted := st.submitted[self.Seat]
	view := &AuctionView{
		PublicView: st.publicView(items, uses),
		Self: SelfView{
			Seat:       self.Seat,
			IsPractice: self.IsPractice,
			EntryCost:  self.EntryCost,
			Carry:      self.Carry,
			Spent:      self.Spent,
			Remaining:  self.Remaining(),
			Bid:        self.Bid,
			Submitted:  submitted,
			CanBid:     m.Status == match.StatusPlaying && m.Phase == PhaseBidding && st.canBid(self) && !submitted,
			Uses:       []UseView{},
		},
	}
	for _, use := range uses {
		if use.Seat == self.Seat {
			view.Self.Uses = append(view.Self.Uses, UseView{LotNo: use.LotNo, Round: use.Round, ItemID: use.ItemID, Name: use.Name, Effect: use.Effect, Result: use.Result, CreatedAt: use.CreatedAt})
		}
	}
	if !self.IsPractice {
		view.Self.UsesLeft = max(view.Venue.MaxUses-int32(len(view.Self.Uses)), 0)
	}
	if st.auction.Outcome != "" {
		view.Self.Profit, view.Self.Rank, view.Self.RatingDelta, view.Self.Rating = self.Profit, self.Rank, self.RatingDelta, self.Rating
	}
	return view
}

// publicView 所有人都看得到的部分
func (st *state) publicView(items []WarehouseItem, uses []Use) PublicView {
	m, a := st.match, &st.auction
	rule := &a.Venue.Auction
	ended := a.Outcome != ""
	view := PublicView{
		MatchID:     m.ID,
		Mode:        m.Mode,
		Status:      m.Status,
		Phase:       m.Phase,
		Round:       m.Round,
		LotRound:    st.lot.Round(m.Round),
		PhaseEndsAt: m.PhaseEndsAt,
		Version:     m.Version,
		Venue: VenueInfo{
			Code:           a.Venue.Code,
			Name:           a.Venue.Name,
			BidCurrency:    a.Venue.BidCurrency,
			Rounds:         rule.Rounds,
			QuickWin:       rule.QuickWin,
			MaxExtraRounds: rule.MaxExtraRounds,
			EntryFee:       rule.EntryFee,
			MaxCarry:       rule.MaxCarry,
			LeadDiscount:   rule.LeadDiscount,
			PreviewRate:    rule.PreviewRate,
			MaxUses:        a.Venue.Consumables.MaxPerMatch(st.kind()),
			AllowedItems:   a.Venue.Consumables.Allowed,
			GapSeconds:     a.Venue.GapSeconds,
		},
		LotCount: a.LotCount,
		Lot:      a.Lot,
		Lots:     make([]LotView, len(st.lots)),
		Host:     hostView(a, st.hints, st.hypes, ended),
	}
	for i := range st.lots {
		view.Lots[i] = st.lotView(&st.lots[i], lotItems(items, st.lots[i].LotNo), lotUses(uses, st.lots[i].LotNo), ended)
	}
	if !ended {
		return view
	}
	view.Session = &SessionView{Outcome: a.Outcome, Seats: make([]SessionSeat, len(st.players))}
	for i, p := range st.players {
		view.Session.Seats[i] = SessionSeat{Seat: p.Seat, Carry: p.Carry, Spent: p.Spent, Profit: st.wonProfit(p.Seat), Rank: p.Rank}
	}
	return view
}

// lotView 一个拍品的公开部分; ended 为拍场是否已结束
func (st *state) lotView(lot *Lot, items []WarehouseItem, uses []Use, ended bool) LotView {
	view := LotView{
		LotNo:      lot.LotNo,
		Grid:       lot.Grid,
		StartPrice: lot.StartPrice,
		SpeedFloor: lot.SpeedFloor,
		FirstRound: lot.FirstRound,
		Footprints: []Footprint{},
		Contenders: lot.Contenders,
		Bids:       []PublicBid{},
		Outcome:    lot.Outcome,
		WinnerSeat: lot.WinnerSeat,
		Price:      lot.Price,
	}
	if lot.FirstRound == 0 {
		return view
	}
	view.Footprints = footprints(items)
	settled := lot.Outcome != ""
	// 前 N-1 轮结束后公开, 结算后全部公开
	shownBefore := min(st.match.Round, lot.FirstRound+st.auction.Venue.Auction.Rounds-1)
	for _, b := range st.bids {
		if b.LotNo == lot.LotNo && (settled || b.Round < shownBefore) {
			view.Bids = append(view.Bids, PublicBid{Round: b.Round, Seat: b.Seat, Amount: b.Amount})
		}
	}
	if ended {
		view.Tier = st.auction.Venue.TierNames[lot.Tier]
	}
	if !settled {
		return view
	}

	view.Value, view.ScanFrom = lot.Value, lot.SettledAt
	view.Pieces = make([]PieceView, len(items))
	for i, it := range items {
		view.Pieces[i] = PieceView{
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
			ScanOrder:    it.ScanOrder,
			ScanMs:       it.ScanMs,
		}
	}
	view.Seats = make([]SeatResult, len(st.players))
	for i, p := range st.players {
		view.Seats[i] = SeatResult{Seat: p.Seat, Uses: useCounts(uses, p.Seat)}
		for _, b := range st.bids {
			if b.LotNo == lot.LotNo && b.Seat == p.Seat {
				view.Seats[i].Bid = max(view.Seats[i].Bid, b.Amount)
			}
		}
	}
	return view
}

// footprints 第 1 轮扫出轮廓的藏品占的格子, 与 items 的顺序相同
func footprints(items []WarehouseItem) []Footprint {
	prints := []Footprint{}
	for _, it := range items {
		if it.Preview {
			prints = append(prints, Footprint{X: it.X, Y: it.Y, Width: it.Width, Height: it.Height})
		}
	}
	return prints
}

// useCounts 座位 seat 用过的各种道具的个数, 按第一次使用的顺序
func useCounts(uses []Use, seat int16) []UseCount {
	counts := []UseCount{}
	for _, use := range uses {
		if use.Seat != seat {
			continue
		}
		i := slices.IndexFunc(counts, func(c UseCount) bool { return c.ItemID == use.ItemID })
		if i < 0 {
			counts = append(counts, UseCount{ItemID: use.ItemID, Name: use.Name})
			i = len(counts) - 1
		}
		counts[i].Count++
	}
	return counts
}

// loadItems 拍品 lotNo 里的藏品, 按扫描顺序; lotNo 为 0 时为全部拍品的, 按序号和扫描顺序
func loadItems(tx *gorm.DB, matchID uint64, lotNo int32) ([]WarehouseItem, error) {
	query := tx.Where("match_id = ?", matchID)
	if lotNo > 0 {
		query = query.Where("lot_no = ?", lotNo)
	}
	var items []WarehouseItem
	if err := query.Order("lot_no, scan_order").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// lotItems items 中拍品 lotNo 的藏品
func lotItems(items []WarehouseItem, lotNo int32) []WarehouseItem {
	var out []WarehouseItem
	for _, it := range items {
		if it.LotNo == lotNo {
			out = append(out, it)
		}
	}
	return out
}

// lotUses uses 中用在拍品 lotNo 上的
func lotUses(uses []Use, lotNo int32) []Use {
	var out []Use
	for _, use := range uses {
		if use.LotNo == lotNo {
			out = append(out, use)
		}
	}
	return out
}

// loadDetails 读出展示用的全部藏品、用掉的道具、出价记录、提示和热度变动
func (st *state) loadDetails(tx *gorm.DB) ([]WarehouseItem, []Use, error) {
	items, err := loadItems(tx, st.match.ID, 0)
	if err != nil {
		return nil, nil, err
	}
	uses, err := loadUses(tx, st.match.ID)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Where("match_id = ?", st.match.ID).Order("round, seat").Find(&st.bids).Error; err != nil {
		return nil, nil, err
	}
	if st.hints, st.hypes, err = loadHost(tx, st.match.ID); err != nil {
		return nil, nil, err
	}
	return items, uses, nil
}

// queryState 读出一局暗拍的状态、藏品和用掉的道具, 对局不存在或不是暗拍时返回 match.MatchNotFound
func queryState(ctx context.Context, matchID uint64) (*state, []WarehouseItem, []Use, error) {
	tx := db.DB(ctx)
	var m match.Match
	if err := m.Exact(tx, matchID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, errors.New(match.MatchNotFound)
		}
		return nil, nil, nil, err
	}
	st, err := loadState(tx, &m)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil, errors.New(match.MatchNotFound)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	items, uses, err := st.loadDetails(tx)
	if err != nil {
		return nil, nil, nil, err
	}
	return st, items, uses, nil
}

// QueryAuction 获取用户参与的一局暗拍, 对局不存在、不是暗拍或用户不是该局的真人玩家时返回 match.MatchNotFound
func QueryAuction(ctx context.Context, userID, matchID uint64) (*AuctionView, error) {
	st, items, uses, err := queryState(ctx, matchID)
	if err != nil {
		return nil, err
	}
	view := st.view(userID, items, uses)
	if view == nil {
		return nil, errors.New(match.MatchNotFound)
	}
	return view, nil
}

// QueryAdminAuction 管理员查看一局暗拍, 对局不存在或不是暗拍时返回 match.MatchNotFound
func QueryAdminAuction(ctx context.Context, matchID uint64) (*AdminView, error) {
	st, items, uses, err := queryState(ctx, matchID)
	if err != nil {
		return nil, err
	}
	view := &AdminView{PublicView: st.publicView(items, uses), LotRows: st.lots, Players: st.players, Uses: uses}
	view.Host = hostView(&st.auction, st.hints, st.hypes, true)
	return view, nil
}
