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

package venue

import (
	"slices"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"gorm.io/gorm"
)

// Venue 地图, 每张地图有人机和排位两种玩法
//
// 删除为软删除; 代码在已删除的地图中也唯一, 避免新地图沿用旧地图的统计。
type Venue struct {
	ID            uint64                `gorm:"primaryKey" json:"id"`
	Code          string                `gorm:"size:32;not null;uniqueIndex:uniq_venues_code" json:"code"` // 拼进玩法代码和统计范围, 创建后不能修改
	Name          string                `gorm:"size:64;not null" json:"name"`
	Description   string                `gorm:"size:255;not null;default:''" json:"description"`
	ImageURL      string                `gorm:"size:1024;not null;default:''" json:"image_url"`
	SortOrder     int32                 `gorm:"not null;default:0" json:"sort_order"`     // 大厅中从小到大排列
	IsEnabled     bool                  `gorm:"not null;default:false" json:"is_enabled"` // 停用后不能加入队列, 进行中的对局照常打完
	PveEnabled    bool                  `gorm:"not null;default:false" json:"pve_enabled"`
	RankedEnabled bool                  `gorm:"not null;default:false" json:"ranked_enabled"`
	BidCurrency   string                `gorm:"size:32;not null" json:"bid_currency"`                                 // 出价和计算仓库价值的货币
	EntryCost     reward.Bundle         `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"entry_cost"`   // 入场代价, 数量为正数, 开局时扣除
	Requirements  []condition.Condition `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"requirements"` // 入场条件, 需要全部满足
	Warehouse     WarehouseRule         `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"warehouse"`    // 仓库生成规则
	Auction       AuctionRule           `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"auction"`      // 拍卖参数
	Bots          BotRule               `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"bots"`         // 人机风格, 按模式配置出场的风格和权重
	Pve           PveRule               `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"pve"`          // 人机模式防刷参数
	Consumables   ConsumableRule        `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"consumables"`  // 对局中使用道具的规则
	Host          HostRule              `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"host"`         // 主持人的提示和情报
	Simulation    Simulation            `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"simulation"`   // 保存时的模拟结果, 掉落池或物品改动后需要重新保存才会更新
	CreatedAt     time.Time             `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time             `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt        `gorm:"index" json:"-"`
}

// TableName 自定义表名
func (Venue) TableName() string { return "venues" }

// Exact 按 ID 获取未删除的地图
func (v *Venue) Exact(tx *gorm.DB, id uint64) error {
	if err := tx.Where("id = ?", id).First(v).Error; err != nil {
		return err
	}
	return nil
}

// ExactCode 按代码获取未删除的地图
func (v *Venue) ExactCode(tx *gorm.DB, code string) error {
	if err := tx.Where("code = ?", code).First(v).Error; err != nil {
		return err
	}
	return nil
}

// ModeEnabled 地图启用且开放了这个模式
func (v *Venue) ModeEnabled(mode string) bool {
	switch mode {
	case ModePve:
		return v.IsEnabled && v.PveEnabled
	case ModeRanked:
		return v.IsEnabled && v.RankedEnabled
	}
	return false
}

// Entry 开局时扣除的全部代价: 出价货币的入场费在前, 之后是入场代价
func (v *Venue) Entry() reward.Bundle {
	var entry reward.Bundle
	if v.Auction.EntryFee > 0 {
		entry.Currencies = append(entry.Currencies, reward.CurrencyLine{Currency: v.BidCurrency, Amount: v.Auction.EntryFee})
	}
	entry.Currencies = append(entry.Currencies, v.EntryCost.Currencies...)
	entry.Items = slices.Clone(v.EntryCost.Items)
	return entry
}

// Grid 仓库的格子
type Grid struct {
	Width  int16 `json:"width" label:"仓库宽度" binding:"min=1,max=16"`
	Height int16 `json:"height" label:"仓库高度" binding:"min=1,max=16"`
}

// Slot 仓库的一个档位, 从一个掉落池中抽取若干次, 每次抽到的物品按数量放入仓库
type Slot struct {
	Pool   string `json:"pool" label:"掉落池" binding:"required,max=64"` // 掉落池代码, 只能有物品条目
	Min    int32  `json:"min" label:"最少次数" binding:"min=0,max=100"`   // 抽取次数在 [Min, Max] 内均匀随机
	Max    int32  `json:"max" label:"最多次数" binding:"min=0,max=100"`
	Chance int32  `json:"chance" label:"出现概率" binding:"min=1,max=10000"` // 这一档出现的概率, 基点, 10000 表示一定出现
}

// Swing 仓库的一档数量波动
type Swing struct {
	Factor int32 `json:"factor" label:"数量系数" binding:"min=2000,max=20000"` // 基点, 抽中这一档时按它缩放各档位的抽取次数
	Weight int32 `json:"weight" label:"权重" binding:"min=1,max=10000"`
}

// WarehouseRule 仓库生成规则: 各档位抽取的物品合在一起, 按面积从大到小摆进格子, 摆不下时整仓重抽
//
// 配置了数量波动时, 每局先按权重抽一档, 按它的系数缩放各档位的抽取次数; 藏品的多少在出价时可以从格子看出来,
// 看得懂格子的人能在大仓库多出价、小仓库少出价, 只出固定价格的人多半拍到小仓库。
type WarehouseRule struct {
	Grid       Grid    `json:"grid" label:"仓库格子"`
	Slots      []Slot  `json:"slots" label:"档位" binding:"min=1,max=8,dive"`
	Swings     []Swing `json:"swings" label:"数量波动" binding:"max=8,dive"`          // 为空表示不波动
	MaxRerolls int32   `json:"max_rerolls" label:"重抽次数上限" binding:"min=0,max=50"` // 超过后开局失败
}

// AuctionRule 拍卖参数, 金额按出价货币的最小单位
//
// 管理员只配置轮次、时长和速胜倍率; 金额和结算比例由保存时模拟的仓库价值分布推导(见 Calibrate), 提交的值会被覆盖。
// 旧数据中推导的字段为 0, 按旧规则进行: 不收入场费、速胜不限金额、按最高价成交、没有奖池, 重新保存后生效。
type AuctionRule struct {
	Rounds         int32   `json:"rounds" label:"轮数" binding:"min=1,max=10"`
	RoundSeconds   int32   `json:"round_seconds" label:"每轮时长" binding:"min=5,max=120"`
	RevealSeconds  int32   `json:"reveal_seconds" label:"开箱后停留时长" binding:"min=3,max=60"`            // 全部藏品扫描完后再停留的时长, 扫描时长见品质的 scan_ms
	QuickWin       []int32 `json:"quick_win" label:"速胜倍率" binding:"max=9,dive,min=10000,max=100000"` // 前 Rounds-1 轮的速胜倍率, 基点; 第一名不低于第二名的这么多倍且不低于速胜下限时直接成交
	MaxExtraRounds int32   `json:"max_extra_rounds" label:"加轮上限" binding:"min=0,max=5"`              // 最后一轮最高价并列时加轮, 之后按先出价的人成交

	EntryFee     int64 `json:"entry_fee"`     // 出价货币的入场费, 开局时与入场代价一起扣除
	StartPrice   int64 `json:"start_price"`   // 起拍价
	MaxCarry     int64 `json:"max_carry"`     // 带入资金上限, 开局时冻结 min(可用余额, 上限), 出价不能超过它
	BotCarry     int64 `json:"bot_carry"`     // 人机的带入资金, 不超过带入资金上限; 价值远高于期望的仓库人机追不上
	SpeedFloor   int64 `json:"speed_floor"`   // 速胜下限, 速胜时第一名的出价不能低于它
	LeadDiscount int32 `json:"lead_discount"` // 领先折让, 基点: 成交价 = 第二名的出价 + 领先部分 × (1 - 折让), 不低于起拍价
	PoolRate     int32 `json:"pool_rate"`     // 入场费进奖池的比例, 基点, 其余销毁; 成交后奖池分给出过价的真人输家
	RebateRate   int32 `json:"rebate_rate"`   // 真人成交价超过仓库价值时, 超出部分进奖池的比例, 基点
	RebateCap    int64 `json:"rebate_cap"`    // 每局进奖池的超付部分的上限
	PreviewRate  int32 `json:"preview_rate"`  // 首轮扫描比例, 基点: 第 1 轮每件藏品按这个概率公开占的格子, 至少一件; 0 表示不公开
	IntelPrice   int64 `json:"intel_price"`   // 向主持人买一次情报的价格, 从可用余额扣除; 0 表示不卖情报
}

// BotBudget 人机的带入资金; 推导出它之前保存的地图为带入资金上限
func (r *AuctionRule) BotBudget() int64 {
	if r.BotCarry > 0 {
		return r.BotCarry
	}
	return r.MaxCarry
}

// BotRule 人机风格: 每个模式出场的风格和权重, 开局时每个人机按权重随机选一种; 开放的模式至少配置一种
type BotRule struct {
	Pve    []bot.Weight `json:"pve" label:"人机模式" binding:"max=10,dive"`
	Ranked []bot.Weight `json:"ranked" label:"排位模式" binding:"max=10,dive"`
}

// Weights 这个模式出场的风格和权重
func (r *BotRule) Weights(mode string) []bot.Weight {
	switch mode {
	case ModePve:
		return r.Pve
	case ModeRanked:
		return r.Ranked
	}
	return nil
}

// PveRule 人机模式的防刷参数
type PveRule struct {
	PracticeOnly   bool  `json:"practice_only" label:"只能练习"`                                        // 人机模式只能打练习局
	DailyProfitCap int64 `json:"daily_profit_cap" label:"每日收益上限" binding:"min=0,max=1000000000000"` // 当天在这张地图人机模式的净收益达到上限后转为练习局, 0 表示不单独限制; 全局的人机每日收益上限另外生效
	ValueFactor    int32 `json:"value_factor" label:"仓库价值系数" binding:"min=1000,max=10000"`          // 人机模式按这个系数缩放各档位的抽取次数
}

// ConsumableLimit 一种模式中使用道具的次数限制
type ConsumableLimit struct {
	MaxPerMatch int32 `json:"max_per_match" label:"每局最多使用" binding:"min=0,max=10"` // 0 表示这个模式不能用道具
}

// ConsumableRule 对局中使用道具的规则, 只有效果属于对局场景的物品可以用
type ConsumableRule struct {
	Pve     ConsumableLimit `json:"pve" label:"人机模式"`
	Ranked  ConsumableLimit `json:"ranked" label:"排位模式"`
	Allowed []uint64        `json:"allowed" label:"可用道具" binding:"max=50"` // 物品 ID, 为空表示所有对局中可用的道具都可以用
}

// MaxPerMatch 这个模式每局最多使用几个道具, 0 表示不能用
func (r *ConsumableRule) MaxPerMatch(mode string) int32 {
	switch mode {
	case ModePve:
		return r.Pve.MaxPerMatch
	case ModeRanked:
		return r.Ranked.MaxPerMatch
	}
	return 0
}

// Allows 是否允许使用这个物品, 不检查物品的效果
func (r *ConsumableRule) Allows(itemID uint64) bool {
	return len(r.Allowed) == 0 || slices.Contains(r.Allowed, itemID)
}

// HostRule 主持人: 每轮开始时公开一条关于仓库的提示, 有一定概率说错, 性格开箱后才公开; 出价阶段还可以向他买只有自己看得到的情报
//
// 性格在开局时从 Personas 中随机选一种并公开, 决定说错时往哪边错, 见 constants.go 的 Host; 热度(Hype)由每轮的加价推动, 影响部分性格的撒谎率。
// 说错的提示在开箱后公开。情报的价格按期望价值推导(见 AuctionRule.IntelPrice), 买过情报的座位对所有人公开, 情报的内容只给自己。
type HostRule struct {
	Enabled        bool     `json:"enabled" label:"主持人提示"`                                             // 关闭时没有提示和热度, 情报商店仍按 IntelLimit 开放
	Personas       []string `json:"personas" label:"性格" binding:"max=3,dive,oneof=steady showman sly"` // 为空表示全部性格
	LieRate        int32    `json:"lie_rate" label:"撒谎率" binding:"min=0,max=2500"`                     // 基点, 提示说错的概率, 按性格和热度调整, 不超过 25%
	HighQuality    int16    `json:"high_quality" label:"高品质" input:"quality" binding:"min=0"`          // 提示涉及的品质不低于它时撒谎率 × 1.5, 0 表示不区分; 也是"高品质件数"提示统计的最低品质
	IntelLimit     int32    `json:"intel_limit" label:"每局情报上限" binding:"min=0,max=10"`                 // 每人每局最多买几次情报, 每轮最多一次, 0 表示不卖情报
	IntelPrecision int32    `json:"intel_precision" label:"估价情报误差" binding:"min=0,max=5000"`           // 后几轮的情报为仓库价值的区间, 宽度为价值 × 误差 × 2, 基点
}

// ValueStats 模拟仓库的价值分布, 价值为出价货币的回收价之和
type ValueStats struct {
	FailureRate int32 `json:"failure_rate"` // 摆放失败率, 基点
	Mean        int64 `json:"mean"`
	P10         int64 `json:"p10"`
	P50         int64 `json:"p50"`
	P90         int64 `json:"p90"`
	Max         int64 `json:"max"`
	CV          int32 `json:"cv"`          // 变异系数, 标准差 ÷ 平均值, 基点
	MeanPieces  int32 `json:"mean_pieces"` // 平均藏品件数, 向下取整
}

// Simulation 保存地图时模拟的结果
type Simulation struct {
	Runs        int        `json:"runs"`
	Ranked      ValueStats `json:"ranked"`  // 排位模式, 不缩放抽取次数
	Pve         ValueStats `json:"pve"`     // 人机模式, 按仓库价值系数缩放抽取次数
	Balance     *Balance   `json:"balance"` // 对局验算, 旧数据为空
	SimulatedAt time.Time  `json:"simulated_at"`
}

// Balance 对局验算: 按推导出的拍卖参数模拟几种阵容的对局, 检查成交价、速胜和盈亏是否落在安全区间
//
// 真人按模拟的估值误差出价, 人机按地图配置的风格出价; 结构类的闸门只看 4 个真人的参考阵容, 有人机的阵容看人机估值和真人盈亏。
type Balance struct {
	Runs   int          `json:"runs"`   // 每种阵容模拟的局数
	Target int32        `json:"target"` // 成交价 ÷ 仓库价值的目标, 基点, 为 1 - 出售手续费率, 系统收支在这里平衡
	Squads []SquadStats `json:"squads"` // 参考阵容在前
	Checks []Check      `json:"checks"`
	Passed bool         `json:"passed"` // 全部闸门通过, 未通过的地图不能启用
	// Version 推导和验算时玩法平衡设置的版本, 与当前版本不同时需要重新推导
	Version int64 `json:"version"`
}

// SquadStats 一种阵容的验算结果, 金额为出价货币的最小单位, 比例为基点
type SquadStats struct {
	Squad        string `json:"squad"`         // 见 constants.go 的 Squad
	Humans       int32  `json:"humans"`        // 真人数, 其余为人机
	Sold         int32  `json:"sold"`          // 成交率
	PriceValue   int32  `json:"price_value"`   // 成交价之和 ÷ 仓库价值之和
	SpeedWin     int32  `json:"speed_win"`     // 速胜率, 最后一轮之前成交的比例
	BidCV        int32  `json:"bid_cv"`        // 出价离散度: 每局出过价的人的出价变异系数 ÷ √3 的平均值
	TopShare     int32  `json:"top_share"`     // 最贵一件占仓库价值的比例的平均值
	AppraiseBias int32  `json:"appraise_bias"` // 人机估值之和 ÷ 仓库价值之和, 没有人机时为 0
	AvgRounds    int32  `json:"avg_rounds"`    // 平均在第几轮结束, 百分之一轮
	WinRate      int32  `json:"win_rate"`      // 每个真人席位的成交率
	SeatEV       int64  `json:"seat_ev"`       // 每个真人席位每局的平均盈亏, 藏品按扣掉出售手续费后的回收价计
	WinnerEV     int64  `json:"winner_ev"`     // 真人成交时的平均盈亏
	LoserEV      int64  `json:"loser_ev"`      // 真人没成交时的平均盈亏, 含奖池分红
	Share        int64  `json:"share"`         // 每局发给真人输家的奖池合计的平均值
	HypeHot      int32  `json:"hype_hot"`      // 每轮结束后热度火热(不低于 70)的比例, 加轮和成交的那一轮不算
	HypeCold     int32  `json:"hype_cold"`     // 同上, 冷清(不高于 30)的比例
	HypeEnd      int32  `json:"hype_end"`      // 结束时热度的平均值, 0-100
}

// Check 验算的一项闸门
type Check struct {
	Code   string `json:"code"`            // 见 constants.go 的 Check
	Squad  string `json:"squad,omitempty"` // 按阵容判定时为阵容
	Value  int64  `json:"value"`           // 单位见 Unit
	Min    *int64 `json:"min"`             // 下限, 为空表示不限
	Max    *int64 `json:"max"`             // 上限, 为空表示不限
	Unit   string `json:"unit"`            // 见 constants.go 的 Unit
	Passed bool   `json:"passed"`
}

// Piece 仓库里的一件藏品, 坐标从 0 开始, 左上角为原点
type Piece struct {
	ItemID       uint64 `json:"item_id"`
	Quality      int16  `json:"quality"`
	Width        int16  `json:"width"`
	Height       int16  `json:"height"`
	X            int16  `json:"x"`
	Y            int16  `json:"y"`
	SellCurrency string `json:"sell_currency"`
	SellPrice    int64  `json:"sell_price"` // 生成时的回收价
}

// PieceOdds 按公示概率算出的一种藏品(品质和尺寸相同的合在一起)在一个仓库中的期望件数和平均回收价, 用于人机估值
type PieceOdds struct {
	Quality int16   `json:"quality"`
	Width   int16   `json:"width"`
	Height  int16   `json:"height"`
	Count   float64 `json:"count"` // 期望件数, 不考虑摆不下时的重抽
	Price   float64 `json:"price"` // 平均回收价, 出价货币; 回收货币不同的按 0 计
}

// Warehouse 生成的仓库
type Warehouse struct {
	Pieces []Piece `json:"pieces"` // 按摆放顺序, 面积大的在前
	Value  int64   `json:"value"`  // 出价货币的回收价之和, 其他货币的物品不计入
}
