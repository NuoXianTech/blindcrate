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

package pack

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard/leaderboardadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot/lootadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/shop/shopadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue/venueadmin"
)

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
func contentOf[T any](t *testing.T, section string, items ...T) Content {
	t.Helper()
	data, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	c, err := sectionOf(section).Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// testPack 由各种配置的内容组成配置包
func testPack(t *testing.T, contents map[string]Content) *pack {
	t.Helper()
	sections := make([]string, 0, len(contents))
	for section := range contents {
		sections = append(sections, section)
	}
	ordered, err := orderPackSections(sections)
	if err != nil {
		t.Fatal(err)
	}
	return &pack{manifest: PackManifest{Sections: ordered}, contents: contents}
}

// itemsOf 一种配置解析后的条目
func itemsOf[T any](p *pack, section string) []T {
	return p.contents[section].(*ListContent[T]).Items
}

func manifestJSON(version int, sections ...string) string {
	data, _ := json.Marshal(PackManifest{Format: packFormat, Version: version, Name: " 码头 ", Sections: sections})
	return string(data)
}

func TestReadPack(t *testing.T) {
	data := buildPack(t, map[string]string{
		packManifest:              manifestJSON(1, PackLeaderboards, PackCurrencies),
		"currencies.json":         `[{"code":"chip","name":"筹码","decimals":0,"sort_order":0,"is_enabled":true}]`,
		"leaderboards.json":       `[]`,
		"items.json":              `[{"id":1}]`, // 没有列在 sections 中, 不读取
		"images/item/ab/abcd.png": "png",
	})
	p, err := readPack(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.manifest.Name != "码头" {
		t.Errorf("name = %q, want trimmed", p.manifest.Name)
	}
	if want := []string{PackCurrencies, PackLeaderboards}; !reflect.DeepEqual(p.manifest.Sections, want) {
		t.Errorf("sections = %v, want %v", p.manifest.Sections, want)
	}
	if currencies := itemsOf[packCurrency](p, PackCurrencies); len(currencies) != 1 || currencies[0].Code != "chip" || p.contents[PackItems] != nil {
		t.Errorf("currencies = %+v, contents = %v", currencies, p.contents)
	}
	if p.images["images/item/ab/abcd.png"] == nil || len(p.digest) != 64 {
		t.Errorf("images = %v, digest = %q", p.images, p.digest)
	}
}

func TestReadPackInvalid(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"no manifest", map[string]string{"currencies.json": `[]`}, PackInvalid},
		{"wrong format", map[string]string{packManifest: `{"format":"other","version":1,"name":"a","sections":["settings"]}`}, PackInvalid},
		{"newer version", map[string]string{packManifest: manifestJSON(2, PackSettings)}, fmt.Sprintf(PackVersionUnsupported, 2)},
		{"unknown section", map[string]string{packManifest: manifestJSON(1, "seasons")}, fmt.Sprintf(PackSectionInvalid, "seasons")},
		{"missing file", map[string]string{packManifest: manifestJSON(1, PackItems)}, fmt.Sprintf(PackFileInvalid, "items.json")},
		{"unknown field", map[string]string{packManifest: manifestJSON(1, PackItems), "items.json": `[{"id":1,"price":3}]`}, fmt.Sprintf(PackFileInvalid, "items.json")},
		{"bad image path", map[string]string{packManifest: manifestJSON(1, PackSettings), "settings.json": `{}`, "images/../a.png": "x"}, fmt.Sprintf(PackFileInvalid, "images/../a.png")},
		{"too many", map[string]string{packManifest: manifestJSON(1, PackLeaderboards), "leaderboards.json": "[" + strings.Repeat("{},", maxPackEntries) + "{}]"},
			fmt.Sprintf(PackTooMany, sectionLabel(PackLeaderboards), maxPackEntries)},
	}
	for _, c := range cases {
		_, err := readPack(buildPack(t, c.files))
		var packErr *PackError
		if !errors.As(err, &packErr) || err.Error() != c.want {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
	}
	if _, err := readPack([]byte("not a zip")); err == nil || err.Error() != PackInvalid {
		t.Errorf("not a zip: err = %v", err)
	}
}

func TestOrderPackSections(t *testing.T) {
	got, err := orderPackSections([]string{PackSettings, PackItems, PackItems, PackCurrencies})
	if want := []string{PackCurrencies, PackItems, PackSettings}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("got %v %v, want %v", got, err, want)
	}
	if _, err := orderPackSections(nil); err == nil || err.Error() != PackSectionsRequired {
		t.Errorf("empty: err = %v", err)
	}
}

func TestSnakeCase(t *testing.T) {
	for in, want := range map[string]string{
		"Code":        "code",
		"SellPrice":   "sell_price",
		"ImageURL":    "image_url",
		"ID":          "id",
		"MinQuantity": "min_quantity",
		"URLPath":     "url_path",
	} {
		if got := snakeCase(in); got != want {
			t.Errorf("snakeCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenameScope(t *testing.T) {
	for _, c := range []struct{ scope, want string }{
		{"venue:dock", "venue:pier"},
		{"venue:dock:pve", "venue:pier:pve"},
		{"venue:dockyard", "venue:dockyard"},
		{"mode:ranked", "mode:ranked"},
		{"all", "all"},
	} {
		if got := RenameScope(c.scope, "dock", "pier"); got != c.want {
			t.Errorf("RenameScope(%q) = %q, want %q", c.scope, got, c.want)
		}
	}
}

func TestCompareStatus(t *testing.T) {
	a := leaderboardadmin.LeaderboardRequest{Code: "wins", Name: "胜场"}
	b := a
	if got := CompareStatus(false, a, b); got != PackStatusNew {
		t.Errorf("not found: %s", got)
	}
	if got := CompareStatus(true, a, &b); got != PackStatusSame {
		t.Errorf("same: %s", got)
	}
	b.Size = 50
	if got := CompareStatus(true, a, b); got != PackStatusChanged {
		t.Errorf("changed: %s", got)
	}
	// 条件参数中的字段顺序不影响比较
	x := venueadmin.VenueRequest{Requirements: []condition.Condition{{Type: "stat", Params: json.RawMessage(`{"metric":"wins","scope":"all"}`)}}}
	y := venueadmin.VenueRequest{Requirements: []condition.Condition{{Type: "stat", Params: json.RawMessage(`{"scope": "all", "metric": "wins"}`)}}}
	if got := CompareStatus(true, x, y); got != PackStatusSame {
		t.Errorf("params order: %s", got)
	}
}

func TestChooseActions(t *testing.T) {
	newPack := func() (*Run, []*Entry) {
		p := testPack(t, map[string]Content{
			PackCurrencies: contentOf(t, PackCurrencies, packCurrency{Code: "chip"}),
			PackLootPools:  contentOf(t, PackLootPools, packLootPool{LootPoolRequest: lootadmin.LootPoolRequest{Code: "crate"}}, packLootPool{LootPoolRequest: lootadmin.LootPoolRequest{Code: "box"}}),
			PackVenues: contentOf(t, PackVenues, venueadmin.VenueRequest{
				Code:         "dock",
				Warehouse:    venue.WarehouseRule{Slots: []venue.Slot{{Pool: "crate"}, {Pool: "box"}}},
				Requirements: []condition.Condition{{Type: "stat", Params: json.RawMessage(`{"metric":"wins","scope":"venue:dock:pve"}`)}},
			}),
			PackShopItems:    contentOf(t, PackShopItems, shopadmin.ShopItemRequest{Code: "pass", Requirements: []condition.Condition{{Type: "stat", Params: json.RawMessage(`{"metric":"wins","scope":"venue:dock"}`)}}}),
			PackLeaderboards: contentOf(t, PackLeaderboards, leaderboardadmin.LeaderboardRequest{Code: "wins", Scope: "venue:dock"}),
		})
		r := &Run{pack: p}
		entries, err := r.entries()
		if err != nil {
			t.Fatal(err)
		}
		return r, entries
	}

	r, entries := newPack()
	chosen, err := r.chooseActions(entries, []PackAction{
		{Section: PackLootPools, Key: "crate", Action: PackActionRename, Code: "crate_2"},
		{Section: PackVenues, Key: "dock", Action: PackActionRename, Code: "pier"},
		{Section: PackCurrencies, Key: "chip", Action: PackActionSkip},
		{Section: PackItems, Key: "9", Action: PackActionSkip}, // 包中没有, 忽略
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(chosen) != 3 {
		t.Errorf("chosen = %d, want 3", len(chosen))
	}
	v := itemsOf[venueadmin.VenueRequest](r.pack, PackVenues)[0]
	if v.Code != "pier" || v.Warehouse.Slots[0].Pool != "crate_2" || v.Warehouse.Slots[1].Pool != "box" {
		t.Errorf("venue = %+v", v)
	}
	if !strings.Contains(string(v.Requirements[0].Params), `"venue:pier:pve"`) {
		t.Errorf("requirement params = %s", v.Requirements[0].Params)
	}
	if board := itemsOf[leaderboardadmin.LeaderboardRequest](r.pack, PackLeaderboards)[0]; board.Scope != "venue:pier" {
		t.Errorf("leaderboard scope = %s", board.Scope)
	}
	if item := itemsOf[shopadmin.ShopItemRequest](r.pack, PackShopItems)[0]; !strings.Contains(string(item.Requirements[0].Params), `"venue:pier"`) {
		t.Errorf("shop item params = %s", item.Requirements[0].Params)
	}

	for _, c := range []struct {
		name   string
		action PackAction
		want   string
	}{
		{"bad action", PackAction{Section: PackCurrencies, Key: "chip", Action: "merge"}, PackActionInvalid},
		{"not renamable", PackAction{Section: PackCurrencies, Key: "chip", Action: PackActionRename, Code: "gold"}, fmt.Sprintf(PackRenameNotAllowed, sectionLabel(PackCurrencies))},
		{"code in pack", PackAction{Section: PackLootPools, Key: "crate", Action: PackActionRename, Code: "box"}, fmt.Sprintf(PackRenameInvalid, sectionLabel(PackLootPools), "crate")},
		{"empty code", PackAction{Section: PackLootPools, Key: "crate", Action: PackActionRename}, fmt.Sprintf(PackRenameInvalid, sectionLabel(PackLootPools), "crate")},
	} {
		r, entries := newPack()
		if _, err := r.chooseActions(entries, []PackAction{c.action}); err == nil || err.Error() != c.want {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
	}
}

func TestPackEntriesDuplicate(t *testing.T) {
	r := &Run{pack: testPack(t, map[string]Content{PackVenues: contentOf(t, PackVenues, venueadmin.VenueRequest{Code: "dock"}, venueadmin.VenueRequest{Code: "dock"})})}
	if _, err := r.entries(); err == nil || err.Error() != fmt.Sprintf(PackKeyDuplicate, sectionLabel(PackVenues), "dock") {
		t.Errorf("err = %v", err)
	}
}

func TestPackFailure(t *testing.T) {
	if !packFailure(errors.New(PackItemDeleted)) || !packFailure(&venue.InvalidError{}) || !packFailure(packErrorf(PackImageMissing, "a")) {
		t.Error("validation errors should be failures")
	}
	if packFailure(fmt.Errorf("query: %w", errors.New("conn reset"))) {
		t.Error("wrapped errors should abort")
	}
}

func TestSections(t *testing.T) {
	var codes []string
	for _, s := range Sections() {
		codes = append(codes, s.Code)
	}
	want := []string{PackCurrencies, PackQualities, PackItems, PackLootPools, PackVenues, PackShopItems, PackLeaderboards, PackSettings}
	if !reflect.DeepEqual(codes, want) {
		t.Errorf("sections = %v, want %v", codes, want)
	}
	for _, s := range []Section{
		{Code: PackItems, Label: "物品", Export: sectionOf(PackItems).Export, Decode: sectionOf(PackItems).Decode},
		{Code: "Bad-Code", Label: "x", Export: sectionOf(PackItems).Export, Decode: sectionOf(PackItems).Decode},
		{Code: "missions"},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("RegisterSection(%q) should panic", s.Code)
				}
			}()
			RegisterSection(s)
		}()
	}
}
