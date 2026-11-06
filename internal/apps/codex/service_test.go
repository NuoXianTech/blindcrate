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

package codex

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// TestValidate 只校验不需要查库的部分
func TestValidate(t *testing.T) {
	valid := func() Set {
		return Set{Code: "dock_all", Name: "码头全收集", ItemIDs: []uint64{3, 1, 2}}
	}
	tooMany := make([]uint64, MaxSetItems+1)
	for i := range tooMany {
		tooMany[i] = uint64(i + 1)
	}
	cases := []struct {
		name   string
		modify func(s *Set)
		want   string
	}{
		{"code", func(s *Set) { s.Code = "Dock" }, CodeInvalid},
		{"name", func(s *Set) { s.Name = "" }, NameInvalid},
		{"description", func(s *Set) { s.Description = strings.Repeat("说", maxDescriptionLength+1) }, DescriptionInvalid},
		{"items empty", func(s *Set) { s.ItemIDs = nil }, ItemsEmpty},
		{"items too many", func(s *Set) { s.ItemIDs = tooMany }, ItemsTooMany},
	}
	for _, c := range cases {
		s := valid()
		c.modify(&s)
		if err := Validate(nil, &s, nil); err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}

func TestNewEntry(t *testing.T) {
	row := entryRow{ID: 7, Name: "金怀表", Description: "表盖内刻着名字", Quality: 4, ImageURL: "/a.png", SellCurrency: "chip", SellPrice: 1500}
	// 没有收集时只返回品质和图片
	if got := newEntry(row); got != (EntryView{ItemID: 7, Quality: 4, ImageURL: "/a.png"}) {
		t.Errorf("missing: got %+v", got)
	}
	// 卖出后数量为 0, 仍算已收集
	firstAt := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	row.FirstAt = &firstAt
	want := EntryView{
		ItemID: 7, Name: "金怀表", Description: "表盖内刻着名字", Quality: 4, ImageURL: "/a.png",
		SellCurrency: "chip", SellPrice: 1500, Collected: true, FirstAt: &firstAt,
	}
	if got := newEntry(row); got != want {
		t.Errorf("collected: got %+v", got)
	}
}

func TestMembers(t *testing.T) {
	entries := []EntryView{
		{ItemID: 5, Collected: true},
		{ItemID: 2},
		{ItemID: 9, Collected: true},
		{ItemID: 1},
	}
	// 按图鉴中的顺序排列, 不在启用的藏品中的物品(如 4)不计入
	ids, collected := members(entries, []uint64{1, 4, 5, 9})
	if !slices.Equal(ids, []uint64{5, 9, 1}) || collected != 2 {
		t.Errorf("got %v %d", ids, collected)
	}
	ids, collected = members(entries, []uint64{4})
	if ids == nil || len(ids) != 0 || collected != 0 {
		t.Errorf("none: got %v %d", ids, collected)
	}
}
