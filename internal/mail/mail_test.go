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

package mail

import "testing"

func TestIsAlias(t *testing.T) {
	cases := map[string]bool{
		"name@example.com":      false,
		"first.last@gmail.com":  false,
		"name+tag@example.com":  true,
		"name+@example.com":     true,
		"+name@example.com":     true,
		"name@exam+ple.com":     false,
		"a+b+c@example.com":     true,
		"name_tag-1@example.cn": false,
	}
	for address, want := range cases {
		if got := IsAlias(address); got != want {
			t.Errorf("IsAlias(%q) = %t, want %t", address, got, want)
		}
	}
}

func TestInDomains(t *testing.T) {
	domains := "QQ.com\n @163.com , example.org;\r\n.edu.cn\n*.ac.uk\n\n"
	cases := map[string]bool{
		"name@ox.ac.uk":         true,
		"name@qq.com":           true,
		"name@QQ.COM":           true,
		"name@163.com":          true,
		"name@mail.example.org": true,
		"name@pku.edu.cn":       true,
		"name@notqq.com":        false,
		"name@qq.com.cn":        false,
		"qq.com@gmail.com":      false,
	}
	for address, want := range cases {
		if got := InDomains(address, domains); got != want {
			t.Errorf("InDomains(%q) = %t, want %t", address, got, want)
		}
	}
	if InDomains("name@qq.com", "") {
		t.Errorf("empty list should match nothing")
	}
}
