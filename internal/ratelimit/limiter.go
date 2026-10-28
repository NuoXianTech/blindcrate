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

package ratelimit

import (
	"context"
	"fmt"
	"net/netip"
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/redis/go-redis/v9"
)

// Rule 限流规则, 平均每 Period 内不超过 Limit 次
type Rule struct {
	Limit  int           // 允许连续操作的次数
	Period time.Duration // 用完后完全恢复所需的时间
}

// PerSecond 每秒 n 次
func PerSecond(n int) Rule {
	return Rule{Limit: n, Period: time.Second}
}

// PerMinute 每分钟 n 次
func PerMinute(n int) Rule {
	return Rule{Limit: n, Period: time.Minute}
}

// PerHour 每小时 n 次
func PerHour(n int) Rule {
	return Rule{Limit: n, Period: time.Hour}
}

// check 规则不合法时 panic: Limit 需要大于 0, 恢复一次的间隔 Period/Limit 至少为 1 微秒
func (r Rule) check() {
	if r.Limit <= 0 || r.Period.Microseconds() < int64(r.Limit) {
		panic(fmt.Sprintf("ratelimit: invalid rule %+v", r))
	}
}

// allowScript GCRA 限流, 允许时计入一次并返回 0, 否则返回还需等待的微秒数
//
// key 中保存理论到达时间 tat, 每次操作把 tat 推后 interval, 推后后超出当前时间 period 以上则拒绝, 拒绝的操作不计入。
// KEYS[1]: 限流 key; ARGV[1]: interval, 恢复一次所需的微秒数; ARGV[2]: period 的微秒数
var allowScript = redis.NewScript(`
local interval = tonumber(ARGV[1])
local period = tonumber(ARGV[2])
local clock = redis.call('TIME')
local now = tonumber(clock[1]) * 1000000 + tonumber(clock[2])
local tat = math.max(tonumber(redis.call('GET', KEYS[1])) or now, now) + interval
if tat - now > period then
	return tat - now - period
end
redis.call('SET', KEYS[1], tat, 'PX', math.ceil((tat - now) / 1000))
return 0
`)

// Allow 判断 key 是否还能再操作一次, 能则计入一次; 不能时返回 false 和还需等待的时间
//
// Redis 出错时放行并记录日志; 规则不合法时 panic。
func Allow(ctx context.Context, key string, rule Rule) (bool, time.Duration) {
	rule.check()
	period := rule.Period.Microseconds()
	wait, err := allowScript.Run(ctx, db.Redis, []string{keyPrefix + key}, period/int64(rule.Limit), period).Int64()
	if err != nil {
		logger.ErrorF(ctx, "[RateLimit] allow %s failed: %v", key, err)
		return true, 0
	}
	if wait > 0 {
		return false, time.Duration(wait) * time.Microsecond
	}
	return true, 0
}

// IPKey 按 IP 限流的 key, IPv6 地址按 /64 网段计算, 一个用户通常拥有整个网段
func IPKey(scope, ip string) string {
	if addr, err := netip.ParseAddr(ip); err == nil {
		addr = addr.Unmap()
		if addr.Is6() {
			ip = netip.PrefixFrom(addr, 64).Masked().String()
		} else {
			ip = addr.String()
		}
	}
	return scope + ":ip:" + ip
}

// UserKey 按用户限流的 key
func UserKey(scope string, userID uint64) string {
	return scope + ":user:" + strconv.FormatUint(userID, 10)
}
