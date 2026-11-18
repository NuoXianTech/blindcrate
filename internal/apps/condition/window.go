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

package condition

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/task/schedule"
	"gorm.io/gorm"
)

// 本文件是时间窗口: 起止时间、星期几、每天几点到几点, 填了的几项需要同时满足

// timeWindowParams 时间窗口的参数; 星期和每天的时间按 schedule.time_zone 计算, 与按天统计一致
type timeWindowParams struct {
	StartsAt string `json:"starts_at" label:"开始时间" help:"从这一刻开始, 留空为不限" input:"datetime"`
	EndsAt   string `json:"ends_at" label:"结束时间" help:"到这一刻为止, 留空为不限" input:"datetime"`
	Weekdays []int  `json:"weekdays" label:"星期" help:"不选为每天" options:"1:周一,2:周二,3:周三,4:周四,5:周五,6:周六,7:周日" binding:"max=7,unique,dive,min=1,max=7"`
	From     string `json:"from" label:"每天开始" help:"时:分, 如 19:00; 与每天结束都留空为全天" binding:"max=5"`
	To       string `json:"to" label:"每天结束" help:"时:分, 如 23:00, 到这一刻为止; 早于每天开始时跨到第二天, 星期按开始的那天算" binding:"max=5"`
}

// window 解析后的时间窗口
type window struct {
	starts, ends time.Time // 零值为不限
	weekdays     []int     // 周一为 1, 周日为 7, 从小到大; 为空表示每天
	clock        bool      // 是否限制每天的时间
	from, to     int       // 每天的开始和结束, 从零点起的分钟数; from 大于 to 时跨到第二天
}

// weekdayNames 星期的名称, 下标为周一 1 到周日 7
var weekdayNames = [...]string{"", "周一", "周二", "周三", "周四", "周五", "周六", "周日"}

// parseClock 解析 时:分, 返回从零点起的分钟数; end 为 true 时可以是 24:00
func parseClock(s string, end bool) (int, bool) {
	if len(s) != 5 || s[2] != ':' {
		return 0, false
	}
	for _, i := range []int{0, 1, 3, 4} {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	hour, minute := int(s[0]-'0')*10+int(s[1]-'0'), int(s[3]-'0')*10+int(s[4]-'0')
	if minute > 59 || hour > 24 || (hour == 24 && (minute != 0 || !end)) {
		return 0, false
	}
	return hour*60 + minute, true
}

// parse 解析参数, 不合法时返回原因
func (p *timeWindowParams) parse() (*window, string) {
	w := &window{weekdays: slices.Sorted(slices.Values(p.Weekdays))}
	var err error
	if p.StartsAt != "" {
		if w.starts, err = time.Parse(time.RFC3339, p.StartsAt); err != nil {
			return nil, WindowTimeInvalid
		}
	}
	if p.EndsAt != "" {
		if w.ends, err = time.Parse(time.RFC3339, p.EndsAt); err != nil {
			return nil, WindowTimeInvalid
		}
	}
	if !w.starts.IsZero() && !w.ends.IsZero() && !w.ends.After(w.starts) {
		return nil, WindowTimeOrder
	}
	if (p.From == "") != (p.To == "") {
		return nil, WindowClockPair
	}
	if p.From != "" {
		var okFrom, okTo bool
		w.from, okFrom = parseClock(p.From, false)
		w.to, okTo = parseClock(p.To, true)
		if !okFrom || !okTo {
			return nil, WindowClockFormat
		}
		if w.from == w.to {
			return nil, WindowClockSame
		}
		w.clock = true
	}
	if w.starts.IsZero() && w.ends.IsZero() && len(w.weekdays) == 0 && !w.clock {
		return nil, WindowEmpty
	}
	return w, ""
}

// Validate 时间的格式和先后必须正确, 至少填写一项限制
func (p *timeWindowParams) Validate(*gorm.DB) (string, error) {
	_, reason := p.parse()
	return reason, nil
}

// contains at 是否在时间窗口内
//
// 每天的时间跨到第二天时(如 22:00 到 02:00), 零点之后的部分属于前一天开始的那一段, 星期按前一天算。
func (w *window) contains(at time.Time) bool {
	if !w.starts.IsZero() && at.Before(w.starts) {
		return false
	}
	if !w.ends.IsZero() && !at.Before(w.ends) {
		return false
	}
	day := at.In(schedule.Location)
	if w.clock {
		minute := day.Hour()*60 + day.Minute()
		if w.from < w.to {
			if minute < w.from || minute >= w.to {
				return false
			}
		} else {
			if minute >= w.to && minute < w.from {
				return false
			}
			if minute < w.to {
				day = day.AddDate(0, 0, -1)
			}
		}
	}
	if len(w.weekdays) == 0 {
		return true
	}
	weekday := int(day.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return slices.Contains(w.weekdays, weekday)
}

// text 展示给玩家的描述, 如 "开放时间: 每周六、周日 19:00-23:00"
func (w *window) text() string {
	const layout = "2006-01-02 15:04"
	var parts []string
	starts, ends := w.starts.In(schedule.Location).Format(layout), w.ends.In(schedule.Location).Format(layout)
	switch {
	case !w.starts.IsZero() && !w.ends.IsZero():
		parts = append(parts, starts+" 至 "+ends)
	case !w.starts.IsZero():
		parts = append(parts, starts+" 起")
	case !w.ends.IsZero():
		parts = append(parts, ends+" 前")
	}
	if len(w.weekdays) > 0 {
		names := make([]string, len(w.weekdays))
		for i, weekday := range w.weekdays {
			names[i] = weekdayNames[weekday]
		}
		parts = append(parts, "每"+strings.Join(names, "、"))
	}
	if w.clock {
		next := ""
		if w.from > w.to {
			next = "次日 "
		}
		parts = append(parts, fmt.Sprintf("%02d:%02d-%s%02d:%02d", w.from/60, w.from%60, next, w.to/60, w.to%60))
	}
	return "开放时间: " + strings.Join(parts, " ")
}

func checkTimeWindow(_ *gorm.DB, _ uint64, p *timeWindowParams) (*Progress, error) {
	w, reason := p.parse()
	if reason != "" {
		return nil, fmt.Errorf("condition: invalid time window: %s", reason)
	}
	return &Progress{Met: w.contains(time.Now()), Text: w.text()}, nil
}
