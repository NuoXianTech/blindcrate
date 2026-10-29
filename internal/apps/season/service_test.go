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

package season

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
)

func TestStatusAt(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	s := Season{StartsAt: start, EndsAt: start.AddDate(0, 1, 0)}
	cases := []struct {
		name string
		now  time.Time
		want string
	}{
		{"before start", start.Add(-time.Second), StatusUpcoming},
		{"at start", start, StatusActive},
		{"before end", s.EndsAt.Add(-time.Second), StatusActive},
		{"at end", s.EndsAt, StatusEnded},
	}
	for _, c := range cases {
		if got := s.StatusAt(c.now); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
	settled := s.EndsAt.Add(time.Hour)
	s.SettledAt = &settled
	if got := s.StatusAt(settled); got != StatusSettled {
		t.Errorf("settled: got %s", got)
	}
}

func TestValidate(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	valid := Season{Code: "s1", Name: "第一赛季", StartsAt: start, EndsAt: start.AddDate(0, 1, 0)}
	if err := Validate(&valid); err != nil {
		t.Fatalf("valid: got %v", err)
	}
	cases := []struct {
		name   string
		modify func(s *Season)
		want   string
	}{
		{"code upper", func(s *Season) { s.Code = "S1" }, CodeInvalid},
		{"code long", func(s *Season) { s.Code = strings.Repeat("a", 25) }, CodeInvalid},
		{"name empty", func(s *Season) { s.Name = "" }, NameInvalid},
		{"name long", func(s *Season) { s.Name = strings.Repeat("赛", 65) }, NameInvalid},
		{"end equals start", func(s *Season) { s.EndsAt = s.StartsAt }, TimeInvalid},
		{"end before start", func(s *Season) { s.EndsAt = s.StartsAt.Add(-time.Hour) }, TimeInvalid},
		{"carry negative", func(s *Season) { s.RatingCarry = -1 }, CarryInvalid},
		{"carry over", func(s *Season) { s.RatingCarry = 101 }, CarryInvalid},
	}
	for _, c := range cases {
		s := valid
		c.modify(&s)
		if err := Validate(&s); err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}

// TestPeriodLength 最长的赛季代码拼上前缀后不能超过 stats.period 的长度
func TestPeriodLength(t *testing.T) {
	if got := len(Period(strings.Repeat("a", 24))); got > 32 {
		t.Errorf("period length %d exceeds 32", got)
	}
}

func TestNormalizeRewards(t *testing.T) {
	chip := func(amount int64) reward.Bundle {
		return reward.Bundle{Currencies: []reward.CurrencyLine{{Currency: "chip", Amount: amount}}}
	}
	got, err := NormalizeRewards([]Reward{
		{MinRating: 1250, Bundle: reward.Bundle{Items: []reward.ItemLine{{ItemID: 3, Quantity: 1}, {ItemID: 3, Quantity: 2}}}},
		{MinRating: 0, Bundle: chip(100)},
	})
	if err != nil {
		t.Fatalf("valid: got %v", err)
	}
	want := []Reward{
		{MinRating: 0, Bundle: chip(100)},
		{MinRating: 1250, Bundle: reward.Bundle{Items: []reward.ItemLine{{ItemID: 3, Quantity: 3}}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("valid: got %+v, want %+v", got, want)
	}
	if got, err := NormalizeRewards(nil); err != nil || got == nil || len(got) != 0 {
		t.Errorf("nil: got %v, %v", got, err)
	}

	many := make([]Reward, maxRewards+1)
	for i := range many {
		many[i] = Reward{MinRating: int32(i), Bundle: chip(1)}
	}
	lines := reward.Bundle{}
	for i := range maxRewardLines + 1 {
		lines.Items = append(lines.Items, reward.ItemLine{ItemID: uint64(i + 1), Quantity: 1})
	}
	cases := []struct {
		name    string
		rewards []Reward
		want    string
	}{
		{"too many", many, RewardsTooMany},
		{"rating negative", []Reward{{MinRating: -1, Bundle: chip(1)}}, RewardRatingInvalid},
		{"rating over", []Reward{{MinRating: maxRewardRating + 1, Bundle: chip(1)}}, RewardRatingInvalid},
		{"rating taken", []Reward{{MinRating: 1100, Bundle: chip(1)}, {MinRating: 1100, Bundle: chip(2)}}, RewardRatingTaken},
		{"empty", []Reward{{MinRating: 0}}, RewardEmpty},
		{"lines", []Reward{{MinRating: 0, Bundle: lines}}, RewardLinesTooMany},
		{"amount zero", []Reward{{MinRating: 0, Bundle: chip(0)}}, RewardAmountInvalid},
		{"quantity negative", []Reward{{MinRating: 0, Bundle: reward.Bundle{Items: []reward.ItemLine{{ItemID: 1, Quantity: -1}}}}}, RewardAmountInvalid},
		{"currency empty", []Reward{{MinRating: 0, Bundle: reward.Bundle{Currencies: []reward.CurrencyLine{{Amount: 1}}}}}, RewardAmountInvalid},
	}
	for _, c := range cases {
		if _, err := NormalizeRewards(c.rewards); err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}

func TestRewardOf(t *testing.T) {
	s := Season{Rewards: []Reward{{MinRating: 1100}, {MinRating: 1250}, {MinRating: 1400}}}
	cases := []struct {
		peak int32
		want int32 // 领取的分数线, -1 为没有奖励
	}{
		{1099, -1},
		{1100, 1100},
		{1249, 1100},
		{1250, 1250},
		{5000, 1400},
	}
	for _, c := range cases {
		got := s.RewardOf(c.peak)
		if (got == nil) != (c.want < 0) || (got != nil && got.MinRating != c.want) {
			t.Errorf("peak %d: got %+v, want %d", c.peak, got, c.want)
		}
	}
	if got := (&Season{}).RewardOf(5000); got != nil {
		t.Errorf("no rewards: got %+v", got)
	}
}
