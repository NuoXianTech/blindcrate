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
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"gorm.io/gorm"
)

// testStrategy 测试用的策略, 在 init 中注册
const testStrategy = "test_echo"

// echoParams 测试策略的参数
type echoParams struct {
	Mode string `json:"mode" label:"方向" options:"up:向上,down:向下" binding:"oneof=up down"`
	Step int32  `json:"step" label:"步长" unit:"bp" binding:"min=0,max=10000"`
	Low  int32  `json:"low" label:"下限" binding:"min=0"`
	High int32  `json:"high" label:"上限" binding:"min=0"`
	Loud bool   `json:"loud" label:"大声"`
}

// Validate 下限不能高于上限
func (p *echoParams) Validate(*gorm.DB) (string, error) {
	if p.Low > p.High {
		return "下限不能高于上限", nil
	}
	return "", nil
}

type echoObservation struct{ Value int64 }

type echoDecision struct {
	Value int64
	Mode  string
	Roll  int64
}

func decideEcho(obs echoObservation, p *echoParams, rng *rand.Rand) echoDecision {
	return echoDecision{Value: obs.Value + int64(p.Step), Mode: p.Mode, Roll: rng.Int64N(1000)}
}

func init() {
	RegisterStrategy(testStrategy, "测试策略", "观察加上步长", echoParams{Mode: "up", Step: 500, High: 10}, decideEcho)
}

func TestStrategies(t *testing.T) {
	views := Strategies()
	i := slices.IndexFunc(views, func(v StrategyView) bool { return v.Code == testStrategy })
	if i < 0 {
		t.Fatalf("%s not listed", testStrategy)
	}
	view := views[i]
	if view.Label != "测试策略" || len(view.Fields) != 5 {
		t.Fatalf("got %+v", view)
	}
	if f := view.Fields[0]; f.Key != "mode" || f.Type != "string" || f.Input != "select" || len(f.Options) != 2 {
		t.Errorf("mode: got %+v", f)
	}
	if f := view.Fields[1]; f.Key != "step" || f.Type != "int" || f.Unit != "bp" {
		t.Errorf("step: got %+v", f)
	}
	// 默认参数与从数据库读出的一致, 数字为 float64
	want := map[string]any{"mode": "up", "step": float64(500), "low": float64(0), "high": float64(10), "loud": false}
	if !maps.Equal(view.Defaults, want) {
		t.Errorf("defaults: got %v, want %v", view.Defaults, want)
	}
	view.Defaults["step"] = float64(1)
	view.Fields[0].Key = "changed"
	if again := Strategies()[i]; again.Defaults["step"] != float64(500) || again.Fields[0].Key != "mode" {
		t.Error("views should be copies")
	}
}

func TestValidateParams(t *testing.T) {
	got, err := validateParams(nil, testStrategy, map[string]any{"step": 100, "loud": true})
	want := map[string]any{"mode": "up", "step": float64(100), "low": float64(0), "high": float64(10), "loud": true}
	if err != nil || !maps.Equal(got, want) {
		t.Fatalf("missing params: got %v, %v, want %v", got, err, want)
	}
	if got, err := validateParams(nil, testStrategy, nil); err != nil || got["step"] != float64(500) {
		t.Fatalf("nil params: got %v, %v", got, err)
	}

	cases := []struct {
		name     string
		strategy string
		params   map[string]any
		want     string // 提示的前缀
	}{
		{"unknown strategy", "missing", nil, StrategyNotFound},
		{"unknown param", testStrategy, map[string]any{"speed": 1}, ParamsInvalid},
		{"wrong type", testStrategy, map[string]any{"step": "100"}, ParamsInvalid},
		{"binding", testStrategy, map[string]any{"mode": "left"}, strings.TrimSuffix(ParamsRuleInvalid, "%s")},
		{"validator", testStrategy, map[string]any{"low": 20}, fmt.Sprintf(ParamsRuleInvalid, "下限不能高于上限")},
	}
	for _, c := range cases {
		_, err := validateParams(nil, c.strategy, c.params)
		var invalid *InvalidError
		if !errors.As(err, &invalid) || !strings.HasPrefix(invalid.Reason, c.want) {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}

func TestDecide(t *testing.T) {
	rng := func() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }
	// 已删除的参数忽略, 缺少的参数取默认值
	s := &Snapshot{Code: "echo", Strategy: testStrategy, Params: map[string]any{"step": float64(7), "removed": true}}
	first, err := Decide[echoObservation, echoDecision](s, echoObservation{Value: 10}, rng())
	if err != nil || first.Value != 17 || first.Mode != "up" {
		t.Fatalf("got %+v, %v", first, err)
	}
	if again, err := Decide[echoObservation, echoDecision](s, echoObservation{Value: 10}, rng()); err != nil || again != first {
		t.Errorf("same rng: got %+v, %v, want %+v", again, err, first)
	}

	failures := map[string]func() error{
		"unknown strategy": func() error {
			_, err := Decide[echoObservation, echoDecision](&Snapshot{Strategy: "missing"}, echoObservation{}, rng())
			return err
		},
		"bad params": func() error {
			_, err := Decide[echoObservation, echoDecision](&Snapshot{Strategy: testStrategy, Params: map[string]any{"step": "x"}}, echoObservation{}, rng())
			return err
		},
		"observation type": func() error {
			_, err := Decide[int, echoDecision](s, 1, rng())
			return err
		},
		"decision type": func() error {
			_, err := Decide[echoObservation, int](s, echoObservation{}, rng())
			return err
		},
	}
	for name, fn := range failures {
		if fn() == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestRegisterStrategyPanics(t *testing.T) {
	long := strings.Repeat("a", maxStrategyCodeLength+1)
	cases := map[string]func(){
		"empty code":       func() { RegisterStrategy("", "", "", echoParams{Mode: "up"}, decideEcho) },
		"long code":        func() { RegisterStrategy(long, "", "", echoParams{Mode: "up"}, decideEcho) },
		"duplicate":        func() { RegisterStrategy(testStrategy, "", "", echoParams{Mode: "up"}, decideEcho) },
		"invalid defaults": func() { RegisterStrategy("test_invalid", "", "", echoParams{Mode: "left"}, decideEcho) },
	}
	for name, register := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: expected panic", name)
				}
			}()
			register()
		}()
	}
	if _, ok := strategies["test_invalid"]; ok {
		t.Error("invalid strategy should not be registered")
	}
}
