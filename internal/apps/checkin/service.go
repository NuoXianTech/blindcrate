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

package checkin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/ledger"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/task/schedule"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	audit.RegisterAction(ActionUpdateDays, "修改签到奖励")
	ledger.RegisterBizType(ledger.KindWallet, BizCheckin, "签到奖励")
	ledger.RegisterBizType(ledger.KindInventory, BizCheckin, "签到奖励")
	stats.RegisterMetric(MetricCheckins, "签到天数")
	// 只需要今天和昨天的签到记录来计算连续天数, 更早的随按天统计一起清理
	stats.RegisterCleanup(Record{}.TableName(), "period")
}

// Settings 签到的设置
var Settings = settings.Register(settings.Group[CheckinSettings]{
	Key:         "checkin",
	Label:       "每日签到",
	Description: "每日签到的开关, 奖励在签到管理中按天配置",
	Category:    settings.CategoryGame,
	Sort:        40,
	Defaults:    func() *CheckinSettings { return &CheckinSettings{Enabled: true} },
})

// CheckinSettings 每日签到
type CheckinSettings struct {
	Enabled bool `json:"enabled" label:"开启签到" help:"关闭后大厅不显示签到, 也不能签到; 再次开启时, 关闭期间断开的连续天数不会恢复"`
}

// InvalidError 签到奖励配置不合法, Error 返回的提示可以直接展示给管理员
type InvalidError struct {
	Reason string
}

func (e *InvalidError) Error() string { return e.Reason }

// RejectedError 不能签到, Error 返回的提示可以直接展示给玩家
type RejectedError struct {
	Reason string
}

func (e *RejectedError) Error() string { return e.Reason }

// ValidateDay 保存某一天的奖励前校验, 校验后合并奖励中的同类项; 不合法时返回 *InvalidError
func ValidateDay(tx *gorm.DB, d *Day) error {
	if d.Day < 1 || d.Day > MaxDays {
		return &InvalidError{Reason: DayInvalid}
	}
	bundle, err := reward.Check(tx, d.Reward)
	if err != nil {
		var invalid *reward.InvalidError
		if errors.As(err, &invalid) {
			return &InvalidError{Reason: fmt.Sprintf(DayRewardInvalid, d.Day, invalid.Reason)}
		}
		return err
	}
	if bundle.IsEmpty() {
		return &InvalidError{Reason: fmt.Sprintf(DayRewardEmpty, d.Day)}
	}
	d.Reward = bundle
	return nil
}

// QueryDays 签到周期中每一天的奖励, 按天排列
func QueryDays(tx *gorm.DB) ([]Day, error) {
	days := []Day{}
	if err := tx.Order("day").Find(&days).Error; err != nil {
		return nil, err
	}
	return days, nil
}

// periods 今天和昨天的签到周期
func periods(now time.Time) (today, yesterday string) {
	year, month, day := now.In(schedule.Location).Date()
	// 取中午, 避免夏令时切换的那天加减一天后落到别的日期
	return stats.PeriodDay(now), stats.PeriodDay(time.Date(year, month, day-1, 12, 0, 0, 0, schedule.Location))
}

// lastRecord 今天或昨天的签到记录, 都没有时为 nil
func lastRecord(tx *gorm.DB, userID uint64, today, yesterday string) (*Record, error) {
	var records []Record
	if err := tx.Where("user_id = ? AND period IN ?", userID, []string{today, yesterday}).
		Order("period DESC").Limit(1).Find(&records).Error; err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return &records[0], nil
}

// cycleDay 连续签到 streak 天时领第几天的奖励, 超过周期时从第 1 天重新开始
func cycleDay(streak int32, days int) int16 {
	return int16((int(streak)-1)%days + 1)
}

// DayView 签到周期中的一天
type DayView struct {
	Day    int16         `json:"day"`
	Reward []reward.Line `json:"reward"`
}

// StatusView 玩家的签到情况
type StatusView struct {
	Enabled   bool      `json:"enabled"`    // 关闭时其他字段为空
	CheckedIn bool      `json:"checked_in"` // 今天已签到
	Streak    int32     `json:"streak"`     // 连续签到的天数, 今天还没签时为到昨天为止的天数, 已断开时为 0
	Today     int16     `json:"today"`      // 今天签到领(或已领)第几天的奖励, 已签到后周期有变化时按新的周期计算; 没有配置奖励时为 0
	Total     int64     `json:"total"`      // 累计签到天数
	Days      []DayView `json:"days"`       // 整个周期的奖励
}

// QueryStatus 获取玩家今天的签到情况和整个周期的奖励
func QueryStatus(ctx context.Context, userID uint64) (*StatusView, error) {
	cfg, err := Settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	view := &StatusView{Days: []DayView{}}
	if !cfg.Enabled {
		return view, nil
	}
	view.Enabled = true

	tx := db.DB(ctx)
	days, err := QueryDays(tx)
	if err != nil {
		return nil, err
	}
	bundles := make([]reward.Bundle, len(days))
	for i := range days {
		bundles[i] = days[i].Reward
	}
	lines, err := reward.Lines(tx, bundles...)
	if err != nil {
		return nil, err
	}
	for i := range days {
		view.Days = append(view.Days, DayView{Day: days[i].Day, Reward: lines[i]})
	}

	today, yesterday := periods(time.Now())
	last, err := lastRecord(tx, userID, today, yesterday)
	if err != nil {
		return nil, err
	}
	// 含今天在内的连续天数; 已签到时也按它计算今天对应第几天, 周期改短后不会超出天数, 并与明天衔接
	streak := int32(1)
	if last != nil {
		view.Streak = last.Streak
		view.CheckedIn = last.Period == today
		streak = last.Streak
		if !view.CheckedIn {
			streak++
		}
	}
	if len(days) > 0 {
		view.Today = cycleDay(streak, len(days))
	}
	if view.Total, err = stats.Get(tx, userID, stats.ScopeAll, stats.PeriodTotal, MetricCheckins); err != nil {
		return nil, err
	}
	return view, nil
}

// CheckinResult 签到的结果
type CheckinResult struct {
	Streak   int32         `json:"streak"`   // 含今天在内的连续签到天数
	Day      int16         `json:"day"`      // 领到的是第几天的奖励
	Received []reward.Line `json:"received"` // 实际得到的货币和物品, 已下架的物品按回收价折成了货币
}

// Checkin 今天签到, 按连续天数领取对应一天的奖励
//
// 签到记录的主键防止同一天重复签到, 发放的幂等键再兜底一次; 并发签到时只有一个能写入记录。
// 奖励中已下架或已删除的物品按回收价折成货币; 货币已停用时返回 RewardUnavailable, 今天之后仍可以再签。
func Checkin(ctx context.Context, userID uint64) (*CheckinResult, error) {
	cfg, err := Settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, &RejectedError{Reason: CheckinDisabled}
	}
	var (
		record   Record
		result   *reward.Result
		replayed bool
	)
	now := time.Now()
	today, yesterday := periods(now)
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 同一用户的签到串行执行, 避免跨零点的两次请求都读不到对方的记录而断开连续天数
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", fmt.Sprintf("blindcrate:checkin:%d", userID)).Error; err != nil {
			return err
		}
		days, err := QueryDays(tx)
		if err != nil {
			return err
		}
		if len(days) == 0 {
			return &RejectedError{Reason: DaysEmpty}
		}
		last, err := lastRecord(tx, userID, today, yesterday)
		if err != nil {
			return err
		}
		if last != nil && last.Period == today {
			return errors.New(AlreadyCheckedIn)
		}
		record = Record{UserID: userID, Period: today, Streak: 1}
		if last != nil {
			record.Streak = last.Streak + 1
		}
		record.CycleDay = cycleDay(record.Streak, len(days))
		inserted := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&record)
		if inserted.Error != nil {
			return inserted.Error
		}
		if inserted.RowsAffected == 0 {
			return errors.New(AlreadyCheckedIn)
		}
		if err := stats.Add(tx, userID, []string{stats.ScopeAll}, now, stats.Values{MetricCheckins: 1}); err != nil {
			return err
		}
		bundle, err := reward.Deliverable(tx, days[record.CycleDay-1].Reward, nil)
		if err != nil {
			return err
		}
		if result, replayed, err = reward.Apply(tx, &reward.ApplyRequest{
			UserID:         userID,
			Bundle:         bundle,
			BizType:        BizCheckin,
			BizID:          today,
			IdempotencyKey: fmt.Sprintf("checkin:%d:%s", userID, today),
			Remark:         fmt.Sprintf("签到第 %d 天", record.CycleDay),
		}); err != nil {
			switch err.Error() {
			case item.ItemNotFound, item.ItemDisabled, wallet.CurrencyNotFound, wallet.CurrencyDisabled:
				return &RejectedError{Reason: RewardUnavailable}
			case wallet.AmountInvalid, inventory.QuantityInvalid:
				return &RejectedError{Reason: RewardTooLarge}
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !replayed {
		result.Push(ctx)
	}
	lines, err := reward.Lines(db.DB(ctx), result.Bundle)
	if err != nil {
		return nil, err
	}
	return &CheckinResult{Streak: record.Streak, Day: record.CycleDay, Received: lines[0]}, nil
}
