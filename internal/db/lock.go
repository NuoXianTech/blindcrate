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

package db

import "gorm.io/gorm"

// lockPrefix 咨询锁的 key 的前缀, 与同一个数据库中的其他应用区分
const lockPrefix = "blindcrate:"

// Lock 在事务 tx 中取得名为 name 的咨询锁, 事务结束时自动释放; 同名的并发事务在这里排队
//
// name 用模块加业务键, 如 "shop:12"(同一用户的购买)、"wallet:<幂等键>"。锁按 name 的 32 位哈希取, 不同的 name 有很小的概率共用一把锁,
// 只会多排队, 不影响正确性。tx 必须是调用方开启的事务。
func Lock(tx *gorm.DB, name string) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockPrefix+name).Error
}
