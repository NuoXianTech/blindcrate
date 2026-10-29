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

package shop

const (
	ItemNotFound       = "商品不存在"
	CodeInvalid        = "商品代码需为 1-32 个小写字母、数字或下划线"
	CodeTaken          = "商品代码已存在"
	CodeImmutable      = "商品代码不能修改"
	NameInvalid        = "商品名称需为 1-64 个字符"
	TabInvalid         = "页签最多 16 个字符"
	GoodsEmpty         = "商品至少包含一项货币或物品"
	LinesTooMany       = "价格和商品各自最多 8 项"
	AmountInvalid      = "价格和商品的数量需大于 0"
	CurrencyNotFound   = "价格或商品中的货币不存在"
	ItemsNotFound      = "价格或商品中的物品不存在"
	LimitInvalid       = "限购数量和库存需为 0-1000000"
	TimeInvalid        = "下架时间需晚于上架时间"
	RequirementInvalid = "购买条件: %s"

	OffShelf            = "商品未上架"
	SoldOut             = "商品库存不足"
	QuantityInvalid     = "购买数量需为 1-99"
	PriceChanged        = "商品价格已变动, 请刷新后重试"
	GoodsUnavailable    = "商品或价格中的物品、货币已停用, 暂时无法购买"
	RequirementUnmet    = "未满足购买条件: %s"
	LimitReached        = "%s限购 %d 份, 已购买 %d 份"
	SeasonRequired      = "赛季限购的商品只能在赛季中购买"
	AmountTooLarge      = "购买数量过多, 请减少数量"
	IdempotencyConflict = "幂等键已被其他请求使用"
	DuplicateRequest    = "请求正在处理, 请勿重复提交"

	OrderNotFound      = "订单不存在"
	OrderRefunded      = "订单已退款"
	RefundInsufficient = "用户的货币或物品已不足以收回商品, 无法退款"
	RefundUnavailable  = "价格中的物品或货币已停用, 无法退还"
)
