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
	"slices"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
)

// Match 一局暗拍的对局级数据, 与 match.Match 一一对应, 开局时创建
type Match struct {
	MatchID     uint64     `gorm:"primaryKey;autoIncrement:false" json:"match_id"`
	VenueID     uint64     `gorm:"not null" json:"venue_id"`
	Venue       Snapshot   `gorm:"type:jsonb;serializer:json;not null" json:"venue"`                   // 开局时的地图配置, 之后修改地图不影响这一局
	Value       int64      `gorm:"not null" json:"value"`                                              // 仓库价值, 出价货币的回收价之和
	PieceCount  int32      `gorm:"not null" json:"piece_count"`                                        // 藏品件数, 开局时公开
	Contenders  []int16    `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"contenders"` // 加轮时可以出价的座位, 为空表示所有人
	ExtraRounds int32      `gorm:"not null;default:0" json:"extra_rounds"`                             // 已加的轮数
	Outcome     string     `gorm:"size:16;not null;default:''" json:"outcome"`                         // 结算前为空, 之后为 sold、unsold 或 cancelled
	WinnerSeat  int16      `gorm:"not null;default:0" json:"winner_seat"`                              // 成交者的座位, 没有成交时为 0
	Price       int64      `gorm:"not null;default:0" json:"price"`                                    // 成交价
	CreatedAt   time.Time  `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
	SettledAt   *time.Time `json:"settled_at"` // 结算或取消的时间
}

// TableName 自定义表名
func (Match) TableName() string { return "auction_matches" }

// CanBid 座位在本轮是否可以出价, 加轮时只有并列最高的人可以
func (a *Match) CanBid(seat int16) bool {
	return len(a.Contenders) == 0 || slices.Contains(a.Contenders, seat)
}

// Snapshot 开局时快照的地图配置
type Snapshot struct {
	Code        string            `json:"code"`
	Name        string            `json:"name"`
	BidCurrency string            `json:"bid_currency"`
	EntryCost   reward.Bundle     `json:"entry_cost"`
	Grid        venue.Grid        `json:"grid"`
	Auction     venue.AuctionRule `json:"auction"`
	Bots        venue.BotRule     `json:"bots"`
}

// Player 一局暗拍中的一个座位
//
// 真人在准入时创建, 此时还没有座位, 开局时写入座位号; 人机在开局时创建。
type Player struct {
	ID         uint64        `gorm:"primaryKey" json:"id"`
	MatchID    uint64        `gorm:"not null;index:idx_auction_players_match_id" json:"match_id"`
	Seat       int16         `gorm:"not null;default:0" json:"seat"`
	UserID     uint64        `gorm:"not null;default:0" json:"user_id"`                                  // 人机为 0
	IsPractice bool          `gorm:"not null;default:false" json:"is_practice"`                          // 练习局, 准入时决定, 之后不再变化
	EntryCost  reward.Bundle `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"entry_cost"` // 实际扣除的入场代价, 练习局和人机为空
	Carry      int64         `gorm:"not null;default:0" json:"carry"`                                    // 带入资金, 出价的上限
	Frozen     int64         `gorm:"not null;default:0" json:"frozen"`                                   // 冻结的带入资金, 练习局和人机为 0
	Estimate   int64         `gorm:"not null;default:0" json:"estimate"`                                 // 人机对仓库价值的估值, 真人为 0
	Bid        int64         `gorm:"not null;default:0" json:"bid"`                                      // 当前出价, 0 表示还没出过价
	BidAt      *time.Time    `json:"bid_at"`                                                             // 当前出价的提交时间, 加轮用完仍并列时先出价的人成交
	Feedback   []Feedback    `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"feedback"`   // 每轮结束后的反馈, 只给真人
	Profit     int64         `gorm:"not null;default:0" json:"profit"`                                   // 结算后的盈亏, 出价货币; 练习局按正式局计算
	CreatedAt  time.Time     `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time     `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Player) TableName() string { return "auction_players" }

// Feedback 一轮结束后给玩家的反馈, 不透露其他人的出价
type Feedback struct {
	Round int32  `json:"round"`
	Rank  int16  `json:"rank"` // 自己的出价在出过价的人中排第几, 并列算同一名, 没出过价为 0
	Hint  string `json:"hint"` // 见 constants.go 的 Hint
}

// Bid 每轮的一次出价, 真人提交时写入, 人机在本轮结束时写入
//
// 唯一索引 uniq_auction_bids_match_id_round_seat 保证每人每轮只出价一次。
type Bid struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	MatchID   uint64    `gorm:"not null;uniqueIndex:uniq_auction_bids_match_id_round_seat,priority:1" json:"match_id"`
	Round     int32     `gorm:"not null;uniqueIndex:uniq_auction_bids_match_id_round_seat,priority:2" json:"round"`
	Seat      int16     `gorm:"not null;uniqueIndex:uniq_auction_bids_match_id_round_seat,priority:3" json:"seat"`
	Amount    int64     `gorm:"not null;default:0" json:"amount"` // 0 表示不加价
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (Bid) TableName() string { return "auction_bids" }

// WarehouseItem 仓库里的一件藏品, 开局时快照名称和回收价
//
// 唯一索引 uniq_auction_warehouse_items_match_id_reveal_order 保证公开顺序不重复。
type WarehouseItem struct {
	ID           uint64 `gorm:"primaryKey" json:"id"`
	MatchID      uint64 `gorm:"not null;uniqueIndex:uniq_auction_warehouse_items_match_id_reveal_order,priority:1" json:"match_id"`
	RevealOrder  int32  `gorm:"not null;uniqueIndex:uniq_auction_warehouse_items_match_id_reveal_order,priority:2" json:"reveal_order"` // 公开品质的顺序, 从 1 开始, 第 N 轮公开前 N 件
	ItemID       uint64 `gorm:"not null" json:"item_id"`
	Name         string `gorm:"size:64;not null" json:"name"`
	ImageURL     string `gorm:"size:1024;not null;default:''" json:"image_url"`
	Quality      int16  `gorm:"not null" json:"quality"`
	Width        int16  `gorm:"not null" json:"width"`
	Height       int16  `gorm:"not null" json:"height"`
	X            int16  `gorm:"not null" json:"x"`
	Y            int16  `gorm:"not null" json:"y"`
	SellCurrency string `gorm:"size:32;not null" json:"sell_currency"`
	SellPrice    int64  `gorm:"not null" json:"sell_price"`
}

// TableName 自定义表名
func (WarehouseItem) TableName() string { return "auction_warehouse_items" }
