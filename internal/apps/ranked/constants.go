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

// Package ranked 段位分: 排位按名次加减的分数, 每个赛季分开计算
//
//   - 赛季: 进行中的赛季按赛季 ID 记分, 赛季之外的排位记在赛季 0, 见 seasonIDOf
//   - 初始分: 第一次在赛季中结算排位时创建, 有上一个赛季的段位分时按赛季的继承比例向基础分靠拢, 见 initialRating
//   - 加减: 成交的排位局按名次加减, 人机占名次但不计分, 流拍不计分; 扣到 0 为止, 见 Settle
//   - 条件: 注册条件类型 min_rating, 地图可以按段位分限制入场
package ranked

// TypeMinRating 条件类型: 当前赛季的段位分不少于
const TypeMinRating = "min_rating"
