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

package botadmin

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
)

func TestCleanNames(t *testing.T) {
	names, repeated, err := cleanNames([]string{" 晚风 ", "星河", "晚风", "星河\t", "南城"})
	if err != nil || !slices.Equal(names, []string{"晚风", "星河", "南城"}) || !slices.Equal(repeated, []string{"晚风", "星河"}) {
		t.Fatalf("got %v, %v, %v", names, repeated, err)
	}

	cases := []struct {
		name string
		raw  []string
		want string
	}{
		{"empty", nil, bot.NamesEmpty},
		{"too many", slices.Repeat([]string{"晚风"}, bot.MaxBatchNames+1), bot.NamesTooMany},
		{"blank", []string{"晚风", "  "}, fmt.Sprintf(bot.NamesInvalid, 2)},
		{"too long", []string{strings.Repeat("风", 33)}, fmt.Sprintf(bot.NamesInvalid, 1)},
	}
	for _, c := range cases {
		if _, _, err := cleanNames(c.raw); err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}
