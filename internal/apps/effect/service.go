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
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"gorm.io/gorm"
)

func init() {
	item.RegisterCategory(CategoryConsumable, "道具")
	stats.RegisterMetric(MetricItemUsed, "使用道具次数")
}

// sceneNames 场景的展示名称
var sceneNames = map[string]string{
	SceneMatch: "对局中",
}

// InvalidError 效果配置不合法, Error 返回的提示可以直接展示给管理员
type InvalidError struct {
	Reason string
}

func (e *InvalidError) Error() string { return e.Reason }

// definition 一种效果, 由 Register 生成
type definition struct {
	scene       string
	label       string
	description string
	fields      []condition.Field
	decode      func(params map[string]any, strict bool) (any, error) // 返回参数结构体的指针
	apply       func(ctx, params any) (any, error)
}

var (
	// definitions 已注册的效果, 只在启动前写入
	definitions = map[string]*definition{}
	// codeOrder 效果的注册顺序, 后台按这个顺序展示
	codeOrder []string
)

// Register 注册一种效果, 需要在启动前调用; 代码为空、过长、重复, 场景不存在或参数结构体有不支持的字段时 panic
//
// C 为场景提供的上下文, R 为使用的结果, 由场景所在的模块定义, 同一场景的效果使用相同的 C 和 R;
// P 为参数结构体, 字段 tag 与条件类型相同(json、label、help、binding、input), P 的指针可以实现 condition.ParamsValidator。
// apply 不能写库, 结果由场景记录。
func Register[C, P, R any](code, scene, label, description string, apply func(ctx C, params *P) (R, error)) {
	if code == "" || len(code) > maxCodeLength {
		panic(fmt.Sprintf("effect: invalid code %q", code))
	}
	if _, ok := definitions[code]; ok {
		panic(fmt.Sprintf("effect: %s already registered", code))
	}
	if _, ok := sceneNames[scene]; !ok {
		panic(fmt.Sprintf("effect: unknown scene %s of %s", scene, code))
	}
	definitions[code] = &definition{
		scene:       scene,
		label:       label,
		description: description,
		fields:      condition.FieldsOf[P](code),
		decode: func(params map[string]any, strict bool) (any, error) {
			target := new(P)
			raw, err := json.Marshal(params)
			if err != nil {
				return nil, err
			}
			dec := json.NewDecoder(bytes.NewReader(raw))
			if strict {
				dec.DisallowUnknownFields()
			}
			if err := dec.Decode(target); err != nil {
				return nil, err
			}
			return target, nil
		},
		apply: func(ctx, params any) (any, error) {
			c, ok := ctx.(C)
			if !ok {
				return nil, fmt.Errorf("effect: %s got context %T", code, ctx)
			}
			return apply(c, params.(*P))
		},
	}
	codeOrder = append(codeOrder, code)
}

// Types 全部效果类型, 按注册顺序
func Types() []TypeView {
	views := make([]TypeView, 0, len(codeOrder))
	for _, code := range codeOrder {
		def := definitions[code]
		views = append(views, TypeView{
			Code:        code,
			Scene:       def.scene,
			SceneName:   sceneNames[def.scene],
			Label:       def.label,
			Description: def.description,
			Fields:      slices.Clone(def.fields),
		})
	}
	return views
}

// SceneOf 效果所属的场景, 效果不存在时为空
func SceneOf(code string) string {
	if def, ok := definitions[code]; ok {
		return def.scene
	}
	return ""
}

// Validate 保存物品前校验类别、效果和参数, 返回规范化后的参数: 按参数结构体重新编码, 没有效果时为空对象
//
// 配置不合法时返回 *InvalidError, 提示可以直接展示; 其他错误为查库失败等内部错误。
func Validate(tx *gorm.DB, category, code string, params map[string]any) (map[string]any, error) {
	if code == "" {
		if len(params) > 0 {
			return nil, &InvalidError{Reason: ParamsWithoutEffect}
		}
		return map[string]any{}, nil
	}
	if category == item.CategoryCollectible {
		return nil, &InvalidError{Reason: EffectNotAllowed}
	}
	def, ok := definitions[code]
	if !ok {
		return nil, &InvalidError{Reason: EffectNotFound}
	}
	decoded, err := def.decode(params, true)
	if err != nil {
		return nil, &InvalidError{Reason: ParamsInvalid}
	}
	if err := settings.ValidateStruct(decoded); err != nil {
		return nil, &InvalidError{Reason: fmt.Sprintf(ParamsRuleInvalid, err.Error())}
	}
	if validator, ok := decoded.(condition.ParamsValidator); ok {
		reason, err := validator.Validate(tx)
		if err != nil {
			return nil, err
		}
		if reason != "" {
			return nil, &InvalidError{Reason: fmt.Sprintf(ParamsRuleInvalid, reason)}
		}
	}
	return normalize(decoded)
}

// normalize 参数结构体转回 map, 数字统一为 float64, 与从数据库读出的一致, 便于比较
func normalize(decoded any) (map[string]any, error) {
	raw, err := json.Marshal(decoded)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Apply 按效果代码执行, 返回使用的结果; 效果不存在或不属于 scene 时返回错误
//
// 参数在保存物品时已校验, 这里只解析, 已删除的参数字段忽略。
func Apply[C, R any](scene, code string, ctx C, params map[string]any) (R, error) {
	var zero R
	def, ok := definitions[code]
	if !ok || def.scene != scene {
		return zero, fmt.Errorf("effect: %s is not an effect of scene %s", code, scene)
	}
	decoded, err := def.decode(params, false)
	if err != nil {
		return zero, fmt.Errorf("effect: decode params of %s failed: %w", code, err)
	}
	result, err := def.apply(ctx, decoded)
	if err != nil {
		return zero, err
	}
	typed, ok := result.(R)
	if !ok {
		return zero, fmt.Errorf("effect: %s returned %T", code, result)
	}
	return typed, nil
}

// MetricUse 按物品统计使用次数的指标, 如 use:12
func MetricUse(itemID uint64) string {
	return metricUsePrefix + strconv.FormatUint(itemID, 10)
}

// Consume 扣除一个物品并累加使用次数, tx 必须是调用方开启的事务
//
// 数量不足时返回 inventory.InsufficientQuantity; 事务提交后 replayed 为 false 时调用 Result.Push 推送给用户。
func Consume(tx *gorm.DB, req *UseRequest) (*reward.Result, bool, error) {
	result, replayed, err := reward.Apply(tx, &reward.ApplyRequest{
		UserID:         req.UserID,
		Bundle:         reward.Bundle{Items: []reward.ItemLine{{ItemID: req.ItemID, Quantity: -1}}},
		BizType:        inventory.BizItemUse,
		BizID:          req.BizID,
		IdempotencyKey: req.IdempotencyKey,
		Remark:         req.Remark,
	})
	if err != nil || replayed {
		return result, replayed, err
	}
	values := stats.Values{MetricItemUsed: 1, MetricUse(req.ItemID): 1}
	if err := stats.Add(tx, req.UserID, req.Scopes, time.Now(), values); err != nil {
		return nil, false, err
	}
	return result, false, nil
}

// Refund 退还用掉的物品, 已下架或已删除的按当前回收价折成货币; 使用次数的统计不扣回
//
// tx 必须是调用方开启的事务; 事务提交后 replayed 为 false 时调用 Result.Push 推送给用户。
func Refund(tx *gorm.DB, req *RefundRequest) (*reward.Result, bool, error) {
	var bundle reward.Bundle
	for _, itemID := range slices.Sorted(maps.Keys(req.Items)) {
		bundle.Items = append(bundle.Items, reward.ItemLine{ItemID: itemID, Quantity: req.Items[itemID]})
	}
	bundle, err := reward.Deliverable(tx, bundle, nil)
	if err != nil {
		return nil, false, err
	}
	return reward.Apply(tx, &reward.ApplyRequest{
		UserID:         req.UserID,
		Bundle:         bundle,
		BizType:        inventory.BizItemUseRefund,
		BizID:          req.BizID,
		IdempotencyKey: req.IdempotencyKey,
		Remark:         req.Remark,
	})
}
