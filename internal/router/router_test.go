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

package router

import (
	"reflect"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
)

// TestPackSections 配置种类由各模块的后台包在 init 中注册; 路由引用了全部后台包, 所以这里能看到完整的列表, 按依赖顺序排列
//
// 新增配置种类时在这里补上, 少了一种说明它所在的包没有被引用。
func TestPackSections(t *testing.T) {
	var codes []string
	for _, s := range pack.Sections() {
		codes = append(codes, s.Code)
	}
	want := []string{
		"currencies", "settings", "qualities", "items", "codex_sets", "seasons", "loot_pools", "bot_personas", "bot_names", "venues",
		"shop_items", "leaderboards", "missions", "checkin_days", "ranked_tiers",
	}
	if !reflect.DeepEqual(codes, want) {
		t.Errorf("sections = %v, want %v", codes, want)
	}
}
