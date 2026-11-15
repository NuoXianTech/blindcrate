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

package inventory

import (
	"reflect"
	"testing"
	"time"
)

func TestGroupRecords(t *testing.T) {
	won := time.Date(2026, 11, 6, 8, 0, 0, 0, time.UTC)
	used := won.Add(time.Minute)
	// 一局拍下三种藏品记为一批, 之后单独用掉一个道具
	rows := []Transaction{
		{ID: 10, ItemID: 1, ItemName: "金怀表", Quantity: 1, QuantityAfter: 1, BizType: BizMatchReward, BizID: "42", Remark: "码头仓库·人机", CreatedAt: won},
		{ID: 11, BatchID: 10, ItemID: 2, ItemName: "银勺", Quantity: 2, QuantityAfter: 5, BizType: BizMatchReward, BizID: "42", Remark: "码头仓库·人机", CreatedAt: won},
		{ID: 12, BatchID: 10, ItemID: 3, ItemName: "古董花瓶", Quantity: 1, QuantityAfter: 1, BizType: BizMatchReward, BizID: "42", Remark: "码头仓库·人机", CreatedAt: won},
		{ID: 13, ItemID: 9, ItemName: "侦察卡", Quantity: -1, QuantityAfter: 0, BizType: BizItemUse, BizID: "43", CreatedAt: used},
	}
	got := groupRecords([]uint64{13, 10}, rows)
	want := []Record{
		{ID: 13, BizType: BizItemUse, BizID: "43", CreatedAt: used, Items: []RecordLine{
			{ItemID: 9, ItemName: "侦察卡", Quantity: -1, QuantityAfter: 0},
		}},
		{ID: 10, BizType: BizMatchReward, BizID: "42", Remark: "码头仓库·人机", CreatedAt: won, Items: []RecordLine{
			{ItemID: 1, ItemName: "金怀表", Quantity: 1, QuantityAfter: 1},
			{ItemID: 2, ItemName: "银勺", Quantity: 2, QuantityAfter: 5},
			{ItemID: 3, ItemName: "古董花瓶", Quantity: 1, QuantityAfter: 1},
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("batches: got %+v, want %+v", got, want)
	}

	// 按物品筛选时只剩批次中的一行, 记录仍以批次中第一行的 ID 标识
	got = groupRecords([]uint64{10}, rows[1:2])
	want = []Record{{ID: 10, BizType: BizMatchReward, BizID: "42", Remark: "码头仓库·人机", CreatedAt: won, Items: []RecordLine{
		{ItemID: 2, ItemName: "银勺", Quantity: 2, QuantityAfter: 5},
	}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("filtered: got %+v, want %+v", got, want)
	}
}
