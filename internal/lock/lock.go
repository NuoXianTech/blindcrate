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

package lock

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/redis/go-redis/v9"
)

// Lock 已获取的锁, 只能由获取它的请求释放
type Lock struct {
	key   string
	token string // 获取时生成的随机值, 释放和续期时校验, 避免误删其他持有者的锁
}

// releaseScript 值等于 token 时删除
var releaseScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('DEL', KEYS[1])
end
return 0
`)

// refreshScript 值等于 token 时重设过期时间, ARGV[2] 为毫秒
var refreshScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return 0
`)

// TryAcquire 尝试获取锁, 已被持有时立即返回 Busy
//
// ttl 至少为 1 毫秒, 否则 panic。
func TryAcquire(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
	if ttl < time.Millisecond {
		panic(fmt.Sprintf("lock: invalid ttl %v", ttl))
	}
	l := &Lock{key: keyPrefix + key, token: rand.Text()}
	ok, err := db.Redis.SetNX(ctx, l.key, l.token, ttl).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New(Busy)
	}
	return l, nil
}

// Acquire 获取锁, 已被持有时每隔 retryInterval 重试, 最多等待 wait, 仍未获取时返回 Busy
//
// ctx 取消时返回 ctx 的错误。
func Acquire(ctx context.Context, key string, ttl, wait time.Duration) (*Lock, error) {
	deadline := time.Now().Add(wait)
	for {
		l, err := TryAcquire(ctx, key, ttl)
		if err == nil || err.Error() != Busy || !time.Now().Before(deadline) {
			return l, err
		}
		timer := time.NewTimer(min(retryInterval, time.Until(deadline)))
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// Release 释放锁, 锁已过期或已被其他请求重新获取时什么都不做
func (l *Lock) Release(ctx context.Context) error {
	return releaseScript.Run(ctx, db.Redis, []string{l.key}, l.token).Err()
}

// Refresh 把锁的过期时间重设为 ttl, 锁已过期或已被其他请求重新获取时返回 NotHeld
func (l *Lock) Refresh(ctx context.Context, ttl time.Duration) error {
	if ttl < time.Millisecond {
		panic(fmt.Sprintf("lock: invalid ttl %v", ttl))
	}
	ok, err := refreshScript.Run(ctx, db.Redis, []string{l.key}, l.token, ttl.Milliseconds()).Int()
	if err != nil {
		return err
	}
	if ok == 0 {
		return errors.New(NotHeld)
	}
	return nil
}

// Do 持有锁执行 fn, 最多等待 wait, 执行完后释放
//
// 获取不到时返回 Busy, 不执行 fn; fn 的耗时应明显小于 ttl。
// 释放失败只记录日志, 锁会在 ttl 后自动过期。
func Do(ctx context.Context, key string, ttl, wait time.Duration, fn func(ctx context.Context) error) error {
	l, err := Acquire(ctx, key, ttl, wait)
	if err != nil {
		return err
	}
	defer func() {
		// 请求已取消时也要释放, 否则其他请求要等到锁过期
		if releaseErr := l.Release(context.WithoutCancel(ctx)); releaseErr != nil {
			logger.ErrorF(ctx, "[Lock] release %s failed: %v", key, releaseErr)
		}
	}()
	return fn(ctx)
}
