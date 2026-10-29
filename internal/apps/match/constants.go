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

// Package match 多人对战的骨架: 对局、阶段推进、阶段超时、匹配和人机补位, 具体玩法通过 Mode 接入
//
// 新增一种玩法:
//   - 实现 Mode, 在玩法包的 init 中调用 Register, API 和 worker 进程都需要导入该包; 同一种玩法的不同场次注册为不同的代码
//   - 场次存在数据库中(如每张地图一种玩法)时, 改为调用 RegisterProvider 按代码前缀提供玩法, 后台增删场次不需要重启
//   - 排队凑满 MaxHumans 个真人时立即开局, 不够 PlayerCount 的座位由人机补齐; 只要求 MinHumans 个真人的玩法, 等满 BotFillAfter 后由人机补齐
//   - 对局由匹配创建, 先为每个真人调用 Mode.Admit 检查门槛和扣入场费, 再调用 Mode.Start 进入第一个阶段; 阶段到期时 worker 调用 Mode.OnTimeout
//   - 玩家操作(如出价)在玩法自己的 ws 处理函数中通过 Advance 写入, 需要提前结束阶段时返回下一个阶段
//   - 玩法自己的数据(出价、手牌等)放在玩法自己的表中, 按 match_id 关联; 入场费和奖励在钩子中调用 reward.Apply
//
// 阶段切换后推送 MessageUpdated 给对局中的真人玩家; 玩家看到的对局不区分真人和人机。
// 玩法可以在玩家结算后释放对局(Phase.Release), 之后的阶段只用于展示, 玩家可以离开并加入新的队列。
// 客户端断线重连后调用 GET /matches/current 恢复当前对局。
package match

import "time"

// 对局状态
const (
	StatusPlaying   = "playing"   // 进行中
	StatusFinished  = "finished"  // 正常结束
	StatusCancelled = "cancelled" // 被管理员取消
)

// 推送给客户端的消息类型
const (
	MessageUpdated             = "match:updated"              // 对局开始、切换阶段或结束, data 为 MatchView
	MessageMatchmakingFailed   = "match:matchmaking_failed"   // 创建对局失败, 已移出队列, data 为 MatchmakingFailedMessage
	MessageMatchmakingRejected = "match:matchmaking_rejected" // 开局时不满足玩法的条件, 已移出队列, data 为 MatchmakingRejectedMessage
	MessageQueueUpdated        = "match:queue_updated"        // 队列中的人数变化, 推送给仍在队列中的人, data 为 QueueUpdatedMessage
)

// 人机等级, 等级越高利用的信息越多, 具体含义由玩法决定
const (
	BotLevelLow    int16 = 1
	BotLevelMedium int16 = 2
	BotLevelHigh   int16 = 3
)

const (
	// maxPlayers 每局最多的玩家数
	maxPlayers = 16
	// timeoutGrace 阶段到期超过这个时间仍在进行中时, 巡检重新投递超时任务
	timeoutGrace = 10 * time.Second
	// sweepBatch 每次巡检最多重新投递的超时任务数
	sweepBatch = 100
	// sweepScanCount 巡检扫描队列时每次 SCAN 的数量提示
	sweepScanCount = 100
)

// Redis key
const (
	// queueKeyPrefix 每种玩法一个有序集合, 成员为用户 ID, 分数为加入时间的毫秒时间戳
	queueKeyPrefix = "blindcrate:matchmaking:queue:"
	// queueUsersKey 哈希, 用户 ID -> 正在排队的玩法, 保证同一用户同时只在一个队列中
	queueUsersKey = "blindcrate:matchmaking:users"
)

// botNames 人机的昵称, 每局随机选取且不重复
var botNames = []string{
	"橘子汽水", "晚风", "不吃香菜", "摸鱼达人", "小熊软糖", "夜猫子",
	"一只咸鱼", "清风徐来", "芝士就是力量", "拾光者", "奶茶续命", "长安",
	"今天也要加油", "星河", "柠檬不萌", "南城", "白开水", "鹿鸣",
	"慢半拍", "云朵", "抹茶控", "北岛", "早睡早起", "小满",
}
