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

package mission

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/ledger"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/task/schedule"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	ledger.RegisterBizType(ledger.KindWallet, BizReward, "任务奖励")
	ledger.RegisterBizType(ledger.KindInventory, BizReward, "任务奖励")
	audit.RegisterAction(ActionCreate, "新建任务")
	audit.RegisterAction(ActionUpdate, "修改任务")
	audit.RegisterAction(ActionDelete, "删除任务")
	// 每日和每周的领取记录过了当前周期就不再查询, 随按天、按周统计一起清理
	stats.RegisterCleanup(ClaimRecord{}.TableName(), "period")
}

// codePattern 任务代码, 小写字母、数字和下划线
var codePattern = regexp.MustCompile(`^[a-z0-9_]{1,32}$`)

// InvalidError 任务配置不合法, Error 返回的提示可以直接展示给管理员
type InvalidError struct {
	Reason string
}

func (e *InvalidError) Error() string { return e.Reason }

// RejectedError 不能领取, Error 返回的提示可以直接展示给玩家
type RejectedError struct {
	Reason string
}

func (e *RejectedError) Error() string { return e.Reason }

// Cycle 一种任务周期
type Cycle struct {
	Code  string `json:"code"`
	Label string `json:"label"` // 每日、每周或成就
}

// Cycles 全部任务周期, 按玩家页面的页签顺序
func Cycles() []Cycle {
	return slices.Clone(cycles)
}

// ValidCycle 是否为已知的任务周期
func ValidCycle(cycle string) bool {
	return slices.ContainsFunc(cycles, func(c Cycle) bool { return c.Code == cycle })
}

// Validate 保存任务前校验配置, 校验后合并奖励中的同类项
//
// 配置不合法时返回 *InvalidError; 其他错误为查库失败等内部错误。名称等文本字段需要调用方先去掉首尾空白。
func Validate(tx *gorm.DB, m *Mission) error {
	if !codePattern.MatchString(m.Code) {
		return &InvalidError{Reason: CodeInvalid}
	}
	if n := utf8.RuneCountInString(m.Name); n < 1 || n > maxNameLength {
		return &InvalidError{Reason: NameInvalid}
	}
	if utf8.RuneCountInString(m.Description) > maxDescriptionLength {
		return &InvalidError{Reason: DescriptionInvalid}
	}
	if !ValidCycle(m.Cycle) {
		return &InvalidError{Reason: CycleInvalid}
	}
	if !stats.ValidScope(m.Scope) {
		return &InvalidError{Reason: ScopeInvalid}
	}
	if m.Target < 1 || m.Target > MaxTarget {
		return &InvalidError{Reason: TargetInvalid}
	}
	if _, ok, err := stats.MetricName(tx, m.Metric); err != nil {
		return err
	} else if !ok {
		return &InvalidError{Reason: MetricInvalid}
	}
	bundle, err := reward.Check(tx, m.Reward)
	if err != nil {
		var invalid *reward.InvalidError
		if errors.As(err, &invalid) {
			return &InvalidError{Reason: fmt.Sprintf(RewardInvalid, invalid.Reason)}
		}
		return err
	}
	if bundle.IsEmpty() {
		return &InvalidError{Reason: RewardEmpty}
	}
	m.Reward = bundle
	if m.Requirements == nil {
		m.Requirements = []condition.Condition{}
	}
	if err := condition.Validate(tx, m.Requirements); err != nil {
		var conditionErr *condition.InvalidError
		if errors.As(err, &conditionErr) {
			return &InvalidError{Reason: fmt.Sprintf(RequirementInvalid, conditionErr.Error())}
		}
		return err
	}
	return nil
}

// Period 周期为 cycle 的任务在 now 时对应的统计周期, 也是领取记录的周期
func Period(cycle string, now time.Time) string {
	switch cycle {
	case CycleDaily:
		return stats.PeriodDay(now)
	case CycleWeekly:
		return stats.PeriodWeek(now)
	}
	return stats.PeriodTotal
}

// ResetsAt 周期为 cycle 的任务在 now 之后下次重置的时间, 按 schedule.time_zone 计算; 成就不重置, 返回 nil
func ResetsAt(cycle string, now time.Time) *time.Time {
	local := now.In(schedule.Location)
	year, month, day := local.Date()
	var next time.Time
	switch cycle {
	case CycleDaily:
		next = time.Date(year, month, day+1, 0, 0, 0, 0, schedule.Location)
	case CycleWeekly:
		// 到下一个周一的天数, 今天是周一时为 7
		days := 7 - (int(local.Weekday())+6)%7
		next = time.Date(year, month, day+days, 0, 0, 0, 0, schedule.Location)
	default:
		return nil
	}
	return &next
}

// MissionView 玩家看到的任务, 带当前周期的进度和领取情况
type MissionView struct {
	Code         string               `json:"code"`
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	Cycle        string               `json:"cycle"`
	Progress     int64                `json:"progress"` // 当前周期的指标值, 净收益可以为负
	Target       int64                `json:"target"`
	Currency     string               `json:"currency"` // 净收益指标的货币, 进度和目标按它的最小单位计; 计数指标为空
	Decimals     int16                `json:"decimals"` // 货币的展示小数位, 计数指标为 0
	Reward       []reward.Line        `json:"reward"`
	Requirements []condition.Progress `json:"requirements"` // 每个任务条件的达成情况
	Eligible     bool                 `json:"eligible"`     // 满足全部任务条件
	Claimed      bool                 `json:"claimed"`      // 当前周期已领取
	Claimable    bool                 `json:"claimable"`    // 已完成、满足条件且未领取
}

// GroupView 一种周期的任务
type GroupView struct {
	Cycle    string        `json:"cycle"`
	Label    string        `json:"label"`     // 每日、每周或成就
	ResetsAt *time.Time    `json:"resets_at"` // 下次重置的时间, 成就为空
	Missions []MissionView `json:"missions"`
}

// MissionsData 玩家的全部任务
type MissionsData struct {
	Groups    []GroupView `json:"groups"`    // 按每日、每周、成就排列, 没有任务的周期也返回
	Claimable int         `json:"claimable"` // 可以领取的任务数
}

// statKey 一个任务的进度对应的统计
type statKey struct {
	scope, period, metric string
}

// claimKey 一条领取记录
type claimKey struct {
	missionID uint64
	period    string
}

// QueryMissions 获取启用的任务, 按周期分组, 同一周期内按排序值和 ID 排列, 附带用户的进度和领取情况
func QueryMissions(ctx context.Context, userID uint64) (*MissionsData, error) {
	tx := db.DB(ctx)
	now := time.Now()
	var missions []Mission
	if err := tx.Where("is_enabled").Order("sort_order, id").Find(&missions).Error; err != nil {
		return nil, err
	}

	progress, err := queryProgress(tx, userID, missions, now)
	if err != nil {
		return nil, err
	}
	claimed, err := queryClaimed(tx, userID, missions, now)
	if err != nil {
		return nil, err
	}
	bundles := make([]reward.Bundle, len(missions))
	for i := range missions {
		bundles[i] = missions[i].Reward
	}
	lines, err := reward.Lines(tx, bundles...)
	if err != nil {
		return nil, err
	}
	decimals, err := queryDecimals(tx, missions)
	if err != nil {
		return nil, err
	}

	data := &MissionsData{Groups: make([]GroupView, len(cycles))}
	groups := map[string]*GroupView{}
	for i, c := range cycles {
		data.Groups[i] = GroupView{Cycle: c.Code, Label: c.Label, ResetsAt: ResetsAt(c.Code, now), Missions: []MissionView{}}
		groups[c.Code] = &data.Groups[i]
	}
	for i := range missions {
		m := &missions[i]
		group := groups[m.Cycle]
		if group == nil {
			continue
		}
		requirements, err := condition.Evaluate(tx, userID, m.Requirements)
		if err != nil {
			return nil, err
		}
		eligible := true
		for _, p := range requirements {
			eligible = eligible && p.Met
		}
		period := Period(m.Cycle, now)
		view := MissionView{
			Code:         m.Code,
			Name:         m.Name,
			Description:  m.Description,
			Cycle:        m.Cycle,
			Progress:     progress[statKey{m.Scope, period, m.Metric}],
			Target:       m.Target,
			Reward:       lines[i],
			Requirements: requirements,
			Eligible:     eligible,
			Claimed:      claimed[claimKey{m.ID, period}],
		}
		if code, ok := stats.ProfitCurrency(m.Metric); ok {
			view.Currency, view.Decimals = code, decimals[code]
		}
		view.Claimable = eligible && !view.Claimed && view.Progress >= view.Target
		if view.Claimable {
			data.Claimable++
		}
		group.Missions = append(group.Missions, view)
	}
	return data, nil
}

// queryProgress 一次查出用户在每个任务当前周期的指标值, 没有记录的为 0
func queryProgress(tx *gorm.DB, userID uint64, missions []Mission, now time.Time) (map[statKey]int64, error) {
	progress := map[statKey]int64{}
	if len(missions) == 0 {
		return progress, nil
	}
	keys := make([][]any, len(missions))
	for i, m := range missions {
		keys[i] = []any{m.Scope, Period(m.Cycle, now), m.Metric}
	}
	var rows []stats.Stat
	if err := tx.Select("scope, period, metric, value").
		Where("user_id = ? AND (scope, period, metric) IN ?", userID, keys).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		progress[statKey{row.Scope, row.Period, row.Metric}] = row.Value
	}
	return progress, nil
}

// queryDecimals 净收益任务用到的货币的小数位, 没有这类任务时不查库
func queryDecimals(tx *gorm.DB, missions []Mission) (map[string]int16, error) {
	decimals := map[string]int16{}
	var codes []string
	for _, m := range missions {
		if code, ok := stats.ProfitCurrency(m.Metric); ok {
			codes = append(codes, code)
		}
	}
	if len(codes) == 0 {
		return decimals, nil
	}
	var currencies []wallet.Currency
	if err := tx.Select("code, decimals").Where("code IN ?", codes).Find(&currencies).Error; err != nil {
		return nil, err
	}
	for _, c := range currencies {
		decimals[c.Code] = c.Decimals
	}
	return decimals, nil
}

// queryClaimed 一次查出用户在每个任务当前周期是否已领取
func queryClaimed(tx *gorm.DB, userID uint64, missions []Mission, now time.Time) (map[claimKey]bool, error) {
	claimed := map[claimKey]bool{}
	if len(missions) == 0 {
		return claimed, nil
	}
	ids := make([]uint64, len(missions))
	for i, m := range missions {
		ids[i] = m.ID
	}
	periods := make([]string, len(cycles))
	for i, c := range cycles {
		periods[i] = Period(c.Code, now)
	}
	var rows []ClaimRecord
	if err := tx.Select("mission_id, period").
		Where("mission_id IN ? AND user_id = ? AND period IN ?", ids, userID, periods).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		claimed[claimKey{row.MissionID, row.Period}] = true
	}
	return claimed, nil
}

// ClaimResult 领取的结果
type ClaimResult struct {
	Received []reward.Line `json:"received"` // 实际得到的货币和物品, 已下架的物品按回收价折成了货币
}

// Claim 领取任务当前周期的奖励
//
// 任务加共享锁, 与后台修改和删除互斥; 领取记录的主键防止同一周期重复领取, 发放的幂等键再兜底一次。
// 奖励中已下架或已删除的物品按回收价折成货币; 货币已停用时返回 RewardUnavailable, 可以之后再领取。
func Claim(ctx context.Context, userID uint64, code string) (*ClaimResult, error) {
	var (
		result   *reward.Result
		replayed bool
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var m Mission
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("code = ? AND is_enabled", code).First(&m).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(MissionNotFound)
			}
			return err
		}
		period := Period(m.Cycle, time.Now())
		unmet, err := condition.FirstUnmet(tx, userID, m.Requirements)
		if err != nil {
			return err
		}
		if unmet != nil {
			return &RejectedError{Reason: fmt.Sprintf(RequirementUnmet, unmet.Text)}
		}
		value, err := stats.Get(tx, userID, m.Scope, period, m.Metric)
		if err != nil {
			return err
		}
		if value < m.Target {
			return errors.New(NotCompleted)
		}
		inserted := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&ClaimRecord{MissionID: m.ID, UserID: userID, Period: period})
		if inserted.Error != nil {
			return inserted.Error
		}
		if inserted.RowsAffected == 0 {
			return errors.New(AlreadyClaimed)
		}
		bundle, err := reward.Deliverable(tx, m.Reward, nil)
		if err != nil {
			return err
		}
		if result, replayed, err = reward.Apply(tx, &reward.ApplyRequest{
			UserID:         userID,
			Bundle:         bundle,
			BizType:        BizReward,
			BizID:          strconv.FormatUint(m.ID, 10),
			IdempotencyKey: fmt.Sprintf("mission:%d:%s:%d", m.ID, period, userID),
			Remark:         m.Name,
		}); err != nil {
			switch err.Error() {
			case item.ItemNotFound, item.ItemDisabled, wallet.CurrencyNotFound, wallet.CurrencyDisabled:
				return errors.New(RewardUnavailable)
			case wallet.AmountInvalid, inventory.QuantityInvalid:
				return errors.New(RewardTooLarge)
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
	return &ClaimResult{Received: lines[0]}, nil
}
