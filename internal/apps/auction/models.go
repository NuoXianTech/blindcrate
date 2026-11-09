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

// Match 一局暗拍(拍场)的对局级数据, 与 match.Match 一一对应, 开局时创建; 拍场依次开拍 LotCount 个拍品, 见 Lot
type Match struct {
	MatchID   uint64     `gorm:"primaryKey;autoIncrement:false" json:"match_id"`
	VenueID   uint64     `gorm:"not null" json:"venue_id"`
	Venue     Snapshot   `gorm:"type:jsonb;serializer:json;not null" json:"venue"` // 开局时的地图配置, 之后修改地图不影响这一局
	LotCount  int32      `gorm:"not null;default:1" json:"lot_count"`              // 拍品数, 开局时公开
	Lot       int32      `gorm:"not null;default:1" json:"lot"`                    // 当前拍品的序号, 从 1 开始
	Outcome   string     `gorm:"size:16;not null;default:''" json:"outcome"`       // 拍场结算前为空, 之后为 sold(至少成交一个拍品)、unsold 或 cancelled
	Host      string     `gorm:"size:16;not null;default:''" json:"host"`          // 主持人的性格, 开局时选出, 整场不变, 拍场结束后才公开; 没有主持人提示时为空
	Hype      int32      `gorm:"not null;default:0" json:"hype"`                   // 当前热度, 0-100, 跨拍品延续, 每次变动见 HypeEvent
	CreatedAt time.Time  `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
	SettledAt *time.Time `json:"settled_at"` // 拍场结算或取消的时间
}

// TableName 自定义表名
func (Match) TableName() string { return "auction_matches" }

// Snapshot 开局时快照的地图配置
type Snapshot struct {
	Code        string               `json:"code"`
	Name        string               `json:"name"`
	BidCurrency string               `json:"bid_currency"`
	EntryCost   reward.Bundle        `json:"entry_cost"` // 开局时扣除的全部代价, 含出价货币的入场费
	Auction     venue.AuctionRule    `json:"auction"`    // 拍场的拍卖参数, 起拍价等按拍品的, 见 Lot.Rule
	Consumables venue.ConsumableRule `json:"consumables"`
	Host        venue.HostRule       `json:"host"`
	GapSeconds  int32                `json:"gap_seconds"` // 拍品开箱后停留多久进入下一个, 见 venue.LotRule
	TierNames   map[string]string    `json:"tier_names"`  // 档次代码 -> 名称, 拍场结束后公开拍品的档次
	HypeDecay   int32                `json:"hype_decay"`  // 每个拍品开场时热度向开局热度回落的比例, 基点, 开局时从玩法平衡设置快照
}

// Lot 拍场中的一个拍品(仓库), 开局时全部生成, 之后不变; 轮次在整个拍场内连续编号, 这个拍品从第 FirstRound 轮开始
//
// 唯一索引 uniq_auction_lots_match_id_lot_no 保证每个序号只有一个拍品。
type Lot struct {
	ID          uint64            `gorm:"primaryKey" json:"id"`
	MatchID     uint64            `gorm:"not null;uniqueIndex:uniq_auction_lots_match_id_lot_no,priority:1" json:"match_id"`
	LotNo       int32             `gorm:"not null;uniqueIndex:uniq_auction_lots_match_id_lot_no,priority:2" json:"lot_no"` // 出场顺序, 从 1 开始
	Tier        string            `gorm:"size:32;not null;default:''" json:"tier"`                                         // 档次代码, 不公开
	Grid        venue.Grid        `gorm:"type:jsonb;serializer:json;not null" json:"grid"`                                 // 格子大小, 开局时公开
	Odds        []venue.PieceOdds `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"odds"`                    // 按格子大小推断的公示分布, 人机估值和主持人用, 见 publicLots
	StartPrice  int64             `gorm:"not null;default:0" json:"start_price"`                                           // 起拍价、速胜下限和情报价格按格子大小推导, 见 venue.Venue.Calibrate
	SpeedFloor  int64             `gorm:"not null;default:0" json:"speed_floor"`
	IntelPrice  int64             `gorm:"not null;default:0" json:"intel_price"`
	FirstRound  int32             `gorm:"not null;default:0" json:"first_round"`                              // 开拍时的轮次, 还没开拍时为 0
	Value       int64             `gorm:"not null" json:"value"`                                              // 仓库价值, 出价货币的回收价之和
	PieceCount  int32             `gorm:"not null" json:"piece_count"`                                        // 藏品件数, 开箱前不公开
	Contenders  []int16           `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"contenders"` // 加轮时可以出价的座位, 为空表示所有人
	ExtraRounds int32             `gorm:"not null;default:0" json:"extra_rounds"`                             // 已加的轮数
	Outcome     string            `gorm:"size:16;not null;default:''" json:"outcome"`                         // 结算前为空, 之后为 sold、unsold 或 cancelled
	WinnerSeat  int16             `gorm:"not null;default:0" json:"winner_seat"`                              // 成交者的座位, 没有成交时为 0
	Price       int64             `gorm:"not null;default:0" json:"price"`                                    // 成交价
	SettledAt   *time.Time        `json:"settled_at"`                                                         // 结算的时间, 即开始扫描的时间
}

// TableName 自定义表名
func (Lot) TableName() string { return "auction_lots" }

// CanBid 座位在本轮是否可以出价, 加轮时只有并列最高的人可以
func (l *Lot) CanBid(seat int16) bool {
	return len(l.Contenders) == 0 || slices.Contains(l.Contenders, seat)
}

// Round 拍场的第 round 轮是这个拍品的第几轮
func (l *Lot) Round(round int32) int32 {
	return round - l.FirstRound + 1
}

// Rule 这个拍品的拍卖参数: 拍场的参数, 起拍价、速胜下限和情报价格换成拍品的
func (l *Lot) Rule(base *venue.AuctionRule) venue.AuctionRule {
	return base.ForTier(&venue.TierAuction{StartPrice: l.StartPrice, SpeedFloor: l.SpeedFloor, IntelPrice: l.IntelPrice})
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
	Carry       int64         `gorm:"not null;default:0" json:"carry"`                                    // 带入资金, 整个拍场共用; 真人在准入时自选, 见 CarryPreference
	Frozen      int64         `gorm:"not null;default:0" json:"frozen"`                                   // 冻结的带入资金, 练习局和人机为 0
	Spent       int64         `gorm:"not null;default:0" json:"spent"`                                    // 已成交的拍品的成交价之和, 剩余资金为 Carry - Spent
	Estimate    int64         `gorm:"not null;default:0" json:"estimate"`                                 // 人机对当前拍品按公开信息的最新估值, 真人为 0
	Bid         int64         `gorm:"not null;default:0" json:"bid"`                                      // 当前拍品的出价, 0 表示还没出过价; 每个拍品开拍时清零
	BidAt       *time.Time    `json:"bid_at"`                                                             // 当前出价的提交时间, 加轮用完仍并列时先出价的人成交
	Profit      int64         `gorm:"not null;default:0" json:"profit"`                                   // 拍场结算后的盈亏, 出价货币, 不含准入时扣除的入场代价; 练习局按正式局计算
	Rank        int16         `gorm:"not null;default:0" json:"rank"`                                     // 拍场的名次, 全部流拍时为 0
	RatingDelta int32         `gorm:"not null;default:0" json:"rating_delta"`                             // 本局段位分的变化, 只有排位且至少成交一个拍品时有
	Rating      int32         `gorm:"not null;default:0" json:"rating"`                                   // 结算后的段位分, 同上
	CreatedAt   time.Time     `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time     `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Player) TableName() string { return "auction_players" }

// Remaining 剩余资金, 当前拍品出价的上限
func (p *Player) Remaining() int64 { return p.Carry - p.Spent }

// CarryPreference 玩家在一张地图上选的带入金额, 准入时按它冻结, 没有时为地图的带入资金上限
type CarryPreference struct {
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false" json:"-"`
	Venue     string    `gorm:"primaryKey;size:32" json:"venue"` // 地图代码
	Amount    int64     `gorm:"not null" json:"amount"`
	UpdatedAt time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (CarryPreference) TableName() string { return "auction_carry_preferences" }

// Bid 每轮的一次出价, 真人提交时写入, 人机在本轮结束时写入; 拍品的前 N-1 轮结束后公开, 最后一轮和加轮在这个拍品结算后公开
//
// 唯一索引 uniq_auction_bids_match_id_round_seat 保证每人每轮只出价一次, 轮次在整个拍场内连续编号。
type Bid struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	MatchID   uint64    `gorm:"not null;uniqueIndex:uniq_auction_bids_match_id_round_seat,priority:1" json:"match_id"`
	LotNo     int32     `gorm:"not null;default:1" json:"lot_no"`
	Round     int32     `gorm:"not null;uniqueIndex:uniq_auction_bids_match_id_round_seat,priority:2" json:"round"`
	Seat      int16     `gorm:"not null;uniqueIndex:uniq_auction_bids_match_id_round_seat,priority:3" json:"seat"`
	Amount    int64     `gorm:"not null;default:0" json:"amount"` // 0 表示不加价
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (Bid) TableName() string { return "auction_bids" }

// WarehouseItem 一个拍品(仓库)里的一件藏品, 开局时快照名称、回收价和扫描时长; 一个拍品中按位置(X, Y)区分, 位置不重复
type WarehouseItem struct {
	ID           uint64 `gorm:"primaryKey" json:"id"`
	MatchID      uint64 `gorm:"not null;index:idx_auction_warehouse_items_match_id" json:"match_id"`
	LotNo        int32  `gorm:"not null;default:1" json:"lot_no"`
	Preview      bool   `gorm:"not null;default:false" json:"preview"` // 是否在这个拍品的第 1 轮扫出轮廓(占的格子), 所有人都看得到, 不含品质和名称
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

// Use 对局中用掉的一个道具, 结果在使用时算出并保存, 之后直接读取; 只作用于使用时的拍品
//
// 唯一索引 uniq_auction_uses_match_id_seat_item_id_round 保证每人每轮同一种道具只用一次。
type Use struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	MatchID   uint64         `gorm:"not null;uniqueIndex:uniq_auction_uses_match_id_seat_item_id_round,priority:1" json:"match_id"`
	LotNo     int32          `gorm:"not null;default:1" json:"lot_no"`
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

// HostHint 主持人在一轮开始时公开的关于当前拍品的提示, 只追加; 字段按类型使用其中的一部分, 见 constants.go 的 HostHint
//
// 唯一索引 uniq_auction_hints_match_id_round 保证每轮只有一条。
type HostHint struct {
	ID        uint64    `gorm:"primaryKey" json:"-"`
	MatchID   uint64    `gorm:"not null;uniqueIndex:uniq_auction_hints_match_id_round,priority:1" json:"-"`
	LotNo     int32     `gorm:"not null;default:1" json:"lot_no"`
	Round     int32     `gorm:"not null;uniqueIndex:uniq_auction_hints_match_id_round,priority:2" json:"round"`
	Kind      string    `gorm:"size:16;not null" json:"kind"`
	Quality   int16     `gorm:"not null;default:0" json:"quality"`
	Count     int32     `gorm:"not null;default:0" json:"count"`
	X         int16     `gorm:"not null;default:0" json:"x"` // 提示针对的藏品的位置
	Y         int16     `gorm:"not null;default:0" json:"y"`
	Above     bool      `gorm:"not null;default:false" json:"above"`
	Truthful  bool      `gorm:"not null" json:"truthful"` // 是否属实, 拍场结束前不给玩家看
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (HostHint) TableName() string { return "auction_hints" }

// HypeEvent 热度的一次变动, 只追加
type HypeEvent struct {
	ID        uint64    `gorm:"primaryKey" json:"-"`
	MatchID   uint64    `gorm:"not null;index:idx_auction_hype_events_match_id" json:"-"`
	LotNo     int32     `gorm:"not null;default:1" json:"lot_no"`
	Round     int32     `gorm:"not null" json:"round"` // 开局和拍品开场(热度回落)时为 0, 之后为结束的轮次
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
