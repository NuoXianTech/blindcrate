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

package effect

import (
	"reflect"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/item"
)

type testParams struct {
	Count int32 `json:"count" label:"件数" binding:"min=1,max=5"`
}

func init() {
	Register("test_double", SceneMatch, "测试", "返回两倍的件数", func(base int32, params *testParams) (int32, error) {
		return base * params.Count * 2, nil
	})
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name     string
		category string
		code     string
		params   map[string]any
		want     string
	}{
		{"collectible", item.CategoryCollectible, "test_double", map[string]any{"count": 1}, EffectNotAllowed},
		{"params without effect", CategoryConsumable, "", map[string]any{"count": 1}, ParamsWithoutEffect},
		{"unknown effect", CategoryConsumable, "nope", nil, EffectNotFound},
		{"unknown field", CategoryConsumable, "test_double", map[string]any{"count": 1, "extra": 1}, ParamsInvalid},
		{"wrong type", CategoryConsumable, "test_double", map[string]any{"count": "x"}, ParamsInvalid},
	}
	for _, c := range cases {
		_, err := Validate(nil, c.category, c.code, c.params)
		if err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
	if _, err := Validate(nil, CategoryConsumable, "test_double", map[string]any{"count": 9}); err == nil {
		t.Errorf("out of range: got nil")
	}

	got, err := Validate(nil, CategoryConsumable, "test_double", map[string]any{"count": 3})
	if err != nil || !reflect.DeepEqual(got, map[string]any{"count": float64(3)}) {
		t.Errorf("valid: got %v %v", got, err)
	}
	got, err = Validate(nil, item.CategoryCollectible, "", nil)
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("no effect: got %v %v", got, err)
	}
}

func TestApply(t *testing.T) {
	got, err := Apply[int32, int32](SceneMatch, "test_double", 10, map[string]any{"count": float64(3)})
	if err != nil || got != 60 {
		t.Errorf("got %d %v, want 60", got, err)
	}
	if _, err := Apply[int32, int32]("other", "test_double", 10, nil); err == nil {
		t.Errorf("wrong scene: got nil")
	}
	if _, err := Apply[string, int32](SceneMatch, "test_double", "x", nil); err == nil {
		t.Errorf("wrong context: got nil")
	}
	if _, err := Apply[int32, string](SceneMatch, "test_double", 10, nil); err == nil {
		t.Errorf("wrong result: got nil")
	}
}
