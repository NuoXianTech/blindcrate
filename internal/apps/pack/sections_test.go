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

package pack_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/crate"
	"github.com/NuoXianTech/blindcrate/internal/apps/item/itemadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard/leaderboardadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot/lootadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"github.com/NuoXianTech/blindcrate/internal/apps/shop/shopadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue/venueadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet/walletadmin"
	"gorm.io/gorm"
)

// 这里的测试用到各模块注册的配置种类, 所以放在外部测试包中; 不需要其他模块的测试在 service_test.go

// buildPack 按文件名和内容打包 zip
func buildPack(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// contentOf 把配置按包中的格式编码后用注册的 Decode 解析
func contentOf[T any](t *testing.T, section string, items ...T) pack.Content {
	t.Helper()
	data, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	c, err := pack.DecodeSection(section, data)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// newRun 由各种配置的内容组成配置包
func newRun(t *testing.T, contents map[string]pack.Content) *pack.Run {
	t.Helper()
	r, err := pack.NewRun(contents)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// itemsOf 一种配置解析后的条目
func itemsOf[T any](c pack.Content) []T {
	return c.(*pack.ListContent[T]).Items
}

func manifestJSON(version int, sections ...string) string {
	data, _ := json.Marshal(pack.PackManifest{Format: pack.PackFormat, Version: version, Name: " 码头 ", Sections: sections})
	return string(data)
}

func TestReadPack(t *testing.T) {
	data := buildPack(t, map[string]string{
		pack.PackManifestFile:     manifestJSON(1, pack.PackLeaderboards, pack.PackCurrencies),
		"currencies.json":         `[{"code":"chip","name":"筹码","decimals":0,"sort_order":0,"is_enabled":true}]`,
		"leaderboards.json":       `[]`,
		"items.json":              `[{"id":1}]`, // 没有列在 sections 中, 不读取
		"images/item/ab/abcd.png": "png",
	})
	p, err := pack.ReadPack(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.Manifest().Name != "码头" {
		t.Errorf("name = %q, want trimmed", p.Manifest().Name)
	}
	if want := []string{pack.PackCurrencies, pack.PackLeaderboards}; !reflect.DeepEqual(p.Manifest().Sections, want) {
		t.Errorf("sections = %v, want %v", p.Manifest().Sections, want)
	}
	if currencies := itemsOf[walletadmin.CurrencyPack](p.Content(pack.PackCurrencies)); len(currencies) != 1 || currencies[0].Code != "chip" || p.Content(pack.PackItems) != nil {
		t.Errorf("currencies = %+v, items = %v", currencies, p.Content(pack.PackItems))
	}
	if !p.HasImage("images/item/ab/abcd.png") || len(p.Digest()) != 64 {
		t.Errorf("image missing or digest = %q", p.Digest())
	}
}

func TestReadPackInvalid(t *testing.T) {
	manifest := pack.PackManifestFile
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"no manifest", map[string]string{"currencies.json": `[]`}, pack.PackInvalid},
		{"wrong format", map[string]string{manifest: `{"format":"other","version":1,"name":"a","sections":["settings"]}`}, pack.PackInvalid},
		{"newer version", map[string]string{manifest: manifestJSON(pack.PackVersion+1, pack.PackSettings)}, fmt.Sprintf(pack.PackVersionUnsupported, pack.PackVersion+1)},
		{"unknown section", map[string]string{manifest: manifestJSON(1, "no_such_section")}, fmt.Sprintf(pack.PackSectionInvalid, "no_such_section")},
		{"missing file", map[string]string{manifest: manifestJSON(1, pack.PackItems)}, fmt.Sprintf(pack.PackFileInvalid, "items.json")},
		{"unknown field", map[string]string{manifest: manifestJSON(1, pack.PackItems), "items.json": `[{"id":1,"price":3}]`}, fmt.Sprintf(pack.PackFileInvalid, "items.json")},
		{"bad image path", map[string]string{manifest: manifestJSON(1, pack.PackSettings), "settings.json": `{}`, "images/../a.png": "x"}, fmt.Sprintf(pack.PackFileInvalid, "images/../a.png")},
		{"too many", map[string]string{manifest: manifestJSON(1, pack.PackLeaderboards), "leaderboards.json": "[" + strings.Repeat("{},", pack.MaxPackEntries) + "{}]"},
			fmt.Sprintf(pack.PackTooMany, pack.SectionLabel(pack.PackLeaderboards), pack.MaxPackEntries)},
	}
	for _, c := range cases {
		_, err := pack.ReadPack(buildPack(t, c.files))
		var packErr *pack.PackError
		if !errors.As(err, &packErr) || err.Error() != c.want {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
	}
	if _, err := pack.ReadPack([]byte("not a zip")); err == nil || err.Error() != pack.PackInvalid {
		t.Errorf("not a zip: err = %v", err)
	}
}

func TestOrderPackSections(t *testing.T) {
	got, err := pack.OrderPackSections([]string{pack.PackSettings, pack.PackItems, pack.PackItems, pack.PackCurrencies})
	if want := []string{pack.PackCurrencies, pack.PackSettings, pack.PackItems}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("got %v %v, want %v", got, err, want)
	}
	if _, err := pack.OrderPackSections(nil); err == nil || err.Error() != pack.PackSectionsRequired {
		t.Errorf("empty: err = %v", err)
	}
}

func TestCompareStatus(t *testing.T) {
	a := leaderboardadmin.LeaderboardRequest{Code: "wins", Name: "胜场"}
	b := a
	if got := pack.CompareStatus(false, a, b); got != pack.PackStatusNew {
		t.Errorf("not found: %s", got)
	}
	if got := pack.CompareStatus(true, a, &b); got != pack.PackStatusSame {
		t.Errorf("same: %s", got)
	}
	b.Size = 50
	if got := pack.CompareStatus(true, a, b); got != pack.PackStatusChanged {
		t.Errorf("changed: %s", got)
	}
	// 条件参数中的字段顺序不影响比较
	x := venueadmin.VenueRequest{Requirements: []condition.Condition{{Type: "stat", Params: json.RawMessage(`{"metric":"wins","scope":"all"}`)}}}
	y := venueadmin.VenueRequest{Requirements: []condition.Condition{{Type: "stat", Params: json.RawMessage(`{"scope": "all", "metric": "wins"}`)}}}
	if got := pack.CompareStatus(true, x, y); got != pack.PackStatusSame {
		t.Errorf("params order: %s", got)
	}
}

func TestChooseActions(t *testing.T) {
	newPack := func() (*pack.Run, []*pack.Entry) {
		r := newRun(t, map[string]pack.Content{
			pack.PackCurrencies: contentOf(t, pack.PackCurrencies, walletadmin.CurrencyPack{Code: "chip"}),
			pack.PackItems: contentOf(t, pack.PackItems,
				itemadmin.ItemPack{ID: 3, ItemRequest: itemadmin.ItemRequest{Effect: crate.EffectOpen, EffectParams: map[string]any{"pool": "crate"}}},
				itemadmin.ItemPack{ID: 4, ItemRequest: itemadmin.ItemRequest{Effect: crate.EffectOpen, EffectParams: map[string]any{"pool": "box"}}},
			),
			pack.PackLootPools: contentOf(t, pack.PackLootPools,
				lootadmin.PoolPack{LootPoolRequest: lootadmin.LootPoolRequest{Code: "crate"}}, lootadmin.PoolPack{LootPoolRequest: lootadmin.LootPoolRequest{Code: "box"}}),
			pack.PackVenues: contentOf(t, pack.PackVenues, venueadmin.VenueRequest{
				Code:         "dock",
				Lots:         venue.LotRule{Tiers: []venue.Tier{{Depot: venue.DepotRule{Slots: []venue.Slot{{Pool: "crate"}, {Pool: "box"}}}}}},
				Requirements: []condition.Condition{{Type: "stat", Params: json.RawMessage(`{"metric":"wins","scope":"venue:dock:pve"}`)}},
			}),
			pack.PackShopItems: contentOf(t, pack.PackShopItems,
				shopadmin.ShopItemRequest{Code: "pass", Requirements: []condition.Condition{{Type: "stat", Params: json.RawMessage(`{"metric":"wins","scope":"venue:dock"}`)}}}),
			pack.PackLeaderboards: contentOf(t, pack.PackLeaderboards, leaderboardadmin.LeaderboardRequest{Code: "wins", Scope: "venue:dock"}),
		})
		entries, err := r.Entries()
		if err != nil {
			t.Fatal(err)
		}
		return r, entries
	}

	r, entries := newPack()
	chosen, err := r.ChooseActions(entries, []pack.PackAction{
		{Section: pack.PackLootPools, Key: "crate", Action: pack.PackActionRename, Code: "crate_2"},
		{Section: pack.PackVenues, Key: "dock", Action: pack.PackActionRename, Code: "pier"},
		{Section: pack.PackCurrencies, Key: "chip", Action: pack.PackActionSkip},
		{Section: pack.PackItems, Key: "9", Action: pack.PackActionSkip}, // 包中没有, 忽略
	})
	if err != nil {
		t.Fatal(err)
	}
	if chosen != 3 {
		t.Errorf("chosen = %d, want 3", chosen)
	}
	v := itemsOf[venueadmin.VenueRequest](r.Content(pack.PackVenues))[0]
	if v.Code != "pier" || v.Lots.Tiers[0].Depot.Slots[0].Pool != "crate_2" || v.Lots.Tiers[0].Depot.Slots[1].Pool != "box" {
		t.Errorf("venue = %+v", v)
	}
	if !strings.Contains(string(v.Requirements[0].Params), `"venue:pier:pve"`) {
		t.Errorf("requirement params = %s", v.Requirements[0].Params)
	}
	// 箱子的掉落池跟着掉落池一起改名
	if crates := itemsOf[itemadmin.ItemPack](r.Content(pack.PackItems)); crates[0].EffectParams["pool"] != "crate_2" || crates[1].EffectParams["pool"] != "box" {
		t.Errorf("crate pools = %v, %v", crates[0].EffectParams, crates[1].EffectParams)
	}
	if pools := itemsOf[lootadmin.PoolPack](r.Content(pack.PackLootPools)); pools[0].Code != "crate_2" || pools[1].Code != "box" {
		t.Errorf("pools = %s, %s", pools[0].Code, pools[1].Code)
	}
	if board := itemsOf[leaderboardadmin.LeaderboardRequest](r.Content(pack.PackLeaderboards))[0]; board.Scope != "venue:pier" {
		t.Errorf("leaderboard scope = %s", board.Scope)
	}
	if item := itemsOf[shopadmin.ShopItemRequest](r.Content(pack.PackShopItems))[0]; !strings.Contains(string(item.Requirements[0].Params), `"venue:pier"`) {
		t.Errorf("shop item params = %s", item.Requirements[0].Params)
	}

	for _, c := range []struct {
		name   string
		action pack.PackAction
		want   string
	}{
		{"bad action", pack.PackAction{Section: pack.PackCurrencies, Key: "chip", Action: "merge"}, pack.PackActionInvalid},
		{"not renamable", pack.PackAction{Section: pack.PackCurrencies, Key: "chip", Action: pack.PackActionRename, Code: "gold"},
			fmt.Sprintf(pack.PackRenameNotAllowed, pack.SectionLabel(pack.PackCurrencies))},
		{"code in pack", pack.PackAction{Section: pack.PackLootPools, Key: "crate", Action: pack.PackActionRename, Code: "box"},
			fmt.Sprintf(pack.PackRenameInvalid, pack.SectionLabel(pack.PackLootPools), "crate")},
		{"empty code", pack.PackAction{Section: pack.PackLootPools, Key: "crate", Action: pack.PackActionRename},
			fmt.Sprintf(pack.PackRenameInvalid, pack.SectionLabel(pack.PackLootPools), "crate")},
	} {
		r, entries := newPack()
		if _, err := r.ChooseActions(entries, []pack.PackAction{c.action}); err == nil || err.Error() != c.want {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
	}
}

func TestPackEntriesDuplicate(t *testing.T) {
	r := newRun(t, map[string]pack.Content{
		pack.PackVenues: contentOf(t, pack.PackVenues, venueadmin.VenueRequest{Code: "dock"}, venueadmin.VenueRequest{Code: "dock"}),
	})
	if _, err := r.Entries(); err == nil || err.Error() != fmt.Sprintf(pack.PackKeyDuplicate, pack.SectionLabel(pack.PackVenues), "dock") {
		t.Errorf("err = %v", err)
	}
}

func TestPackFailure(t *testing.T) {
	if !pack.PackFailure(errors.New(pack.PackConflicts)) || !pack.PackFailure(&venue.InvalidError{}) || !pack.PackFailure(pack.PackErrorf(pack.PackImageMissing, "a")) {
		t.Error("validation errors should be failures")
	}
	if pack.PackFailure(fmt.Errorf("query: %w", errors.New("conn reset"))) {
		t.Error("wrapped errors should abort")
	}
}

func TestSections(t *testing.T) {
	// 被其他配置引用的种类按依赖顺序排列, 其他模块注册的种类排在它们之间
	want := []string{pack.PackCurrencies, pack.PackSettings, pack.PackQualities, pack.PackItems, pack.PackLootPools, pack.PackVenues, pack.PackShopItems, pack.PackLeaderboards}
	var codes []string
	renamable := map[string]bool{}
	for _, s := range pack.Sections() {
		if slices.Contains(want, s.Code) {
			codes = append(codes, s.Code)
		}
		renamable[s.Code] = s.Renamable
	}
	if !reflect.DeepEqual(codes, want) {
		t.Errorf("sections = %v, want %v", codes, want)
	}
	// 能不能改名由注册时有没有 Rename 决定: 物品按 ID 被引用, 货币和品质的代码被直接引用, 都不能改名
	for code, wantRenamable := range map[string]bool{
		pack.PackCurrencies: false, pack.PackSettings: false, pack.PackQualities: false, pack.PackItems: false,
		pack.PackLootPools: true, pack.PackVenues: true, pack.PackShopItems: true, pack.PackLeaderboards: true,
	} {
		if renamable[code] != wantRenamable {
			t.Errorf("%s renamable = %v, want %v", code, renamable[code], wantRenamable)
		}
	}

	export := func(context.Context, *gorm.DB, func(string) string) (any, int, error) { return nil, 0, nil }
	decode := func([]byte) (pack.Content, error) { return nil, nil }
	for _, s := range []pack.Section{
		{Code: pack.PackItems, Label: "物品", Export: export, Decode: decode},
		{Code: "Bad-Code", Label: "x", Export: export, Decode: decode},
		{Code: "no_functions"},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("RegisterSection(%q) should panic", s.Code)
				}
			}()
			pack.RegisterSection(s)
		}()
	}
}

// TestRegisterTable 缺少必要的函数时不能注册
func TestRegisterTable(t *testing.T) {
	type row struct {
		Code string `json:"code"`
	}
	full := func() pack.Table[row, row] {
		return pack.Table[row, row]{
			Code: "test_rows", Label: "测试",
			Query: func(tx *gorm.DB) *gorm.DB { return tx },
			Pack:  func(r *row, _ func(string) string) row { return *r },
			Key:   func(r *row) string { return r.Code },
			Name:  func(r *row) string { return r.Code },
			Find:  func(*gorm.DB, *row, *row) (bool, error) { return false, nil },
			Save:  func(*gorm.DB, *pack.Run, *row, *row) error { return nil },
		}
	}
	for name, strip := range map[string]func(t *pack.Table[row, row]){
		"no query": func(t *pack.Table[row, row]) { t.Query = nil },
		"no pack":  func(t *pack.Table[row, row]) { t.Pack = nil },
		"no key":   func(t *pack.Table[row, row]) { t.Key = nil },
		"no name":  func(t *pack.Table[row, row]) { t.Name = nil },
		"no find":  func(t *pack.Table[row, row]) { t.Find = nil },
		"no save":  func(t *pack.Table[row, row]) { t.Save = nil },
		// 只有 Export 时没法把已有的一行转为包中的内容
		"export without current": func(t *pack.Table[row, row]) {
			t.Query, t.Pack = nil, nil
			t.Export = func(*gorm.DB, func(string) string) ([]row, error) { return nil, nil }
		},
	} {
		table := full()
		strip(&table)
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: RegisterTable should panic", name)
				}
			}()
			pack.RegisterTable(table)
		}()
	}

	// 完整的表注册后按 JSON 数组解析, 有 Rename 才能改名, 改名时引用它的项一起修改
	table := full()
	table.Rename = func(r *row, code string) { r.Code = code }
	table.Renamed = func(r *row, section, old, next string) {
		if section == "test_rows" && r.Code == "ref:"+old {
			r.Code = "ref:" + next
		}
	}
	pack.RegisterTable(table)
	content := contentOf(t, "test_rows", row{Code: "a"}, row{Code: "ref:a"})
	r := newRun(t, map[string]pack.Content{"test_rows": content})
	entries, err := r.Entries()
	if err != nil || len(entries) != 2 || entries[0].Key != "a" || entries[0].Rename == nil {
		t.Fatalf("entries = %v, %v", entries, err)
	}
	if _, err := r.ChooseActions(entries, []pack.PackAction{{Section: "test_rows", Key: "a", Action: pack.PackActionRename, Code: "b"}}); err != nil {
		t.Fatal(err)
	}
	if rows := itemsOf[row](content); rows[0].Code != "b" || rows[1].Code != "ref:b" {
		t.Errorf("rows = %+v", rows)
	}
}
