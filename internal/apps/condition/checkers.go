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
	"cmp"
	"errors"
	"fmt"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/params"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"gorm.io/gorm"
)

func init() {
	// 条件组的参数是一组条件, 要在注册用到它的条件类型之前登记
	params.RegisterKind[[]Condition](KindConditions)

	Register(TypeMinBalance, "最低余额", "可用余额不少于指定金额, 只检查不扣除; 入场费请配置为入场代价", checkMinBalance)
	Register(TypeMinStat, "统计数据", "一个周期内在统计范围内的统计数据不少于指定值, 如累计完成 10 局排位、本周胜利 3 次", checkMinStat)
	Register(TypeStatLimit, "统计上限", "一个周期内在统计范围内的统计数据达到指定值后不再满足, 如每天最多 5 局、累计最多 20 局的新手图、每赛季最多 3 次", checkStatLimit)
	Register(TypeHasItem, "持有物品", "背包中持有的指定物品不少于指定数量, 只检查不消耗; 需要消耗时请配置为入场代价", checkHasItem)
	Register(TypeAccountAge, "注册天数", "注册满指定天数", checkAccountAge)
	Register(TypeEmailVerified, "验证邮箱", "已验证邮箱", checkEmailVerified)
	Register(TypeTimeWindow, "时间窗口", "现在在指定的时间内: 起止时间、星期几、每天几点到几点, 填了的几项需要同时满足; 用于限时地图、周末活动商品", checkTimeWindow)
	Register(TypeAnyOf, "满足其中几个", "列出的条件中至少满足指定的个数, 如持有门票或段位分够高", checkAnyOf)
	Register(TypeNoneOf, "都不满足", "列出的条件一个都不满足时才算满足, 如还没有持有某件物品", checkNoneOf)
}

// minBalanceParams 最低余额的参数
type minBalanceParams struct {
	Currency string `json:"currency" label:"货币" binding:"required,max=32" input:"currency"`
	Amount   int64  `json:"amount" label:"金额" help:"不含冻结, 按最小单位填写" binding:"min=1,max=1000000000000"`
}

// Validate 货币必须存在
func (p *minBalanceParams) Validate(tx *gorm.DB) (string, error) {
	var currency wallet.Currency
	if err := currency.Exact(tx, p.Currency); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return wallet.CurrencyNotFound, nil
		}
		return "", err
	}
	return "", nil
}

func checkMinBalance(tx *gorm.DB, userID uint64, p *minBalanceParams) (*Progress, error) {
	var balance wallet.Balance
	if err := tx.Where("user_id = ? AND currency = ?", userID, p.Currency).Limit(1).Find(&balance).Error; err != nil {
		return nil, err
	}
	// 货币被删除时按代码展示
	currency := wallet.Currency{Code: p.Currency, Name: p.Currency}
	if err := tx.Where("code = ?", p.Currency).Limit(1).Find(&currency).Error; err != nil {
		return nil, err
	}
	return &Progress{
		Met:     balance.Balance >= p.Amount,
		Current: balance.Balance,
		Target:  p.Amount,
		Text:    fmt.Sprintf("%s不少于 %s(当前 %s)", currency.Name, formatAmount(p.Amount, currency.Decimals), formatAmount(balance.Balance, currency.Decimals)),
	}, nil
}

// minStatParams 统计数据的参数
type minStatParams struct {
	Metric string `json:"metric" label:"统计数据" binding:"required,max=64" input:"metric"`
	Scope  string `json:"scope" label:"统计范围" help:"all 为全部对局, mode:<模式> 为某种模式, venue:<地图> 为某张地图, venue:<地图>:<模式> 为某张地图的某种模式" binding:"required,max=64" input:"scope"`
	Period string `json:"period" label:"统计周期" help:"只算这个周期内的数据, 留空为累计; 本赛季只在赛季进行中有数据" binding:"max=16" input:"period"`
	Count  int64  `json:"count" label:"不少于" binding:"min=1,max=1000000"`
}

// Validate 指标和周期必须已注册, 统计范围的格式必须正确
func (p *minStatParams) Validate(*gorm.DB) (string, error) {
	return validateStat(p.Metric, p.Scope, p.Period), nil
}

func checkMinStat(tx *gorm.DB, userID uint64, p *minStatParams) (*Progress, error) {
	value, name, err := periodStat(tx, userID, p.Scope, p.Period, p.Metric)
	if err != nil {
		return nil, err
	}
	// 累计的不写周期, 如 "排位对局数达到 10(3/10)"; 其他的写在最前面, 如 "本周排位对局数达到 10(3/10)"
	period := ""
	if p.Period != "" && p.Period != stats.PeriodTotal {
		period = stats.PeriodLabel(p.Period)
	}
	return &Progress{
		Met:     value >= p.Count,
		Current: value,
		Target:  p.Count,
		Text:    fmt.Sprintf("%s%s%s达到 %d(%d/%d)", period, name, metricLabel(p.Metric), p.Count, value, p.Count),
	}, nil
}

// statLimitParams 统计上限的参数
type statLimitParams struct {
	Metric string `json:"metric" label:"统计数据" binding:"required,max=64" input:"metric"`
	Scope  string `json:"scope" label:"统计范围" help:"all 为全部对局, mode:<模式> 为某种模式, venue:<地图> 为某张地图, venue:<地图>:<模式> 为某张地图的某种模式" binding:"required,max=64" input:"scope"`
	Period string `json:"period" label:"统计周期" help:"只算这个周期内的数据, 留空为累计; 本赛季只在赛季进行中有数据" binding:"max=16" input:"period"`
	Count  int64  `json:"count" label:"最多" help:"达到这个数之后不再满足" binding:"min=1,max=1000000"`
}

// Validate 指标和周期必须已注册, 统计范围的格式必须正确
func (p *statLimitParams) Validate(*gorm.DB) (string, error) {
	return validateStat(p.Metric, p.Scope, p.Period), nil
}

func checkStatLimit(tx *gorm.DB, userID uint64, p *statLimitParams) (*Progress, error) {
	value, name, err := periodStat(tx, userID, p.Scope, p.Period, p.Metric)
	if err != nil {
		return nil, err
	}
	return &Progress{
		Met:     value < p.Count,
		Current: value,
		Target:  p.Count,
		Text:    fmt.Sprintf("%s%s%s最多 %d(%d/%d)", stats.PeriodLabel(periodOf(p.Period)), name, metricLabel(p.Metric), p.Count, value, p.Count),
	}, nil
}

// hasItemParams 持有物品的参数
type hasItemParams struct {
	ItemID   uint64 `json:"item_id" label:"物品" binding:"required" input:"item"`
	Quantity int64  `json:"quantity" label:"数量" binding:"min=1,max=1000000"`
}

// Validate 物品必须存在
func (p *hasItemParams) Validate(tx *gorm.DB) (string, error) {
	var it item.Item
	if err := it.Exact(tx, p.ItemID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item.ItemNotFound, nil
		}
		return "", err
	}
	return "", nil
}

func checkHasItem(tx *gorm.DB, userID uint64, p *hasItemParams) (*Progress, error) {
	var owned inventory.UserItem
	if err := tx.Where("user_id = ? AND item_id = ?", userID, p.ItemID).Limit(1).Find(&owned).Error; err != nil {
		return nil, err
	}
	// 物品被删除时按占位物品展示
	items, err := item.Resolve(tx, []uint64{p.ItemID})
	if err != nil {
		return nil, err
	}
	return &Progress{
		Met:     owned.Quantity >= p.Quantity,
		Current: owned.Quantity,
		Target:  p.Quantity,
		Text:    fmt.Sprintf("持有%s %d 个(%d/%d)", items[p.ItemID].Name, p.Quantity, owned.Quantity, p.Quantity),
	}, nil
}

// accountAgeParams 注册天数的参数
type accountAgeParams struct {
	Days int64 `json:"days" label:"天数" binding:"min=1,max=36500"`
}

func checkAccountAge(tx *gorm.DB, userID uint64, p *accountAgeParams) (*Progress, error) {
	var user auth.User
	if err := user.Exact(tx, userID); err != nil {
		return nil, err
	}
	days := int64(time.Since(user.CreatedAt) / (24 * time.Hour))
	return &Progress{
		Met:     days >= p.Days,
		Current: days,
		Target:  p.Days,
		Text:    fmt.Sprintf("注册满 %d 天(%d/%d)", p.Days, days, p.Days),
	}, nil
}

// emailVerifiedParams 验证邮箱没有参数
type emailVerifiedParams struct{}

func checkEmailVerified(tx *gorm.DB, userID uint64, _ *emailVerifiedParams) (*Progress, error) {
	var user auth.User
	if err := user.Exact(tx, userID); err != nil {
		return nil, err
	}
	return &Progress{Met: user.EmailVerifiedAt != nil, Text: "已验证邮箱"}, nil
}

// validateStat 指标或周期未注册、统计范围的格式不正确时返回原因
func validateStat(metric, scope, period string) string {
	if stats.FindMetric(metric) == nil {
		return MetricNotFound
	}
	if !stats.ValidScope(scope) {
		return ScopeInvalid
	}
	if !stats.HasPeriod(periodOf(period)) {
		return PeriodNotFound
	}
	return ""
}

// periodOf 参数中的周期名称, 留空为累计
func periodOf(period string) string {
	return cmp.Or(period, stats.PeriodTotal)
}

// metricLabel 指标的展示名称, 指标已不再注册时展示 key
func metricLabel(metric string) string {
	if m := stats.FindMetric(metric); m != nil {
		return m.Label
	}
	return metric
}

// periodStat 获取用户现在所在的周期内, 在统计范围内的一个指标, 以及范围的展示名称; period 为周期的名称, 现在没有这种周期(如不在赛季中)时为 0
func periodStat(tx *gorm.DB, userID uint64, scope, period, metric string) (int64, string, error) {
	key, ok, err := stats.PeriodKey(tx, periodOf(period), time.Now())
	if err != nil {
		return 0, "", err
	}
	if !ok {
		return 0, "", fmt.Errorf("condition: unknown period %s", period)
	}
	var value int64
	if key != "" {
		if value, err = stats.Get(tx, userID, scope, key, metric); err != nil {
			return 0, "", err
		}
	}
	name, err := ScopeName(tx, scope)
	if err != nil {
		return 0, "", err
	}
	return value, name, nil
}
