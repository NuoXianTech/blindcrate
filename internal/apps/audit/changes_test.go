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

package audit

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"
)

func TestChanges(t *testing.T) {
	type line struct {
		Currency string `json:"currency"`
		Amount   int64  `json:"amount"`
	}
	type requirement struct {
		Type   string          `json:"type"`
		Params json.RawMessage `json:"params"`
	}
	stock := int64(5)
	fields := func(name string, lines []line, params string, big int64, stock interface{}, extra map[string]any) map[string]interface{} {
		return map[string]interface{}{
			"name":         name,
			"price":        lines,
			"requirements": []requirement{{Type: "min_stat", Params: json.RawMessage(params)}},
			"big":          big,
			"stock":        stock,
			"params":       extra,
			"enabled":      true,
		}
	}
	// 从数据库读出的 JSONB: 键按 JSONB 的顺序排列, 冒号和逗号后有空格
	old := fields("a", []line{{"chip", 100}}, `{"count": 3, "scope": "all", "metric": "matches"}`, 1<<60, stock, map[string]any{"bias": float64(1000), "guard": true})
	// 请求中的写法: 键按表单的顺序, 没有空格; map 中的数字是另一种类型
	same := fields("a", []line{{"chip", 100}}, `{"metric":"matches","scope":"all","count":3}`, 1<<60, &stock, map[string]any{"guard": true, "bias": 1000})
	if changes := Changes(old, same); len(changes) != 0 {
		t.Errorf("same content: got %v", changes)
	}

	next := fields("b", []line{{"chip", 100}, {"gold", 1}}, `{"metric":"matches","scope":"all","count":5}`, 1<<60+1, nil, map[string]any{"bias": 1000, "guard": false})
	changes := Changes(old, next)
	changed := slices.Sorted(maps.Keys(changes))
	if want := []string{"big", "name", "params", "price", "requirements", "stock"}; !slices.Equal(changed, want) {
		t.Errorf("changed: got %v, want %v", changed, want)
	}
	if detail := changes["name"].(map[string]interface{}); detail["old"] != "a" || detail["new"] != "b" {
		t.Errorf("name: got %v", detail)
	}

	// 只比较修改前有的字段; 修改后缺少的字段按空值比较
	if changes := Changes(map[string]interface{}{"name": "a"}, map[string]interface{}{"name": "a", "other": 1}); len(changes) != 0 {
		t.Errorf("extra field: got %v", changes)
	}
	if changes := Changes(map[string]interface{}{"name": "a"}, map[string]interface{}{}); len(changes) != 1 {
		t.Errorf("missing field: got %v", changes)
	}
	// 空切片和 nil 切片的内容不同, 与数据库中的 [] 和 null 一致
	if changes := Changes(map[string]interface{}{"ids": []int{}}, map[string]interface{}{"ids": []int(nil)}); len(changes) != 1 {
		t.Errorf("empty and nil slice: got %v", changes)
	}
	// 无法序列化的值按值比较
	ch := make(chan int)
	if changes := Changes(map[string]interface{}{"c": ch}, map[string]interface{}{"c": ch}); len(changes) != 0 {
		t.Errorf("unmarshalable: got %v", changes)
	}
}
