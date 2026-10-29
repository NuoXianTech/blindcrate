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

package storage

import "testing"

func TestCheckKey(t *testing.T) {
	valid := []string{"item/ab/abcd.png", "avatar/0f/0f12.jpg", "a.png"}
	invalid := []string{"", "item/../x.png", "item/ab/", "/item/a.png", "item//a.png", "item/a", "Item/a.png", "item/a.b/c.png", "item\a.png"}
	for _, key := range valid {
		if err := CheckKey(key); err != nil {
			t.Errorf("%q: %v", key, err)
		}
	}
	for _, key := range invalid {
		if err := CheckKey(key); err == nil {
			t.Errorf("%q: want error", key)
		}
	}
}

func TestValidURL(t *testing.T) {
	prefix := publicURL()
	cases := map[string]bool{
		"https://example.com/a.png":       true,
		"http://example.com":              true,
		"ftp://example.com/a.png":         false,
		"https://":                        false,
		"javascript:alert(1)":             false,
		"not a url":                       false,
		prefix + "/item/ab/abcd.png":      prefix != "" && prefix[0] == '/',
		prefix + "/item/../../etc/passwd": false,
		"/other/item/ab/abcd.png":         false,
	}
	for s, want := range cases {
		if got := ValidURL(s); got != want {
			t.Errorf("%q: got %v, want %v", s, got, want)
		}
	}
}
