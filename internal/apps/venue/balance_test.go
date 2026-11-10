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

package venue

import (
	"slices"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
)

func TestCalibrate(t *testing.T) {
	cases := []struct {
		name  string
		stats ValueStats
		want  AuctionRule
	}{
		{
			"spread out values carry 1.5 times the P90",
			ValueStats{Mean: 10000, P90: 20000, CV: 8900},
			AuctionRule{StartPrice: 400, MaxCarry: 30000, BotCarry: 12000, SpeedFloor: 7500},
		},
		{
			"flat values carry twice the mean",
			ValueStats{Mean: 10000, P90: 12000, CV: 4000},
			AuctionRule{StartPrice: 400, MaxCarry: 20000, BotCarry: 12000, SpeedFloor: 7500},
		},
		{
			"tiny warehouses still start at 1",
			ValueStats{Mean: 10, P90: 10, CV: 0},
			AuctionRule{StartPrice: 1, MaxCarry: 20, BotCarry: 12, SpeedFloor: 7},
		},
	}
	for _, c := range cases {
		// 领先折让和入场费由对局验算求出, 先置 0
		rule := AuctionRule{Rounds: 5, QuickWin: []int32{23000, 18975, 18630, 17825}, LeadDiscount: 1500, EntryFee: 999}
		rule.Calibrate(c.stats, settings.Balance.Default())
		c.want.Rounds, c.want.QuickWin = rule.Rounds, rule.QuickWin
		if !slices.Equal(rule.QuickWin, []int32{23000, 18975, 18630, 17825}) || rule.EntryFee != 0 || rule.StartPrice != c.want.StartPrice ||
			rule.MaxCarry != c.want.MaxCarry || rule.BotCarry != c.want.BotCarry || rule.SpeedFloor != c.want.SpeedFloor || rule.LeadDiscount != 0 ||
			rule.PreviewRate != 3000 {
			t.Errorf("%s: got %+v, want %+v", c.name, rule, c.want)
		}
	}
}

func TestBalanceJudge(t *testing.T) {
	v := &Venue{
		Warehouse:  WarehouseRule{Grid: Grid{Width: 8, Height: 3}},
		Auction:    AuctionRule{Rounds: 5},
		Pve:        PveRule{PracticeOnly: true},
		Simulation: Simulation{Ranked: ValueStats{Mean: 10000}},
	}
	b := &Balance{
		Target: 9800,
		Squads: []SquadStats{
			{Squad: SquadHumans, PriceValue: 9790, SpeedWin: 300, BidCV: 2000, TopShare: 5000, SeatEV: -50},
			{Squad: SquadRanked, AppraiseBias: 10500, WinRate: 2500, SeatEV: -401},
			{Squad: SquadPve, AppraiseBias: 9000, WinRate: 3000, SeatEV: -9999},
		},
	}
	b.judge(v, settings.Balance.Default())
	if b.Passed {
		t.Fatalf("should not pass: %+v", b.Checks)
	}
	var failed []string
	for _, c := range b.Checks {
		if !c.Passed {
			failed = append(failed, c.Code+"/"+c.Squad)
		}
	}
	want := []string{CheckGridHeight + "/", CheckSpeedWin + "/" + SquadHumans, CheckSeatEV + "/" + SquadRanked}
	if !slices.Equal(failed, want) {
		t.Errorf("failed checks %v, want %v", failed, want)
	}
	if got := b.failures(); got != "仓库高度、速胜率(4 个真人)、真人盈亏(排位补人机)" {
		t.Errorf("failures: %q", got)
	}
	for _, c := range b.Checks {
		if c.Code == CheckSeatEV && c.Squad == SquadPve {
			t.Errorf("practice only pve should skip the seat EV check: %+v", c)
		}
	}

	v.Warehouse.Grid.Height = 6
	b.Squads[0].SpeedWin, b.Squads[1].SeatEV = 2000, -400
	b.judge(v, settings.Balance.Default())
	if !b.Passed || b.failures() != "" {
		t.Errorf("should pass: %+v", b.Checks)
	}
	b.Squads[2].WinRate = 2999
	b.judge(v, settings.Balance.Default())
	if got := b.failures(); got != "真人胜率(人机模式)" {
		t.Errorf("win rate gate: %q", got)
	}
	loose := settings.Balance.Default()
	loose.MinRankedWinRate, loose.MinPveWinRate = 0, 0
	b.Squads[1].WinRate = 0
	b.judge(v, loose)
	if !b.Passed {
		t.Errorf("win rate gates turned off should pass: %+v", b.Checks)
	}
	for _, c := range b.Checks {
		if c.Code == CheckWinRate {
			t.Errorf("win rate gates turned off should skip the check: %+v", c)
		}
	}
}

func TestCalibrateSwitches(t *testing.T) {
	b := settings.Balance.Default()
	b.SpeedFloorEnabled, b.PreviewRate = false, 0
	var rule AuctionRule
	rule.Calibrate(ValueStats{Mean: 10000, P90: 20000, CV: 8900}, b)
	if rule.SpeedFloor != 0 || rule.PreviewRate != 0 || rule.StartPrice != 400 {
		t.Errorf("switches off: %+v", rule)
	}
}
