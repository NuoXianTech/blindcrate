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

package crate

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/effect"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
)

// TestOpenEffect 开启箱子的效果属于背包场景, 保存物品时只校验掉落池代码的格式
func TestOpenEffect(t *testing.T) {
	if scene := effect.SceneOf(EffectOpen); scene != effect.SceneInventory {
		t.Errorf("scene = %q, want %q", scene, effect.SceneInventory)
	}
	if item.FindCategory(Category) == nil {
		t.Errorf("category %s not registered", Category)
	}

	got, err := effect.Validate(nil, Category, EffectOpen, map[string]any{"pool": "crate_basic"})
	if err != nil || !reflect.DeepEqual(got, map[string]any{"pool": "crate_basic"}) {
		t.Errorf("valid: got %v %v", got, err)
	}

	cases := []struct {
		name     string
		category string
		params   map[string]any
		want     string
	}{
		{"collectible", item.CategoryCollectible, map[string]any{"pool": "crate_basic"}, effect.EffectNotAllowed},
		{"upper case", Category, map[string]any{"pool": "Crate"}, fmt.Sprintf(effect.ParamsRuleInvalid, PoolInvalid)},
		{"hyphen", Category, map[string]any{"pool": "crate-basic"}, fmt.Sprintf(effect.ParamsRuleInvalid, PoolInvalid)},
		{"unknown field", Category, map[string]any{"pool": "crate_basic", "times": 2}, effect.ParamsInvalid},
		{"wrong type", Category, map[string]any{"pool": 1}, effect.ParamsInvalid},
	}
	for _, c := range cases {
		if _, err := effect.Validate(nil, c.category, EffectOpen, c.params); err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
	for name, params := range map[string]map[string]any{
		"missing":  {},
		"empty":    {"pool": ""},
		"too long": {"pool": strings.Repeat("a", 65)},
	} {
		if _, err := effect.Validate(nil, Category, EffectOpen, params); err == nil {
			t.Errorf("%s: got nil", name)
		}
	}
}

func TestRenamePool(t *testing.T) {
	cases := []struct {
		name   string
		effect string
		params map[string]any
		want   map[string]any
	}{
		{"renamed", EffectOpen, map[string]any{"pool": "old"}, map[string]any{"pool": "new"}},
		{"other pool", EffectOpen, map[string]any{"pool": "other"}, map[string]any{"pool": "other"}},
		{"other effect", "reveal_name", map[string]any{"pool": "old"}, map[string]any{"pool": "old"}},
		{"no effect", "", nil, nil},
	}
	for _, c := range cases {
		RenamePool(c.effect, c.params, "old", "new")
		if !reflect.DeepEqual(c.params, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, c.params, c.want)
		}
	}
}

// TestPick 每次只抽中一项, 数量在这一项的范围内
func TestPick(t *testing.T) {
	entries := []loot.EntryDetail{
		{Entry: loot.Entry{ItemID: 7, Weight: 1, MinQuantity: 1, MaxQuantity: 1}},
		{Entry: loot.Entry{ItemID: 8, Weight: 5, MinQuantity: 2, MaxQuantity: 4}},
		{Entry: loot.Entry{Currency: "chip", Weight: 3, MinQuantity: 100, MaxQuantity: 200}},
	}
	seen := map[string]bool{}
	for range 500 {
		drop := loot.Draw(entries).Bundle()
		switch {
		case len(drop.Items) == 1 && len(drop.Currencies) == 0:
			line := drop.Items[0]
			seen[fmt.Sprintf("item:%d", line.ItemID)] = true
			if (line.ItemID == 7 && line.Quantity != 1) || (line.ItemID == 8 && (line.Quantity < 2 || line.Quantity > 4)) || (line.ItemID != 7 && line.ItemID != 8) {
				t.Fatalf("unexpected item drop %+v", line)
			}
		case len(drop.Items) == 0 && len(drop.Currencies) == 1:
			line := drop.Currencies[0]
			seen[line.Currency] = true
			if line.Currency != "chip" || line.Amount < 100 || line.Amount > 200 {
				t.Fatalf("unexpected currency drop %+v", line)
			}
		default:
			t.Fatalf("drop has %d items and %d currencies, want exactly one line", len(drop.Items), len(drop.Currencies))
		}
	}
	// 每一项的概率都不低于 1/9, 500 次都没抽中的概率可以忽略
	if len(seen) != 3 {
		t.Errorf("seen = %v, want all 3 entries", seen)
	}

	only := []loot.EntryDetail{{Entry: loot.Entry{Currency: "chip", Weight: 1, MinQuantity: 5, MaxQuantity: 5}}}
	want := reward.Bundle{Currencies: []reward.CurrencyLine{{Currency: "chip", Amount: 5}}}
	if got := loot.Draw(only).Bundle(); !reflect.DeepEqual(got, want) {
		t.Errorf("single entry: got %+v, want %+v", got, want)
	}
}
