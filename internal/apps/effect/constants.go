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

// Package effect 物品的使用效果: 带效果的物品可以在效果所属的场景中使用, 用一次扣一个
//
// 效果类型由代码注册, 见 Register; 每种效果属于一个场景, 参数由管理员在物品上配置, 后台按参数结构体生成表单:
//   - 物品: items.effect 为效果代码, items.effect_params 为参数; 藏品不能带效果, 其他类别的物品配置效果后就可以使用
//   - 场景: 效果在哪里使用, 由场景所在的模块决定什么时候能用、限制次数、提供上下文并记录结果, 见 Scene 开头的常量
//   - 使用: 场景在自己的事务中先用 Consume 扣除一个并累加统计, 再用 Apply 按效果代码执行; 需要退还时用 Refund
//
// 新增一种效果只需在场景所在的模块注册; 新增一种场景需要在这里加常量, 并在场景模块中实现使用流程。
package effect

// CategoryConsumable 物品类别: 道具, 主要用于带效果的消耗品, 由本模块注册
const CategoryConsumable = "consumable"

// 场景
const (
	SceneMatch = "match" // 暗拍的出价阶段, 上下文和结果由 auction 定义
)

// MetricItemUsed 统计指标: 使用物品的次数, 每用一个累加 1; 按物品统计的指标见 MetricUse
const MetricItemUsed = "item_used"

// metricUsePrefix 按物品统计使用次数的指标前缀, 见 MetricUse
const metricUsePrefix = "use:"

// maxCodeLength 效果代码的最大长度, 与 items.effect 一致
const maxCodeLength = 32
