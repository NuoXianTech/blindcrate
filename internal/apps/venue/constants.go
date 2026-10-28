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

// Package venue 地图(拍卖场): 管理员在后台配置的仓库掉落、入场条件和代价、拍卖参数、人机参数
//
// 每张地图对应两种玩法 pve:<代码> 和 ranked:<代码>, 由拍卖模块提供; 开局时快照地图配置, 之后修改地图不影响进行中的对局。
//   - 保存: 先调用 Validate 校验配置, 它会模拟 SimulationRuns 个仓库, 摆放失败率过高时拒绝, 并把价值分布写入 Simulation
//   - 开局: NewGenerator 读取掉落池和物品, Generate 抽取藏品并摆进格子
//   - 展示: 玩家看到启用的地图、入场条件的进度和期望价值区间, 详情中附带各档位掉落池的公示概率
//
// 比例都按基点(万分之一)存储, 如速胜倍率 20000 表示 2 倍。
package venue

// 模式, 拼进玩法代码和统计范围
const (
	ModePve    = "pve"    // 人机模式, 1 个真人和 3 个人机
	ModeRanked = "ranked" // 排位模式, 至少 2 个真人, 不够时用人机补齐
)

const (
	// SimulationRuns 保存地图时模拟的仓库数
	SimulationRuns = 10000
	// maxFailureRate 模拟时允许的摆放失败率, 基点
	maxFailureRate = 100
	// maxEntryCostLines 入场代价最多的行数
	maxEntryCostLines = 8
)

// BasisPoints 基点的分母, 10000 表示 100%
const BasisPoints = 10000
