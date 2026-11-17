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
	"fmt"
	"testing"
)

func TestDropFinale(t *testing.T) {
	files := map[string][]byte{
		PackSettings: []byte(`{"balance":{"finale_enabled":true,"pool_rate":7500},"economy":{"sell_fee_rate":200}}`),
		PackVenues:   []byte(`[{"code":"dock","auction":{"rounds":5,"finale":true}},{"code":"old"}]`),
	}
	if err := dropFinale(files); err != nil {
		t.Fatal(err)
	}
	if got, want := string(files[PackSettings]), `{"balance":{"pool_rate":7500},"economy":{"sell_fee_rate":200}}`; got != want {
		t.Errorf("settings = %s, want %s", got, want)
	}
	if got, want := string(files[PackVenues]), `[{"auction":{"rounds":5},"code":"dock"},{"code":"old"}]`; got != want {
		t.Errorf("venues = %s, want %s", got, want)
	}
	if err := dropFinale(map[string][]byte{PackVenues: []byte(`{}`)}); err == nil || err.Error() != fmt.Sprintf(PackFileInvalid, "venues.json") {
		t.Errorf("invalid venues: err = %v", err)
	}
}

func TestDropPool(t *testing.T) {
	files := map[string][]byte{
		PackSettings: []byte(`{"balance":{"pool_enabled":true,"entry_rate":300,"flat_entry_rate":200,"flat_cv":6000,"pool_rate":7500,"rebate_rate":4000,"rebate_cap_rate":10000,"min_seat_ev":-400}}`),
		PackVenues:   []byte(`[{"code":"dock","auction":{"entry_fee":78,"pool_rate":7500,"rebate_rate":4000,"rebate_cap":78}}]`),
	}
	if err := dropPool(files); err != nil {
		t.Fatal(err)
	}
	if got, want := string(files[PackSettings]), `{"balance":{"min_seat_ev":-400}}`; got != want {
		t.Errorf("settings = %s, want %s", got, want)
	}
	if got, want := string(files[PackVenues]), `[{"auction":{"entry_fee":78},"code":"dock"}]`; got != want {
		t.Errorf("venues = %s, want %s", got, want)
	}
	// 包中没有设置和地图时不处理
	if err := dropPool(map[string][]byte{PackShopItems: []byte(`[]`)}); err != nil {
		t.Errorf("no settings or venues: err = %v", err)
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
