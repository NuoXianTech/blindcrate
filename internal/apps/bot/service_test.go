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

package bot

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	cases := map[string]bool{"": false, "晚风": true, strings.Repeat("风", 32): true, strings.Repeat("风", 33): false}
	for name, ok := range cases {
		if err := ValidateName(name); (err == nil) != ok {
			t.Errorf("%q: got %v", name, err)
		}
	}
}

// TestValidate 只校验不需要查库的部分, 测试策略见 strategy_test.go
func TestValidate(t *testing.T) {
	valid := func() Persona {
		return Persona{Code: "steady_2", Name: strings.Repeat("稳", 32), Strategy: testStrategy, Params: map[string]any{"step": 1}}
	}
	p := valid()
	if err := Validate(nil, &p); err != nil || p.Params["step"] != float64(1) || p.Params["mode"] != "up" {
		t.Fatalf("valid: got %v, params %v", err, p.Params)
	}

	cases := []struct {
		name   string
		modify func(p *Persona)
		want   string
	}{
		{"code uppercase", func(p *Persona) { p.Code = "Steady" }, CodeInvalid},
		{"code empty", func(p *Persona) { p.Code = "" }, CodeInvalid},
		{"code too long", func(p *Persona) { p.Code = strings.Repeat("a", 33) }, CodeInvalid},
		{"name empty", func(p *Persona) { p.Name = "" }, PersonaNameInvalid},
		{"name too long", func(p *Persona) { p.Name = strings.Repeat("稳", 33) }, PersonaNameInvalid},
		{"description too long", func(p *Persona) { p.Description = strings.Repeat("说", 256) }, DescriptionInvalid},
		{"strategy", func(p *Persona) { p.Strategy = "missing" }, StrategyNotFound},
		{"params", func(p *Persona) { p.Params = map[string]any{"low": 20} }, fmt.Sprintf(ParamsRuleInvalid, "下限不能高于上限")},
	}
	for _, c := range cases {
		p := valid()
		c.modify(&p)
		var invalid *InvalidError
		if err := Validate(nil, &p); !errors.As(err, &invalid) || invalid.Reason != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}

// TestCheckWeights 重复的风格在查库之前就能发现
func TestCheckWeights(t *testing.T) {
	if reason, err := CheckWeights(nil, nil); reason != "" || err != nil {
		t.Errorf("empty: got %q, %v", reason, err)
	}
	weights := []Weight{{Persona: "steady", Weight: 50}, {Persona: "novice", Weight: 30}, {Persona: "steady", Weight: 20}}
	if reason, err := CheckWeights(nil, weights); reason != fmt.Sprintf(PersonasDuplicated, "steady") || err != nil {
		t.Errorf("duplicated: got %q, %v", reason, err)
	}
}

func TestAssignWithoutBots(t *testing.T) {
	if snapshots, err := Assign(nil, nil, 0); snapshots != nil || err != nil {
		t.Errorf("no bots: got %v, %v", snapshots, err)
	}
	if _, err := Assign(nil, nil, 2); err == nil {
		t.Error("bots without personas should fail")
	}
}
