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

package match

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"gorm.io/gorm"
)

// Phase 对局的一个阶段
type Phase struct {
	Name     string        // 阶段名称, 由玩法定义, 如 bidding, 最长 32
	Round    int32         // 第几轮, 没有轮次时为 0
	Duration time.Duration // 阶段时长, 到期后调用 Mode.OnTimeout; 0 表示不限时, 只能由玩家操作推进
	Finish   bool          // 对局结束, 为 true 时忽略其他字段
}

// Mode 一种对战玩法
//
// 钩子都在锁定对局的事务内调用, 可以写入玩法自己的表和调用 reward.Apply, 返回错误时整个事务回滚;
// 钩子不要修改 m 和 players, 对局状态由返回的 Phase 决定, 名次用 SetRanks 写入。
type Mode interface {
	// Code 玩法代码, 匹配时按它选择玩法, 最长 32
	Code() string
	// PlayerCount 每局的玩家数, 包括人机
	PlayerCount() int
	// MinHumans 开局至少需要的真人数, 1 到 PlayerCount; 等于 PlayerCount 时不用人机
	MinHumans() int
	// BotFillAfter 真人不够一局时, 排队至少 MinHumans 人且最早加入的人等满这个时间后用人机补齐, 0 表示立即补齐
	BotFillAfter() time.Duration
	// Admit 开局前为每个真人调用, 用于检查门槛和扣入场费, 在创建对局的事务中, 此时还没有座位
	//
	// 返回 Reject 的错误时该玩家不参加这一局, 其写入被撤销, 推送 MessageMatchmakingRejected 并移出队列;
	// 剩下的真人不够 MinHumans 时不开局, 剩下的人回到队列中。返回其他错误时这一批排队的玩家都被移出队列。
	Admit(tx *gorm.DB, m *Match, userID uint64) error
	// Start 对局创建后初始化, 返回第一个阶段, 不能为 nil; 返回错误时这一批排队的玩家都被移出队列
	Start(tx *gorm.DB, m *Match, players []Player) (*Phase, error)
	// OnTimeout 当前阶段到期, 返回下一个阶段, 不能为 nil
	OnTimeout(tx *gorm.DB, m *Match, players []Player) (*Phase, error)
	// OnCancel 管理员取消对局, 用于退还入场费等
	OnCancel(tx *gorm.DB, m *Match, players []Player) error
}

// JoinChecker 玩法可以实现的接口, 加入队列前检查门槛, 如余额是否足够; 开局时仍需在 Admit 中检查
type JoinChecker interface {
	// CheckJoin 返回 Reject 的错误时拒绝加入, 原因返回给用户
	CheckJoin(ctx context.Context, userID uint64) error
}

// RejectedError 玩家不满足玩法的条件, Reason 展示给玩家
type RejectedError struct {
	Reason string
}

func (e *RejectedError) Error() string { return e.Reason }

// Reject 在 Admit 和 CheckJoin 中拒绝玩家, reason 展示给玩家
func Reject(reason string) error {
	return &RejectedError{Reason: reason}
}

// modes 已注册的玩法, 只在启动前写入
var modes = map[string]Mode{}

// Register 注册一种玩法, 需要在启动前调用; 代码为空或过长、人数不合法或重复注册时 panic
func Register(mode Mode) {
	code := mode.Code()
	if code == "" || len(code) > 32 {
		panic(fmt.Sprintf("match: invalid mode code %q", code))
	}
	if n := mode.PlayerCount(); n < 1 || n > maxPlayers {
		panic(fmt.Sprintf("match: invalid player count %d of mode %s", n, code))
	}
	if n := mode.MinHumans(); n < 1 || n > mode.PlayerCount() {
		panic(fmt.Sprintf("match: invalid min humans %d of mode %s", n, code))
	}
	if mode.BotFillAfter() < 0 {
		panic(fmt.Sprintf("match: invalid bot fill delay of mode %s", code))
	}
	if _, ok := modes[code]; ok {
		panic(fmt.Sprintf("match: mode %s already registered", code))
	}
	modes[code] = mode
}

// modeCodes 已注册的玩法代码, 按代码排序
func modeCodes() []string {
	return slices.Sorted(maps.Keys(modes))
}
