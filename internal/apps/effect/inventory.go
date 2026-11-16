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

package effect

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
)

// InventoryContext 在背包中使用物品时效果看到的上下文, 背包场景的效果都用它
type InventoryContext struct {
	Tx     *gorm.DB
	UserID uint64
	Item   *item.Item
}

// 背包场景的使用方式, 客户端按方式展示, 新增效果时复用这几种即可
const (
	ModeGrant  = "grant"  // 得到 Options[0] 的全部内容, 如礼包
	ModeRandom = "random" // 按概率得到 Options 中的一项, 如箱子
	ModeChoose = "choose" // 玩家从 Options 中选一项, 如自选礼包
	ModeAction = "action" // 不发放内容, 只执行 Plan.Act, 如改名卡
)

// Option 使用后可以得到的一组内容
type Option struct {
	Bundle      reward.Bundle
	MaxAmount   int64   // ModeRandom 且只有一行时这一行数量的上限, 不大于数量时数量固定
	Probability float64 // ModeRandom 时得到这一项的概率, 各项之和为 1
}

// Input 使用时需要玩家填写的一段文本, 如新的昵称
type Input struct {
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
	MaxLength   int    `json:"max_length"` // 最多多少个字符
}

// Usage 一次使用, 传给 Plan.Act
type Usage struct {
	RecordID uint64 // 使用记录 ID
	UserID   uint64
	Item     *item.Item
	Input    string // 玩家填写的文本, 已去掉首尾空白; 没有 Plan.Input 时为空
}

// Plan 使用一个物品会发生什么, 背包场景的效果按参数给出; 预览和使用都先得到它, 预览不调用 Draw 和 Act
//
// 效果自己不写库: 发放由通用的使用流程完成, 其他改动放在 Act 里, 与扣除物品在同一个事务中。
type Plan struct {
	Mode    string   // 见 Mode 开头的常量
	Summary string   // 一句话说明使用后会发生什么, 展示给玩家
	Options []Option // 可以得到的内容; ModeGrant 只有一项, ModeAction 没有
	Input   *Input   // 需要玩家填写的文本, 不需要时为 nil
	Detail  string   // 记在使用记录里的补充信息, 如掉落池代码
	BizType string   // 发放内容的流水业务类型, 空为 BizUseReward
	// Draw ModeRandom 时抽取一项, 必须给出
	Draw func() reward.Bundle
	// Act 发放之外的改动, 在使用的事务中执行; 返回展示给玩家的一句结果。不能使用时返回 Unusable 的错误, 整次使用回滚
	Act func(tx *gorm.DB, usage *Usage) (string, error)
}

// UseError 物品现在不能使用, Error 返回的提示可以直接展示给玩家
type UseError struct {
	Reason string
}

func (e *UseError) Error() string { return e.Reason }

// Unusable 物品现在不能使用的错误, 效果在生成 Plan 或 Act 时返回; reason 展示给玩家
func Unusable(reason string) error { return &UseError{Reason: reason} }

// UseRecord 在背包中使用一次物品的记录, 记下使用时物品的名称、效果和得到的内容, 之后修改物品不影响已有记录
//
// 扣除物品和发放内容的流水, biz_id 都是使用记录 ID。
//
// 联合索引：
//   - uniq_item_use_records_idempotency_key (user_id, idempotency_key)：幂等键在同一用户内唯一
type UseRecord struct {
	ID             uint64        `gorm:"primaryKey" json:"id"`
	UserID         uint64        `gorm:"not null;index:uniq_item_use_records_idempotency_key,unique,priority:1" json:"user_id"`
	IdempotencyKey string        `gorm:"size:64;not null;index:uniq_item_use_records_idempotency_key,unique,priority:2" json:"-"` // 同一用户内唯一
	ItemID         uint64        `gorm:"not null" json:"item_id"`
	ItemName       string        `gorm:"size:64;not null" json:"item_name"`                              // 使用时物品的名称
	Effect         string        `gorm:"size:32;not null" json:"effect"`                                 // 使用时物品的效果代码
	Detail         string        `gorm:"size:64;not null;default:''" json:"detail"`                      // 效果给出的补充信息, 如掉落池代码
	Choice         int32         `gorm:"not null;default:0" json:"choice"`                               // 自选时选的第几项, 从 0 开始
	Input          string        `gorm:"size:255;not null;default:''" json:"input"`                      // 玩家填写的文本
	Reward         reward.Bundle `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"reward"` // 发放的内容, 没有时为空
	Message        string        `gorm:"size:255;not null;default:''" json:"message"`                    // 效果给出的一句结果
	CreatedAt      time.Time     `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (UseRecord) TableName() string { return "item_use_records" }

// loadPlan 获取物品和使用它的 Plan
//
// 物品不存在、已删除或不能在背包中使用时返回 ItemNotUsable; 现在不能使用时返回 *UseError。
// 已下架的物品照常使用, 下架只是不再发放。
func loadPlan(tx *gorm.DB, userID, itemID uint64) (*item.Item, *Plan, error) {
	// 物品不存在时不算查询错误, 用 Find 查询
	var target item.Item
	if err := tx.Where("id = ?", itemID).Limit(1).Find(&target).Error; err != nil {
		return nil, nil, err
	}
	if target.ID == 0 || SceneOf(target.Effect) != SceneInventory {
		return nil, nil, errors.New(ItemNotUsable)
	}
	ctx := &InventoryContext{Tx: tx, UserID: userID, Item: &target}
	plan, err := Apply[*InventoryContext, *Plan](SceneInventory, target.Effect, ctx, target.EffectParams)
	if err != nil {
		return nil, nil, err
	}
	switch {
	case plan.Mode == ModeGrant && len(plan.Options) != 1,
		plan.Mode == ModeRandom && (len(plan.Options) == 0 || plan.Draw == nil),
		plan.Mode == ModeChoose && len(plan.Options) == 0,
		plan.Mode == ModeAction && (len(plan.Options) != 0 || plan.Act == nil):
		return nil, nil, fmt.Errorf("effect: %s returned a malformed %s plan", target.Effect, plan.Mode)
	}
	return &target, plan, nil
}

// quantityOf 用户背包中某种物品的数量, 没有时为 0
func quantityOf(tx *gorm.DB, userID, itemID uint64) (int64, error) {
	var row inventory.UserItem
	if err := tx.Where("user_id = ? AND item_id = ?", userID, itemID).Limit(1).Find(&row).Error; err != nil {
		return 0, err
	}
	return row.Quantity, nil
}

// OptionView 玩家看到的一组可以得到的内容
type OptionView struct {
	Lines       []reward.Line `json:"lines"`       // amount 为数量; 按概率得到且数量不固定时为最小数量
	MaxAmount   int64         `json:"max_amount"`  // 只有一行时这一行数量的上限, 不大于 amount 时数量固定
	Probability float64       `json:"probability"` // 按概率得到时抽中这一项的概率, 其他方式为 0
}

// UseView 玩家看到的一个可以在背包中使用的物品
type UseView struct {
	ItemID      uint64       `json:"item_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Quality     int16        `json:"quality"`
	ImageURL    string       `json:"image_url"`
	Quantity    int64        `json:"quantity"` // 背包中的数量
	Effect      string       `json:"effect"`   // 效果代码
	Label       string       `json:"label"`    // 效果的名称, 如 "开启箱子"
	Mode        string       `json:"mode"`     // 使用方式: grant 得到全部内容, random 按概率得到一项, choose 自选一项, action 只执行动作
	Summary     string       `json:"summary"`  // 一句话说明使用后会发生什么
	Input       *Input       `json:"input"`    // 使用时需要填写的文本, 不需要时为 null
	Options     []OptionView `json:"options"`  // 可以得到的内容; random 时按品质从高到低, 货币排在最后, 概率之和为 1
}

// QueryUse 获取一个可以在背包中使用的物品、用户持有的数量和使用后可以得到的内容, 用于使用前的确认和公示概率
//
// 不要求用户持有这个物品。
func QueryUse(ctx context.Context, userID, itemID uint64) (*UseView, error) {
	tx := db.DB(ctx)
	target, plan, err := loadPlan(tx, userID, itemID)
	if err != nil {
		return nil, err
	}
	quantity, err := quantityOf(tx, userID, itemID)
	if err != nil {
		return nil, err
	}

	bundles := make([]reward.Bundle, len(plan.Options))
	for i, option := range plan.Options {
		bundles[i] = option.Bundle
	}
	lines, err := reward.Lines(tx, bundles...)
	if err != nil {
		return nil, err
	}
	options := make([]OptionView, len(plan.Options))
	for i, option := range plan.Options {
		options[i] = OptionView{Lines: lines[i], MaxAmount: option.MaxAmount, Probability: option.Probability}
	}
	if plan.Mode == ModeRandom {
		// 稳定排序, 同一品质的保持原来的顺序; 自选的不能排序, 选的是第几项
		slices.SortStableFunc(options, func(a, b OptionView) int {
			return cmp.Compare(topQuality(b.Lines), topQuality(a.Lines))
		})
	}

	return &UseView{
		ItemID:      target.ID,
		Name:        target.Name,
		Description: target.Description,
		Quality:     target.Quality,
		ImageURL:    target.ImageURL,
		Quantity:    quantity,
		Effect:      target.Effect,
		Label:       definitions[target.Effect].label,
		Mode:        plan.Mode,
		Summary:     plan.Summary,
		Input:       plan.Input,
		Options:     options,
	}, nil
}

// topQuality 一组内容中最高的品质, 货币为 0
func topQuality(lines []reward.Line) int16 {
	var top int16
	for _, line := range lines {
		top = max(top, line.Quality)
	}
	return top
}

// InventoryUseRequest 在背包中使用一个物品的请求体
type InventoryUseRequest struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次使用生成一个, 重复提交只使用一次并返回原来的结果
	Choice         int32  `json:"choice" binding:"min=0"`                               // 自选时选第几项, 从 0 开始, 顺序与预览相同
	Input          string `json:"input" binding:"max=255"`                              // 需要填写文本时的内容
}

// UseResult 使用一次的结果
type UseResult struct {
	ID       uint64        `json:"id"`       // 使用记录 ID
	Received []reward.Line `json:"received"` // 得到并已经发放的货币和物品, 没有时为空
	Message  string        `json:"message"`  // 效果给出的一句结果, 没有时为空
	Quantity int64         `json:"quantity"` // 使用后背包中剩余的数量
}

// UseInInventory 在背包中使用一个物品: 扣除 1 个, 发放得到的内容并执行效果的其他改动; 同一个幂等键重复提交时返回原来的结果, replayed 为 true
//
// 一个事务内完成, 任何一步失败整体回滚, 物品不会被扣除; 同一用户的使用在咨询锁上排队, 幂等键不会被并发请求绕过。
// 背包中没有这个物品时返回 inventory.InsufficientQuantity; 现在不能使用时返回 *UseError。
func UseInInventory(ctx context.Context, userID, itemID uint64, req *InventoryUseRequest) (*UseResult, bool, error) {
	var (
		record   UseRecord
		replayed bool
		results  []*reward.Result
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := db.Lock(tx, fmt.Sprintf("item_use:%d", userID)); err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND idempotency_key = ?", userID, req.IdempotencyKey).Limit(1).Find(&record).Error; err != nil {
			return err
		}
		if record.ID != 0 {
			if record.ItemID != itemID {
				return errors.New(IdempotencyConflict)
			}
			replayed = true
			return nil
		}

		var err error
		results, err = useInInventory(tx, userID, itemID, req, &record)
		return err
	})
	if err != nil {
		return nil, false, err
	}
	for _, result := range results {
		result.Push(ctx)
	}

	tx := db.DB(ctx)
	lines, err := reward.Lines(tx, record.Reward)
	if err != nil {
		return nil, false, err
	}
	quantity, err := quantityOf(tx, userID, itemID)
	if err != nil {
		return nil, false, err
	}
	return &UseResult{ID: record.ID, Received: lines[0], Message: record.Message, Quantity: quantity}, replayed, nil
}

// useInInventory 在已加锁的事务中完成一次使用, 把使用记录写入 record, 返回扣除物品和发放内容的结果, 用于提交后推送
func useInInventory(tx *gorm.DB, userID, itemID uint64, req *InventoryUseRequest, record *UseRecord) ([]*reward.Result, error) {
	target, plan, err := loadPlan(tx, userID, itemID)
	if err != nil {
		return nil, err
	}

	var granted reward.Bundle
	switch plan.Mode {
	case ModeGrant:
		granted = plan.Options[0].Bundle
	case ModeRandom:
		granted = plan.Draw()
	case ModeChoose:
		if int(req.Choice) >= len(plan.Options) {
			return nil, errors.New(ChoiceInvalid)
		}
		granted = plan.Options[req.Choice].Bundle
	}
	if plan.Mode != ModeChoose && req.Choice != 0 {
		return nil, errors.New(ChoiceInvalid)
	}
	input := strings.TrimSpace(req.Input)
	switch {
	case plan.Input == nil && input != "":
		return nil, errors.New(InputInvalid)
	case plan.Input != nil && (input == "" || utf8.RuneCountInString(input) > plan.Input.MaxLength):
		return nil, errors.New(InputInvalid)
	}
	if !granted.IsEmpty() {
		// 已下架的物品按回收价折成货币, 与其他奖励一致; 什么也发不出时不能使用
		if granted, err = reward.Prepare(tx, granted); err != nil {
			if reward.FailureOf(err) != reward.Delivered {
				return nil, Unusable(ItemUnavailable)
			}
			return nil, err
		}
	}

	*record = UseRecord{
		UserID:         userID,
		IdempotencyKey: req.IdempotencyKey,
		ItemID:         target.ID,
		ItemName:       target.Name,
		Effect:         target.Effect,
		Detail:         plan.Detail,
		Choice:         req.Choice,
		Input:          input,
		Reward:         granted,
	}
	if err := tx.Create(record).Error; err != nil {
		// 咨询锁之外的兜底, 正常不会走到
		if db.IsUniqueViolation(err) {
			return nil, errors.New(DuplicateRequest)
		}
		return nil, err
	}
	// 扣物品和发内容是两次变动, 先按统一的顺序锁齐, 否则与同时进行的出售互相等待而死锁
	if err := reward.Lock(tx, userID, reward.Bundle{Items: []reward.ItemLine{{ItemID: target.ID, Quantity: -1}}}, granted); err != nil {
		return nil, err
	}
	bizID := strconv.FormatUint(record.ID, 10)
	used, _, err := Consume(tx, &UseRequest{
		UserID:         userID,
		ItemID:         target.ID,
		BizID:          bizID,
		IdempotencyKey: "item_use:" + bizID + ":use",
		Remark:         definitions[target.Effect].label,
		Scopes:         []string{stats.ScopeAll},
	})
	if err != nil {
		return nil, err
	}
	results := []*reward.Result{used}
	if !granted.IsEmpty() {
		result, _, err := reward.Apply(tx, &reward.ApplyRequest{
			UserID:         userID,
			Bundle:         granted,
			BizType:        cmp.Or(plan.BizType, BizUseReward),
			BizID:          bizID,
			IdempotencyKey: "item_use:" + bizID + ":reward",
			Remark:         target.Name,
		})
		if reward.FailureOf(err) != reward.Delivered {
			return nil, Unusable(ItemUnavailable)
		}
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if plan.Act != nil {
		message, err := plan.Act(tx, &Usage{RecordID: record.ID, UserID: userID, Item: target, Input: input})
		if err != nil {
			return nil, err
		}
		if message != "" {
			record.Message = message
			if err := tx.Model(record).Update("message", message).Error; err != nil {
				return nil, err
			}
		}
	}
	return results, nil
}
