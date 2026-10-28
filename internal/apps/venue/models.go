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
	"time"

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
	Bots          BotRule               `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"bots"`         // 人机参数
	Pve           PveRule               `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"pve"`          // 人机模式防刷参数
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

// WarehouseRule 仓库生成规则: 各档位抽取的物品合在一起, 按面积从大到小摆进格子, 摆不下时整仓重抽
type WarehouseRule struct {
	Grid       Grid   `json:"grid" label:"仓库格子"`
	Slots      []Slot `json:"slots" label:"档位" binding:"min=1,max=8,dive"`
	MaxRerolls int32  `json:"max_rerolls" label:"重抽次数上限" binding:"min=0,max=50"` // 超过后开局失败
}

// AuctionRule 拍卖参数, 金额按出价货币的最小单位
type AuctionRule struct {
	StartPrice     int64   `json:"start_price" label:"起拍价" binding:"min=1,max=1000000000000"`
	MaxCarry       int64   `json:"max_carry" label:"带入资金上限" binding:"min=1,max=1000000000000"` // 开局时冻结 min(可用余额, 上限), 出价不能超过它
	Rounds         int32   `json:"rounds" label:"轮数" binding:"min=1,max=10"`
	RoundSeconds   int32   `json:"round_seconds" label:"每轮时长" binding:"min=5,max=120"`
	RevealSeconds  int32   `json:"reveal_seconds" label:"开箱后停留时长" binding:"min=3,max=60"`            // 全部藏品扫描完后再停留的时长, 扫描时长见品质的 scan_ms
	QuickWin       []int32 `json:"quick_win" label:"速胜倍率" binding:"max=9,dive,min=10000,max=100000"` // 前 Rounds-1 轮的速胜倍率, 基点; 第一名不低于第二名的这么多倍时直接成交
	MaxExtraRounds int32   `json:"max_extra_rounds" label:"加轮上限" binding:"min=0,max=5"`              // 最后一轮最高价并列时加轮, 之后按先出价的人成交
}

// BotRule 人机参数, 系数都是基点
type BotRule struct {
	LevelWeights   []int32 `json:"level_weights" label:"人机等级权重" binding:"len=3,dive,min=0,max=100"`            // 低、中、高三个等级的权重
	EstimateErrors []int32 `json:"estimate_errors" label:"估值误差" binding:"len=3,dive,min=0,max=9000"`           // 低、中、高三个等级的估值误差, 估值 = 仓库价值 × (1 ± 误差)
	RoundFactors   []int32 `json:"round_factors" label:"每轮出价系数" binding:"min=1,max=12,dive,min=100,max=10000"` // 每轮出价 = 估值 × 系数, 轮数更多时沿用最后一个
	Jitter         int32   `json:"jitter" label:"出价浮动" binding:"min=0,max=5000"`                               // 出价在 ± 浮动内随机
	MaxFactor      int32   `json:"max_factor" label:"出价上限系数" binding:"min=100,max=10000"`                      // 出价不超过估值 × 上限系数
	BluffChance    int32   `json:"bluff_chance" label:"诈唬概率" binding:"min=0,max=10000"`
	BluffRounds    int32   `json:"bluff_rounds" label:"诈唬轮数" binding:"min=0,max=10"`      // 只在前几轮诈唬
	BluffFactor    int32   `json:"bluff_factor" label:"诈唬系数" binding:"min=100,max=10000"` // 诈唬时按估值 × 诈唬系数出价
}

// PveRule 人机模式的防刷参数
type PveRule struct {
	PracticeOnly   bool  `json:"practice_only" label:"只能练习"`                                        // 人机模式只能打练习局
	DailyProfitCap int64 `json:"daily_profit_cap" label:"每日收益上限" binding:"min=0,max=1000000000000"` // 当天在这张地图人机模式的净收益达到上限后转为练习局, 0 表示不单独限制; 全局的人机每日收益上限另外生效
	ValueFactor    int32 `json:"value_factor" label:"仓库价值系数" binding:"min=1000,max=10000"`          // 人机模式按这个系数缩放各档位的抽取次数
}

// ValueStats 模拟仓库的价值分布, 价值为出价货币的回收价之和
type ValueStats struct {
	FailureRate int32 `json:"failure_rate"` // 摆放失败率, 基点
	Mean        int64 `json:"mean"`
	P10         int64 `json:"p10"`
	P50         int64 `json:"p50"`
	P90         int64 `json:"p90"`
	Max         int64 `json:"max"`
	MeanPieces  int32 `json:"mean_pieces"` // 平均藏品件数, 向下取整
}

// Simulation 保存地图时模拟的结果
type Simulation struct {
	Runs        int        `json:"runs"`
	Ranked      ValueStats `json:"ranked"` // 排位模式, 不缩放抽取次数
	Pve         ValueStats `json:"pve"`    // 人机模式, 按仓库价值系数缩放抽取次数
	SimulatedAt time.Time  `json:"simulated_at"`
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

// Warehouse 生成的仓库
type Warehouse struct {
	Pieces []Piece `json:"pieces"` // 按摆放顺序, 面积大的在前
	Value  int64   `json:"value"`  // 出价货币的回收价之和, 其他货币的物品不计入
}
