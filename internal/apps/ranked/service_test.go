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

package ranked

import "testing"

func TestDeltaOf(t *testing.T) {
	deltas := []int{30, 10, -10, -30}
	cases := []struct {
		rank int16
		want int32
	}{
		{0, 0},
		{1, 30},
		{2, 10},
		{4, -30},
		{5, 0},
	}
	for _, c := range cases {
		if got := deltaOf(deltas, c.rank); got != c.want {
			t.Errorf("rank %d: got %d, want %d", c.rank, got, c.want)
		}
	}
}

func TestApply(t *testing.T) {
	if got := apply(1000, 30); got != 1030 {
		t.Errorf("gain: got %d", got)
	}
	if got := apply(20, -30); got != 0 {
		t.Errorf("floor: got %d", got)
	}
}

func TestCarry(t *testing.T) {
	cases := []struct {
		name    string
		last    int32
		percent int16
		want    int32
	}{
		{"half above", 1400, 50, 1200},
		{"half below", 800, 50, 900},
		{"full reset", 1400, 0, 1000},
		{"full keep", 1400, 100, 1400},
		{"truncate", 1001, 50, 1000},
	}
	for _, c := range cases {
		if got := carry(1000, c.last, c.percent); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}
