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

// Package bot 人机: 昵称库、风格和决策策略
//
//   - 昵称: 开局时从启用的昵称中随机选取, 同一局不重复; 不够时用 "玩家" 加四位数字补齐
//   - 策略: 由玩法模块用 RegisterStrategy 注册, 决定人机每轮怎么行动; 策略只能看到同座位的真人看得到的信息
//   - 风格: 后台配置的一种策略和它的参数, 如稳健、激进; 地图按模式配置出场的风格和权重,
//     开局时为每个人机选一种风格并快照, 之后修改风格不影响进行中的对局
package bot

// 配置包中的人机配置, 由 botadmin 注册; 地图按代码引用风格, 风格改名导入时由 pack 同步修改地图
const (
	PackPersonas = "bot_personas"
	PackNames    = "bot_names"
)

// 操作日志
const (
	TargetName    = "bot_name"    // target_id 为昵称 ID
	TargetPersona = "bot_persona" // target_id 为风格 ID

	ActionNameCreate    = "admin.bot_name_create"
	ActionNameUpdate    = "admin.bot_name_update"
	ActionNameDelete    = "admin.bot_name_delete"
	ActionPersonaCreate = "admin.bot_persona_create"
	ActionPersonaUpdate = "admin.bot_persona_update"
	ActionPersonaDelete = "admin.bot_persona_delete"
)

// 昵称和风格的限制
const (
	maxNameLength         = 32 // 与用户昵称和 match_players.bot_name 一致
	MaxBatchNames         = 200
	maxPersonaNameLength  = 32
	maxDescriptionLength  = 255
	maxStrategyCodeLength = 32
	// fallbackNamePrefix 昵称库不够时补齐的昵称前缀, 后接四位数字
	fallbackNamePrefix = "玩家"
)
