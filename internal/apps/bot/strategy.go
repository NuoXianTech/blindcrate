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

package bot

import (
	"fmt"
	"maps"
	"math/rand/v2"

	"github.com/NuoXianTech/blindcrate/internal/apps/params"
	"gorm.io/gorm"
)

// strategy 一种决策策略, 由 RegisterStrategy 生成
type strategy struct {
	label       string
	description string
	schema      *params.Schema // 缺少的参数取默认值
	defaults    map[string]any
	decide      func(obs, decoded any, rng *rand.Rand) (any, error)
}

var (
	// strategies 已注册的策略, 只在启动前写入
	strategies = map[string]*strategy{}
	// strategyOrder 策略的注册顺序, 后台按这个顺序展示
	strategyOrder []string
)

// RegisterStrategy 注册一种决策策略, 需要在启动前调用; 代码为空、过长、重复, 参数结构体有不支持的字段或默认参数不合法时 panic
//
// O 为玩法提供的观察, D 为决策, 由玩法定义, 同一玩法的策略使用相同的 O 和 D; 观察只能包含同座位的真人看得到的信息。
// P 为参数结构体, 字段的 tag 见 params 包, P 的指针可以实现 params.Validator;
// defaults 为新建风格时的默认参数, 风格中缺少的参数(如策略之后新增的参数)也取默认值。
// decide 不能读写数据库, 随机数只从 rng 取, 相同的观察和随机数得到相同的决策。
func RegisterStrategy[O, P, D any](code, label, description string, defaults P, decide func(obs O, decoded *P, rng *rand.Rand) D) {
	if code == "" || len(code) > maxStrategyCodeLength {
		panic(fmt.Sprintf("bot: invalid strategy code %q", code))
	}
	if _, ok := strategies[code]; ok {
		panic(fmt.Sprintf("bot: strategy %s already registered", code))
	}
	schema := params.New(code, params.Plain, &defaults)
	if err := params.ValidateStruct(&defaults); err != nil {
		panic(fmt.Sprintf("bot: invalid defaults of %s: %v", code, err))
	}
	normalized, err := params.Normalize(&defaults)
	if err != nil {
		panic(fmt.Sprintf("bot: encode defaults of %s failed: %v", code, err))
	}
	strategies[code] = &strategy{
		label:       label,
		description: description,
		schema:      schema,
		defaults:    normalized,
		decide: func(obs, decoded any, rng *rand.Rand) (any, error) {
			o, ok := obs.(O)
			if !ok {
				return nil, fmt.Errorf("bot: strategy %s got observation %T", code, obs)
			}
			return decide(o, decoded.(*P), rng), nil
		},
	}
	strategyOrder = append(strategyOrder, code)
}

// Strategies 全部决策策略, 按注册顺序
func Strategies() []StrategyView {
	views := make([]StrategyView, 0, len(strategyOrder))
	for _, code := range strategyOrder {
		def := strategies[code]
		views = append(views, StrategyView{
			Code:        code,
			Label:       def.label,
			Description: def.description,
			Fields:      def.schema.Fields(),
			Defaults:    maps.Clone(def.defaults),
		})
	}
	return views
}

// validateParams 保存风格前校验策略和参数, 返回规范化后的参数: 按参数结构体重新编码, 缺少的参数取默认值
func validateParams(tx *gorm.DB, code string, values map[string]any) (map[string]any, error) {
	def, ok := strategies[code]
	if !ok {
		return nil, &InvalidError{Reason: StrategyNotFound}
	}
	decoded, problem, err := def.schema.ValidateMap(tx, values)
	switch {
	case err != nil:
		return nil, err
	case problem == nil:
		return params.Normalize(decoded)
	case problem.Malformed:
		return nil, &InvalidError{Reason: ParamsInvalid}
	}
	return nil, &InvalidError{Reason: fmt.Sprintf(ParamsRuleInvalid, problem.Reason)}
}

// Decide 按风格快照中的策略和参数做出决策; 策略没有注册或观察、决策的类型与策略不符时返回错误
//
// 参数在保存风格时已校验, 这里只解析: 缺少的参数取默认值, 已删除的参数字段忽略。
func Decide[O, D any](s *Snapshot, obs O, rng *rand.Rand) (D, error) {
	decide, err := Prepare[O, D](s)
	if err != nil {
		var zero D
		return zero, err
	}
	return decide(obs, rng)
}

// Decider 参数已解析的决策, 见 Prepare
type Decider[O, D any] func(obs O, rng *rand.Rand) (D, error)

// Prepare 解析风格快照中的参数, 返回可以反复调用的决策, 用于模拟大量对局; 策略没有注册或参数解析失败时返回错误
//
// 与 Decide 相同, 观察、决策的类型与策略不符时决策返回错误。可以并发调用, 策略不能修改参数。
func Prepare[O, D any](s *Snapshot) (Decider[O, D], error) {
	code := s.Strategy
	def, ok := strategies[code]
	if !ok {
		return nil, fmt.Errorf("bot: strategy %s not registered", code)
	}
	decoded, err := def.schema.DecodeMap(s.Params, false)
	if err != nil {
		return nil, fmt.Errorf("bot: decode params of %s failed: %w", code, err)
	}
	return func(obs O, rng *rand.Rand) (D, error) {
		var zero D
		result, err := def.decide(obs, decoded, rng)
		if err != nil {
			return zero, err
		}
		typed, ok := result.(D)
		if !ok {
			return zero, fmt.Errorf("bot: strategy %s returned %T", code, result)
		}
		return typed, nil
	}, nil
}
