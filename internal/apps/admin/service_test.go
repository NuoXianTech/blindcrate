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

package admin

import (
	"strings"
	"testing"
)

func TestNormalizeRemark(t *testing.T) {
	cases := []struct {
		name, raw, want string
	}{
		{"empty", "", ""},
		{"blank", " \t\n", ""},
		{"trimmed", "  补偿  ", "补偿"},
		{"max length", strings.Repeat("补", 255), strings.Repeat("补", 255)},
	}
	for _, c := range cases {
		if got, err := NormalizeRemark(c.raw); err != nil || got != c.want {
			t.Errorf("%s: got %q, %v, want %q", c.name, got, err, c.want)
		}
	}

	for _, raw := range []string{strings.Repeat("补", 256), "补\x00偿"} {
		if _, err := NormalizeRemark(raw); err == nil || err.Error() != RemarkInvalid {
			t.Errorf("%q: got %v, want %s", raw, err, RemarkInvalid)
		}
	}
}
