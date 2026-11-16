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

const (
	// MaxEntries 每个掉落池最多的条目数
	MaxEntries = 200
	// MaxRollTimes 一次最多抽取的次数
	MaxRollTimes = 100
)

// 操作日志的对象类型, 见 audit.Record
const (
	TargetLootPool = "loot_pool"
)

// 操作日志的操作类型, 格式为 "模块.动作", 在 init 中登记
const (
	ActionLootPoolCreate        = "admin.loot_pool_create"
	ActionLootPoolUpdate        = "admin.loot_pool_update"
	ActionLootPoolDelete        = "admin.loot_pool_delete"
	ActionLootPoolEntriesUpdate = "admin.loot_pool_entries_update"
)

// SourcePool 可选值来源: 全部掉落池, 值为掉落池代码; 参数字段用 input:"pool" 引用, 见 params.RegisterSource
const SourcePool = "pool"
