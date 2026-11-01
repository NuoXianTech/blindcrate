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

// Package codex 图鉴: 玩家收集过的藏品, 以及集齐后可以领取奖励的收集组
//
//   - 收集: 直接读取背包的 inventory_items, 第一次获得一种物品时插入一行, 数量扣到 0 也保留, 不另外记录;
//     物品改为藏品后, 以前获得过它的玩家也算已收集
//   - 范围: 只算启用的藏品; 下架的藏品和道具不进图鉴, 收集组中这些物品也不计入进度,
//     下架期间集齐其余藏品的玩家即可领取, 重新上架后已领取的不受影响
//   - 领取: 集齐收集组中全部启用的藏品后领取, 每个收集组只能领取一次, 见 Claim
package codex

// BizReward 领取收集组奖励的流水业务类型, biz_id 为收集组 ID
const BizReward = "codex_reward"

// 操作日志
const (
	TargetSet = "codex_set" // target_id 为收集组 ID

	ActionCreate = "admin.codex_set_create"
	ActionUpdate = "admin.codex_set_update"
	ActionDelete = "admin.codex_set_delete"
)

// 收集组的限制
const (
	maxNameLength        = 64
	maxDescriptionLength = 255
	MaxSetItems          = 200 // 一个收集组最多的物品数
)
