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

package ranked

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/redis/go-redis/v9"
)

// afkScript 记一次挂机, 时间窗口内的次数达到阈值时禁赛, 已在禁赛中且剩余时间更长时不缩短
//
// KEYS[1] 挂机记录, KEYS[2] 禁赛; ARGV[1] 对局 ID, ARGV[2] 当前毫秒时间戳, ARGV[3] 时间窗口的毫秒数, ARGV[4] 阈值, 0 表示不禁赛,
// 之后为依次的禁赛毫秒数。同一局重复记录只算一次。返回 {窗口内的次数, 本次禁赛的毫秒数, 没有禁赛为 0}。
var afkScript = redis.NewScript(`
local now = tonumber(ARGV[2])
local window = tonumber(ARGV[3])
local threshold = tonumber(ARGV[4])
redis.call('ZADD', KEYS[1], now, ARGV[1])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now - window)
redis.call('PEXPIRE', KEYS[1], window)
local count = redis.call('ZCARD', KEYS[1])
if threshold == 0 or count < threshold or #ARGV < 5 then
	return {count, 0}
end
local ban = tonumber(ARGV[math.min(5 + count - threshold, #ARGV)])
if redis.call('PTTL', KEYS[2]) < ban then
	redis.call('SET', KEYS[2], 1, 'PX', ban)
end
return {count, ban}
`)

// RecordAfk 记录用户在一局排位中挂机, 24 小时内的次数达到设置时禁止排位; 在对局结算的事务提交后调用
//
// 返回本次的禁赛时长, 没有禁赛为 0。
func RecordAfk(ctx context.Context, userID, matchID uint64) (time.Duration, error) {
	cfg, err := settings.Ranked.Get(ctx)
	if err != nil {
		return 0, err
	}
	args := []interface{}{matchID, time.Now().UnixMilli(), afkWindow.Milliseconds(), cfg.AfkThreshold}
	for _, seconds := range cfg.AfkBans {
		args = append(args, (time.Duration(seconds) * time.Second).Milliseconds())
	}
	key := strconv.FormatUint(userID, 10)
	result, err := afkScript.Run(ctx, db.Redis, []string{afkKeyPrefix + key, afkBanKeyPrefix + key}, args...).Int64Slice()
	if err != nil {
		return 0, err
	}
	return time.Duration(result[1]) * time.Millisecond, nil
}

// AfkBanReason 用户因挂机禁赛中时返回带剩余时间的原因, 可以直接展示; 没有禁赛时返回空字符串
func AfkBanReason(ctx context.Context, userID uint64) (string, error) {
	remaining, err := db.Redis.PTTL(ctx, afkBanKeyPrefix+strconv.FormatUint(userID, 10)).Result()
	if err != nil {
		return "", err
	}
	// 没有禁赛时为 -2, 没有过期时间时为 -1, 禁赛总会带过期时间
	if remaining <= 0 {
		return "", nil
	}
	return fmt.Sprintf(AfkBanned, formatRemaining(remaining)), nil
}

// formatRemaining 剩余时间, 不足一分钟按一分钟, 如 "5 分钟"、"1 小时 30 分钟"
func formatRemaining(d time.Duration) string {
	minutes := int64((d + time.Minute - 1) / time.Minute)
	switch {
	case minutes < 60:
		return fmt.Sprintf("%d 分钟", minutes)
	case minutes%60 == 0:
		return fmt.Sprintf("%d 小时", minutes/60)
	default:
		return fmt.Sprintf("%d 小时 %d 分钟", minutes/60, minutes%60)
	}
}
