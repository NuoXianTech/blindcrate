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

package reward

import (
	"context"
	"errors"

	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"gorm.io/gorm"
)

// Failure 一份奖励因为内容本身而发不出的原因, 见 FailureOf
type Failure int

const (
	// Delivered 不是奖励内容的问题: 没有出错, 或者是余额不足、查库失败等其他错误
	Delivered Failure = iota
	// Unavailable 奖励中的货币已停用或不存在, 物品已下架或已删除, 或者折算后什么也发不出; 配置改好后可以再发
	Unavailable
	// TooLarge 合并同类项后超出单笔变动的上限
	TooLarge
)

// errNothing 折算后什么也发不出, 见 Prepare
var errNothing = errors.New("reward: nothing to deliver")

// FailureOf 判断 Apply、Prepare 或 Deliver 返回的错误是不是奖励内容的问题, 各模块按它换成自己的提示
func FailureOf(err error) Failure {
	if err == nil {
		return Delivered
	}
	if errors.Is(err, errNothing) {
		return Unavailable
	}
	switch err.Error() {
	case item.ItemNotFound, item.ItemDisabled, wallet.CurrencyNotFound, wallet.CurrencyDisabled:
		return Unavailable
	case wallet.AmountInvalid, inventory.QuantityInvalid:
		return TooLarge
	}
	return Delivered
}

// Prepare 把配置中的一份奖励换成现在实际要发放的内容: 已下架或已删除的物品按回收价折成货币(见 Deliverable), 再合并同类项
//
// 全是已下架且没有回收价的物品时什么也发不出, 返回的错误 FailureOf 为 Unavailable; 合并后超出上限时为 TooLarge。
// 需要在发放前记下实际内容时调用, 否则直接用 Deliver。
func Prepare(tx *gorm.DB, bundle Bundle) (Bundle, error) {
	bundle, err := Deliverable(tx, bundle, nil)
	if err != nil {
		return Bundle{}, err
	}
	if bundle, err = bundle.Normalize(); err != nil {
		return Bundle{}, err
	}
	if bundle.IsEmpty() {
		return Bundle{}, errNothing
	}
	return bundle, nil
}

// Delivery 一次发放的结果, 见 Deliver
type Delivery struct {
	Bundle   Bundle // 实际发放的内容, 已下架的物品已折成货币
	replayed bool
	result   *Result
}

// Deliver 发放一份配置好的奖励(任务、签到、邮件附件、兑换码等), req.Bundle 为配置中的内容, 发放前先经过 Prepare
//
// tx 必须是调用方开启的事务, 返回错误时调用方应回滚整个事务; 领取资格和防重复领取由调用方在同一个事务中保证,
// 幂等键只是兜底: 重试时已下架的物品会折成货币, 内容变了, 幂等键拦不住。
// 奖励发不出时返回的错误用 FailureOf 判断, 此时不应记为已领取, 配置改好后还可以再领。
// 事务提交后调用 Delivery.Push 推送给用户。
func Deliver(tx *gorm.DB, req *ApplyRequest) (*Delivery, error) {
	bundle, err := Prepare(tx, req.Bundle)
	if err != nil {
		return nil, err
	}
	prepared := *req
	prepared.Bundle = bundle
	result, replayed, err := Apply(tx, &prepared)
	if err != nil {
		return nil, err
	}
	return &Delivery{Bundle: result.Bundle, replayed: replayed, result: result}, nil
}

// Replayed 这次发放是否命中了之前的流水, 没有产生新的变动
func (d *Delivery) Replayed() bool { return d.replayed }

// Push 把变动后的余额和物品数量推送给用户, 需要在事务提交后调用; 重放时不推送, 推送失败只记录日志
func (d *Delivery) Push(ctx context.Context) {
	if !d.replayed {
		d.result.Push(ctx)
	}
}

// Received 实际发放的内容, 带名称便于展示
func (d *Delivery) Received(tx *gorm.DB) ([]Line, error) {
	lines, err := Lines(tx, d.Bundle)
	if err != nil {
		return nil, err
	}
	return lines[0], nil
}
