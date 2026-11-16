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

// Package crate 箱子: 可以在背包中开启的物品, 开启一次从掉落池中按概率抽取一项货币或物品
//
//   - 箱子: 使用效果为 EffectOpen 的物品, 效果参数指定掉落池, 见 OpenParams; 类别 Category 只用于分类, 能不能开启只看使用效果
//   - 内容: 掉落池中当前可抽取的条目, 公示的概率与开启时一致
//   - 开启: 走背包场景通用的使用流程, 见 effect.UseInInventory, 这里只给出可以开出什么和怎么抽
//
// 结果在接口返回前已经结算, 客户端的开箱动画只负责揭晓, 不影响结果。
package crate

// Category 物品类别: 箱子, 由本模块注册
const Category = "crate"

// EffectOpen 使用效果: 开启箱子, 属于背包场景, 参数见 OpenParams
const EffectOpen = "open_crate"

// BizDrop 流水业务类型: 开启箱子得到的货币和物品, biz_id 为使用记录 ID; 扣除箱子记为 inventory.BizItemUse, biz_id 相同
const BizDrop = "crate_drop"

// paramPool 效果参数中掉落池代码的 key, 与 OpenParams 的 json tag 一致
const paramPool = "pool"

// PoolInvalid 掉落池代码格式不对的提示
const PoolInvalid = "掉落池代码需为 1-64 个小写字母、数字或下划线"
