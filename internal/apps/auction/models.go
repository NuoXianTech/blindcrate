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

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
)

// Match 一局暗拍的对局级数据, 与 match.Match 一一对应, 开局时创建
type Match struct {
	MatchID     uint64     `gorm:"primaryKey;autoIncrement:false" json:"match_id"`
	VenueID     uint64     `gorm:"not null" json:"venue_id"`
	Venue       Snapshot   `gorm:"type:jsonb;serializer:json;not null" json:"venue"`                   // 开局时的地图配置, 之后修改地图不影响这一局
	Value       int64      `gorm:"not null" json:"value"`                                              // 仓库价值, 出价货币的回收价之和
	PieceCount  int32      `gorm:"not null" json:"piece_count"`                                        // 藏品件数, 开箱前不公开
	Contenders  []int16    `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"contenders"` // 加轮时可以出价的座位, 为空表示所有人
	ExtraRounds int32      `gorm:"not null;default:0" json:"extra_rounds"`                             // 已加的轮数
	Outcome     string     `gorm:"size:16;not null;default:''" json:"outcome"`                         // 结算前为空, 之后为 sold、unsold 或 cancelled
	WinnerSeat  int16      `gorm:"not null;default:0" json:"winner_seat"`                              // 成交者的座位, 没有成交时为 0
	Price       int64      `gorm:"not null;default:0" json:"price"`                                    // 成交价
	Host        string     `gorm:"size:16;not null;default:''" json:"host"`                            // 主持人的性格, 开局时选出, 开箱后才公开; 没有主持人提示时为空
	Hype        int32      `gorm:"not null;default:0" json:"hype"`                                     // 当前热度, 0-100, 每次变动见 HypeEvent
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
	Code        string               `json:"code"`
	Name        string               `json:"name"`
	BidCurrency string               `json:"bid_currency"`
	EntryCost   reward.Bundle        `json:"entry_cost"` // 开局时扣除的全部代价, 含出价货币的入场费
	Grid        venue.Grid           `json:"grid"`
	Auction     venue.AuctionRule    `json:"auction"`
	Consumables venue.ConsumableRule `json:"consumables"`
	Host        venue.HostRule       `json:"host"`
	Odds        []venue.PieceOdds    `json:"odds"` // 按公示概率算出的藏品分布, 人机估值用
}

// Player 一局暗拍中的一个座位
//
// 真人在准入时创建, 此时还没有座位, 开局时写入座位号; 人机在开局时创建。
type Player struct {
	ID          uint64        `gorm:"primaryKey" json:"id"`
	MatchID     uint64        `gorm:"not null;index:idx_auction_players_match_id" json:"match_id"`
	Seat        int16         `gorm:"not null;default:0" json:"seat"`
	UserID      uint64        `gorm:"not null;default:0" json:"user_id"`                                  // 人机为 0
	Persona     bot.Snapshot  `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"persona"`    // 人机的风格, 开局时快照, 真人为空
	Seed        int64         `gorm:"not null;default:0" json:"-"`                                        // 人机在这一局的随机种子, 开局时生成, 真人为 0
	IsPractice  bool          `gorm:"not null;default:false" json:"is_practice"`                          // 练习局, 准入时决定, 之后不再变化
	EntryCost   reward.Bundle `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"entry_cost"` // 实际扣除的入场费和入场代价, 练习局和人机为空
	Carry       int64         `gorm:"not null;default:0" json:"carry"`                                    // 带入资金, 出价的上限
	Frozen      int64         `gorm:"not null;default:0" json:"frozen"`                                   // 冻结的带入资金, 练习局和人机为 0
	Estimate    int64         `gorm:"not null;default:0" json:"estimate"`                                 // 人机按公开信息的最新估值, 真人为 0
	Bid         int64         `gorm:"not null;default:0" json:"bid"`                                      // 当前出价, 0 表示还没出过价
	BidAt       *time.Time    `json:"bid_at"`                                                             // 当前出价的提交时间, 加轮用完仍并列时先出价的人成交
	Feedback    []Feedback    `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"feedback"`   // 每轮结束后的反馈, 人机也有, 用于决策
	Profit      int64         `gorm:"not null;default:0" json:"profit"`                                   // 结算后的盈亏, 出价货币; 练习局按正式局计算
	RatingDelta int32         `gorm:"not null;default:0" json:"rating_delta"`                             // 本局段位分的变化, 只有排位成交时有
	Rating      int32         `gorm:"not null;default:0" json:"rating"`                                   // 结算后的段位分, 只有排位成交时有
	CreatedAt   time.Time     `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time     `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
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

// WarehouseItem 仓库里的一件藏品, 开局时快照名称、回收价和扫描时长; 一局中按位置(X, Y)区分, 位置不重复
type WarehouseItem struct {
	ID           uint64 `gorm:"primaryKey" json:"id"`
	MatchID      uint64 `gorm:"not null;index:idx_auction_warehouse_items_match_id" json:"match_id"`
	Preview      bool   `gorm:"not null;default:false" json:"preview"` // 是否在第 1 轮扫出轮廓(占的格子), 所有人都看得到, 不含品质和名称
	ScanOrder    int32  `gorm:"not null;default:0" json:"scan_order"`  // 开箱时扫描的顺序, 从 1 开始, 按位置从上到下、从左到右
	ScanMs       int32  `gorm:"not null;default:0" json:"scan_ms"`     // 开箱时扫描这件的时长, 毫秒, 开局时按品质快照
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

// Use 对局中用掉的一个道具, 结果在使用时算出并保存, 之后直接读取
//
// 唯一索引 uniq_auction_uses_match_id_seat_item_id_round 保证每人每轮同一种道具只用一次。
type Use struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	MatchID   uint64         `gorm:"not null;uniqueIndex:uniq_auction_uses_match_id_seat_item_id_round,priority:1" json:"match_id"`
	Seat      int16          `gorm:"not null;uniqueIndex:uniq_auction_uses_match_id_seat_item_id_round,priority:2" json:"seat"`
	ItemID    uint64         `gorm:"not null;uniqueIndex:uniq_auction_uses_match_id_seat_item_id_round,priority:3" json:"item_id"`
	Round     int32          `gorm:"not null;uniqueIndex:uniq_auction_uses_match_id_seat_item_id_round,priority:4" json:"round"`
	Name      string         `gorm:"size:64;not null" json:"name"`                                   // 使用时的物品名称
	Effect    string         `gorm:"size:32;not null" json:"effect"`                                 // 使用时的效果代码
	Params    map[string]any `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"params"` // 使用时的效果参数
	Result    Intel          `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"result"` // 得到的情报, 只给使用者看
	Cost      int64          `gorm:"not null;default:0" json:"cost"`                                 // 向主持人买情报时付的出价货币, 用道具时为 0
	CreatedAt time.Time      `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (Use) TableName() string { return "auction_uses" }

// IsIntel 是否为向主持人买的情报, 不是用掉的道具
func (u *Use) IsIntel() bool { return u.ItemID == 0 }

// HostHint 主持人在一轮开始时公开的提示, 只追加; 字段按类型使用其中的一部分, 见 constants.go 的 HostHint
//
// 唯一索引 uniq_auction_hints_match_id_round 保证每轮只有一条。
type HostHint struct {
	ID        uint64    `gorm:"primaryKey" json:"-"`
	MatchID   uint64    `gorm:"not null;uniqueIndex:uniq_auction_hints_match_id_round,priority:1" json:"-"`
	Round     int32     `gorm:"not null;uniqueIndex:uniq_auction_hints_match_id_round,priority:2" json:"round"`
	Kind      string    `gorm:"size:16;not null" json:"kind"`
	Quality   int16     `gorm:"not null;default:0" json:"quality"`
	Count     int32     `gorm:"not null;default:0" json:"count"`
	X         int16     `gorm:"not null;default:0" json:"x"` // 提示针对的藏品的位置
	Y         int16     `gorm:"not null;default:0" json:"y"`
	Above     bool      `gorm:"not null;default:false" json:"above"`
	Truthful  bool      `gorm:"not null" json:"truthful"` // 是否属实, 开箱前不给玩家看
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (HostHint) TableName() string { return "auction_hints" }

// HypeEvent 热度的一次变动, 只追加
type HypeEvent struct {
	ID        uint64    `gorm:"primaryKey" json:"-"`
	MatchID   uint64    `gorm:"not null;index:idx_auction_hype_events_match_id" json:"-"`
	Round     int32     `gorm:"not null" json:"round"` // 开局时为 0, 之后为结束的轮次
	Raises    int32     `gorm:"not null;default:0" json:"raises"`
	Delta     int32     `gorm:"not null" json:"delta"`
	Hype      int32     `gorm:"not null" json:"hype"` // 变动后的热度
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (HypeEvent) TableName() string { return "auction_hype_events" }

// PieceIntel 道具看到的一件藏品, 带占的格子, 以位置(X, Y)区分; 不带扫描顺序, 否则能从中推出件数
type PieceIntel struct {
	X        int16  `json:"x"`
	Y        int16  `json:"y"`
	Width    int16  `json:"width"`
	Height   int16  `json:"height"`
	Quality  int16  `json:"quality"`             // 0 为只扫出轮廓
	Name     string `json:"name,omitempty"`      // 只有看名称的道具有
	ImageURL string `json:"image_url,omitempty"` // 同 Name
}

// QualityCount 一种品质的藏品件数
type QualityCount struct {
	Quality int16 `json:"quality"`
	Count   int32 `json:"count"`
}

// Intel 使用一个道具得到的情报, 字段按效果使用其中的一部分
type Intel struct {
	Pieces     []PieceIntel   `json:"pieces,omitempty"`      // 看到品质或名称的藏品, 按位置从上到下、从左到右; 没有符合条件的藏品时为空
	MinQuality int16          `json:"min_quality,omitempty"` // 统计或查看的最低品质
	Counts     []QualityCount `json:"counts,omitempty"`      // 最低品质及以上各品质的件数, 品质从高到低, 没有的品质不列出
	ValueMin   int64          `json:"value_min,omitempty"`   // 仓库价值的区间, 出价货币
	ValueMax   int64          `json:"value_max,omitempty"`
}

// SuspectReview 管理员处理过的一对可疑账号, 见 QuerySuspects
//
// 两个账号之后又一起打了排位时重新进入待处理。
type SuspectReview struct {
	UserA       uint64    `gorm:"primaryKey;autoIncrement:false" json:"user_a"` // 较小的用户 ID
	UserB       uint64    `gorm:"primaryKey;autoIncrement:false" json:"user_b"` // 较大的用户 ID
	LastMatchID uint64    `gorm:"not null" json:"last_match_id"`                // 处理时两人最近一起打的对局
	Note        string    `gorm:"size:255;not null;default:''" json:"note"`
	ReviewedBy  uint64    `gorm:"not null;default:0" json:"reviewed_by"`
	CreatedAt   time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (SuspectReview) TableName() string { return "auction_suspect_reviews" }
