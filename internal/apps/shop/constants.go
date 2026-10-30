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

// Package shop 商城: 管理员配置的商品, 玩家用货币或物品购买, 是货币的主要回收口
//
//   - 商品: 一次买一组货币和物品, 价格也可以是任意货币和物品, 见 Item
//   - 购买: 一个事务内检查上架时间、库存、购买条件和限购, 扣价格、发商品、写订单, 见 Purchase
//   - 限购: 每人累计、每天、每赛季的购买数量记在统计中, 指标为 shop_buy:<code>, 见 Metric
//   - 退款: 管理员退还价格并收回商品, 恢复库存, 不恢复限购数量, 见 Refund
package shop

// 订单的状态
const (
	StatusPaid     = "paid"     // 已支付, 商品已发放
	StatusRefunded = "refunded" // 已退款
)

// 限购的类型
const (
	LimitTotal  = "total"  // 每人累计
	LimitDaily  = "daily"  // 每天
	LimitSeason = "season" // 每赛季
)

// metricPrefix 限购计数指标的前缀, 见 Metric
const metricPrefix = "shop_buy:"

// 商品配置的限制
const (
	maxTabLength = 16        // 页签的最大字符数
	MaxQuantity  = 99        // 一次最多购买的份数
	maxLimit     = 1_000_000 // 限购数量和库存的上限
)
