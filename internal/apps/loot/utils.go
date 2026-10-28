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

package loot

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
)

// PickIndex 按权重随机选出一个下标, 选中下标 i 的概率为 weights[i] / 权重之和
//
// 权重需全部大于 0 且之和不溢出, 否则 panic。使用 crypto/rand, 结果无法被预测。
func PickIndex(weights []int64) int {
	var total int64
	for _, weight := range weights {
		if weight <= 0 || total > math.MaxInt64-weight {
			panic(fmt.Sprintf("loot: invalid weights %v", weights))
		}
		total += weight
	}
	if total == 0 {
		panic("loot: empty weights")
	}

	n := RandInt64N(total)
	for i, weight := range weights {
		if n < weight {
			return i
		}
		n -= weight
	}
	panic("loot: unreachable")
}

// RandInt64N 返回 [0, n) 内均匀分布的随机数, n 需大于 0, 否则 panic; 使用 crypto/rand
func RandInt64N(n int64) int64 {
	if n <= 0 {
		panic(fmt.Sprintf("loot: invalid n %d", n))
	}
	value, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		panic(fmt.Sprintf("loot: read random failed: %v", err))
	}
	return value.Int64()
}
