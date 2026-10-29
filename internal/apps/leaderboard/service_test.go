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

package leaderboard

import (
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
)

// TestStatsPeriod 累计和当天不需要查库, 赛季需要查询赛季表
func TestStatsPeriod(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	if got, current, err := statsPeriod(nil, PeriodTotal, now); err != nil || got != stats.PeriodTotal || current != nil {
		t.Errorf("total: got %s, %v, %v", got, current, err)
	}
	if got, current, err := statsPeriod(nil, PeriodDay, now); err != nil || got != stats.PeriodDay(now) || current != nil {
		t.Errorf("day: got %s, %v, %v", got, current, err)
	}
}

// TestValidateFields 只校验不需要查库的字段, 指标在最后校验, 前面的字段不合法时不会查库
func TestValidateFields(t *testing.T) {
	valid := Board{Code: "ranked_wins", Name: "排位胜场", Metric: "wins", Scope: "mode:ranked", Period: PeriodTotal, Size: DefaultSize}
	cases := []struct {
		name   string
		modify func(b *Board)
		want   string
	}{
		{"code upper", func(b *Board) { b.Code = "Ranked" }, CodeInvalid},
		{"code empty", func(b *Board) { b.Code = "" }, CodeInvalid},
		{"name empty", func(b *Board) { b.Name = "" }, NameInvalid},
		{"scope invalid", func(b *Board) { b.Scope = "mode:" }, ScopeInvalid},
		{"period week", func(b *Board) { b.Period = "week" }, PeriodInvalid},
		{"size small", func(b *Board) { b.Size = MinSize - 1 }, SizeInvalid},
		{"size large", func(b *Board) { b.Size = MaxSize + 1 }, SizeInvalid},
		{"min matches negative", func(b *Board) { b.MinMatches = -1 }, MinMatchesInvalid},
		{"rating scope", func(b *Board) { b.Metric, b.Period = MetricRating, PeriodSeason }, RatingBoardInvalid},
		{"rating period", func(b *Board) { b.Metric, b.Scope = MetricRating, stats.ScopeAll }, RatingBoardInvalid},
	}
	for _, c := range cases {
		b := valid
		c.modify(&b)
		if err := Validate(nil, &b); err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}

// TestValidateRating 段位分榜单不需要查库
func TestValidateRating(t *testing.T) {
	b := Board{Code: "season_rating", Name: "段位分", Metric: MetricRating, Scope: stats.ScopeAll, Period: PeriodSeason, Size: DefaultSize}
	if err := Validate(nil, &b); err != nil {
		t.Errorf("rating: got %v", err)
	}
	if name, err := metricName(nil, MetricRating); err != nil || name != ratingName {
		t.Errorf("rating name: got %q, %v", name, err)
	}
}

func TestMetricNameRegistered(t *testing.T) {
	name, err := metricName(nil, stats.MetricWins)
	if err != nil || name == "" {
		t.Errorf("wins: got %q, %v", name, err)
	}
	if _, err := metricName(nil, "unknown"); err == nil || err.Error() != MetricInvalid {
		t.Errorf("unknown: got %v", err)
	}
}
