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

package redeemadmin

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/redeem"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
)

func TestBatchChanges(t *testing.T) {
	expiresAt := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	requirements := func(params string) []condition.Condition {
		return []condition.Condition{{Type: condition.TypeMinStat, Params: json.RawMessage(params)}}
	}
	old := redeem.Batch{
		Name:   "开服礼包",
		Reward: reward.Bundle{Currencies: []reward.CurrencyLine{{Currency: "chip", Amount: 100}}},
		// 从数据库读出的参数, 键按 JSONB 的顺序排列, 冒号和逗号后有空格
		Requirements: requirements(`{"count": 3, "scope": "all", "metric": "matches"}`),
		MaxUses:      1,
		UserLimit:    1,
		ExpiresAt:    &expiresAt,
		IsEnabled:    true,
	}

	// 请求中的参数按表单的顺序排列, 没有空格; 到期时间是同一时刻的另一个时区
	next := old
	next.Requirements = requirements(`{"metric":"matches","scope":"all","count":3}`)
	local := expiresAt.In(time.FixedZone("UTC+8", 8*3600))
	next.ExpiresAt = &local
	if changes := batchChanges(&old, &next); len(changes) != 0 {
		t.Errorf("same content: got %v", changes)
	}

	next.Requirements = requirements(`{"metric":"matches","scope":"all","count":5}`)
	next.MaxUses = 0
	next.ExpiresAt = nil
	changed := slices.Sorted(maps.Keys(batchChanges(&old, &next)))
	if want := []string{"expires_at", "max_uses", "requirements"}; !slices.Equal(changed, want) {
		t.Errorf("changed: got %v, want %v", changed, want)
	}
}

func TestTruncateMinute(t *testing.T) {
	if got := truncateMinute(nil); got != nil {
		t.Errorf("nil: got %v", got)
	}
	at := time.Date(2026, 12, 1, 8, 30, 45, 123, time.UTC)
	if got := truncateMinute(&at); got == nil || !got.Equal(time.Date(2026, 12, 1, 8, 30, 0, 0, time.UTC)) {
		t.Errorf("got %v", got)
	}
}
