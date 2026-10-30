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

package shop

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// codePattern 商品代码, 小写字母、数字和下划线; 拼上前缀后不能超过 stats.metric 的长度
var codePattern = regexp.MustCompile(`^[a-z0-9_]{1,32}$`)

// InvalidError 商品配置不合法, Error 返回的提示可以直接展示给管理员
type InvalidError struct {
	Reason string
}

func (e *InvalidError) Error() string { return e.Reason }

// RejectedError 不能购买, Error 返回的提示可以直接展示给玩家
type RejectedError struct {
	Reason string
}

func (e *RejectedError) Error() string { return e.Reason }

// Metric 商品的限购计数指标, 如 shop_buy:potion; 范围为 stats.ScopeAll
func Metric(code string) string {
	return metricPrefix + code
}

// Validate 保存商品前校验配置, 校验后合并价格和商品中的同类项
//
// 配置不合法时返回 *InvalidError; 其他错误为查库失败等内部错误。名称等文本字段需要调用方先去掉首尾空白。
func Validate(tx *gorm.DB, it *Item) error {
	if !codePattern.MatchString(it.Code) {
		return &InvalidError{Reason: CodeInvalid}
	}
	if n := utf8.RuneCountInString(it.Name); n < 1 || n > 64 {
		return &InvalidError{Reason: NameInvalid}
	}
	if utf8.RuneCountInString(it.Tab) > maxTabLength {
		return &InvalidError{Reason: TabInvalid}
	}
	var err error
	if it.Goods, err = checkBundle(tx, it.Goods); err != nil {
		return err
	}
	if it.Goods.IsEmpty() {
		return &InvalidError{Reason: GoodsEmpty}
	}
	if it.Price, err = checkBundle(tx, it.Price); err != nil {
		return err
	}
	if it.OriginalPrice, err = checkBundle(tx, it.OriginalPrice); err != nil {
		return err
	}
	for _, limit := range []int64{it.Limits.Total, it.Limits.Daily, it.Limits.Season} {
		if limit < 0 || limit > maxLimit {
			return &InvalidError{Reason: LimitInvalid}
		}
	}
	if it.Stock != nil && (*it.Stock < 0 || *it.Stock > maxLimit) {
		return &InvalidError{Reason: LimitInvalid}
	}
	if it.StartsAt != nil && it.EndsAt != nil && !it.EndsAt.After(*it.StartsAt) {
		return &InvalidError{Reason: TimeInvalid}
	}
	if it.Requirements == nil {
		it.Requirements = []condition.Condition{}
	}
	if err := condition.Validate(tx, it.Requirements); err != nil {
		var conditionErr *condition.InvalidError
		if errors.As(err, &conditionErr) {
			return &InvalidError{Reason: fmt.Sprintf(RequirementInvalid, conditionErr.Error())}
		}
		return err
	}
	return nil
}

// bundleReasons reward.Check 的提示 -> 商城的说法, 没有列出的原样展示
var bundleReasons = map[string]string{
	reward.LinesTooMany:     LinesTooMany,
	reward.AmountInvalid:    AmountInvalid,
	reward.CurrencyNotFound: CurrencyNotFound,
	reward.ItemsNotFound:    ItemsNotFound,
}

// checkBundle 用 reward.Check 校验价格或商品, 提示换成商城的说法; 已停用的货币和已下架的物品可以保存, 购买时才拒绝
func checkBundle(tx *gorm.DB, b reward.Bundle) (reward.Bundle, error) {
	normalized, err := reward.Check(tx, b)
	var invalid *reward.InvalidError
	if errors.As(err, &invalid) {
		if reason, ok := bundleReasons[invalid.Reason]; ok {
			return reward.Bundle{}, &InvalidError{Reason: reason}
		}
		return reward.Bundle{}, &InvalidError{Reason: invalid.Reason}
	}
	return normalized, err
}

// PurchaseRequest 一次购买
type PurchaseRequest struct {
	Code           string
	Quantity       int64          // 份数, 1 到 MaxQuantity
	Price          *reward.Bundle // 玩家看到的一份的价格, 不为空时与当前价格不一致则拒绝
	IdempotencyKey string         // 同一用户内唯一, 重复提交返回原订单
}

// Purchase 购买商品, 返回订单; 同一个幂等键重复提交时返回原订单, replayed 为 true
//
// 一个事务内检查上架时间、库存、购买条件和限购, 扣价格、发商品、写订单、累加限购计数、扣库存, 任何一步失败整体回滚;
// 不能购买时返回 *RejectedError, 余额或物品不足时返回 wallet.InsufficientBalance 或 inventory.InsufficientQuantity。
func Purchase(ctx context.Context, userID uint64, req *PurchaseRequest) (*OrderView, bool, error) {
	if req.Quantity < 1 || req.Quantity > MaxQuantity {
		return nil, false, errors.New(QuantityInvalid)
	}
	var (
		order    Order
		replayed bool
		results  []*reward.Result
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 同一用户的购买在咨询锁上排队, 幂等键和限购计数都不会被并发请求绕过
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", fmt.Sprintf("blindcrate:shop:%d", userID)).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND idempotency_key = ?", userID, req.IdempotencyKey).Limit(1).Find(&order).Error; err != nil {
			return err
		}
		if order.ID != 0 {
			if order.ItemCode != req.Code || order.Quantity != req.Quantity {
				return errors.New(IdempotencyConflict)
			}
			replayed = true
			return nil
		}

		var err error
		results, err = purchase(tx, userID, req, &order)
		return err
	})
	if err != nil {
		switch err.Error() {
		case item.ItemNotFound, item.ItemDisabled, wallet.CurrencyNotFound, wallet.CurrencyDisabled:
			return nil, false, &RejectedError{Reason: GoodsUnavailable}
		case wallet.AmountInvalid, inventory.QuantityInvalid:
			return nil, false, errors.New(AmountTooLarge)
		}
		return nil, false, err
	}
	for _, result := range results {
		result.Push(ctx)
	}
	views, err := newOrderViews(db.DB(ctx), []Order{order})
	if err != nil {
		return nil, false, err
	}
	return &views[0], replayed, nil
}

// purchase 在已加锁的事务中完成一次购买, 把订单写入 order, 返回扣价格和发商品的结果, 用于提交后推送
func purchase(tx *gorm.DB, userID uint64, req *PurchaseRequest, order *Order) ([]*reward.Result, error) {
	// 锁定商品, 与修改商品互斥, 同一商品的并发购买按顺序扣库存, 不会超卖
	var it Item
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", req.Code).First(&it).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(ItemNotFound)
		}
		return nil, err
	}
	now := time.Now()
	if !it.OnShelf(now) {
		return nil, &RejectedError{Reason: OffShelf}
	}
	if req.Price != nil {
		seen, err := req.Price.Normalize()
		if err != nil || !reflect.DeepEqual(seen, it.Price) {
			return nil, errors.New(PriceChanged)
		}
	}
	if it.Stock != nil && *it.Stock < req.Quantity {
		return nil, &RejectedError{Reason: SoldOut}
	}
	unmet, err := condition.FirstUnmet(tx, userID, it.Requirements)
	if err != nil {
		return nil, err
	}
	if unmet != nil {
		return nil, &RejectedError{Reason: fmt.Sprintf(RequirementUnmet, unmet.Text)}
	}
	if err := checkLimits(tx, userID, &it, req.Quantity, now); err != nil {
		return nil, err
	}

	*order = Order{
		UserID:         userID,
		IdempotencyKey: req.IdempotencyKey,
		ShopItemID:     it.ID,
		ItemCode:       it.Code,
		Name:           it.Name,
		Quantity:       req.Quantity,
		Price:          scale(it.Price, req.Quantity),
		Goods:          scale(it.Goods, req.Quantity),
		Status:         StatusPaid,
	}
	if err := tx.Create(order).Error; err != nil {
		// 咨询锁之外的兜底, 正常不会走到
		if db.IsUniqueViolation(err) {
			return nil, errors.New(DuplicateRequest)
		}
		return nil, err
	}
	bizID := strconv.FormatUint(order.ID, 10)
	paid, _, err := reward.Apply(tx, &reward.ApplyRequest{
		UserID:         userID,
		Bundle:         order.Price.Negate(),
		BizType:        wallet.BizShopBuy,
		BizID:          bizID,
		IdempotencyKey: "shop:" + bizID + ":pay",
		Remark:         order.Name,
	})
	if err != nil {
		return nil, err
	}
	granted, _, err := reward.Apply(tx, &reward.ApplyRequest{
		UserID:         userID,
		Bundle:         order.Goods,
		BizType:        wallet.BizShopGrant,
		BizID:          bizID,
		IdempotencyKey: "shop:" + bizID + ":grant",
		Remark:         order.Name,
	})
	if err != nil {
		return nil, err
	}
	if err := stats.Add(tx, userID, []string{stats.ScopeAll}, now, stats.Values{Metric(it.Code): req.Quantity}); err != nil {
		return nil, err
	}
	// 不限库存时 stock 为 NULL, 减去后仍为 NULL
	if err := tx.Model(&it).UpdateColumns(map[string]any{
		"sold":       gorm.Expr("sold + ?", req.Quantity),
		"stock":      gorm.Expr("stock - ?", req.Quantity),
		"updated_at": now,
	}).Error; err != nil {
		return nil, err
	}
	return []*reward.Result{paid, granted}, nil
}

// limitPeriod 一种限购和计数所在的统计周期
type limitPeriod struct {
	Type   string // 见 LimitTotal 等
	Label  string // 展示给玩家的名称
	Limit  int64
	Period string // 统计周期, 赛季限购在没有进行中的赛季时为空
}

// limitPeriods 商品设置了的限购; current 为 at 时进行中的赛季, 可以为空
func limitPeriods(it *Item, at time.Time, current *season.Season) []limitPeriod {
	var periods []limitPeriod
	if it.Limits.Total > 0 {
		periods = append(periods, limitPeriod{Type: LimitTotal, Label: "每人", Limit: it.Limits.Total, Period: stats.PeriodTotal})
	}
	if it.Limits.Daily > 0 {
		periods = append(periods, limitPeriod{Type: LimitDaily, Label: "每天", Limit: it.Limits.Daily, Period: stats.PeriodDay(at)})
	}
	if it.Limits.Season > 0 {
		lp := limitPeriod{Type: LimitSeason, Label: "每赛季", Limit: it.Limits.Season}
		if current != nil {
			lp.Period = season.Period(current.Code)
		}
		periods = append(periods, lp)
	}
	return periods
}

// checkLimits 加上本次购买的份数后不能超过任何一种限购; 赛季限购需要有进行中的赛季
func checkLimits(tx *gorm.DB, userID uint64, it *Item, quantity int64, at time.Time) error {
	var current *season.Season
	if it.Limits.Season > 0 {
		var err error
		if current, err = season.Current(tx, at); err != nil {
			return err
		}
		if current == nil {
			return &RejectedError{Reason: SeasonRequired}
		}
	}
	for _, lp := range limitPeriods(it, at, current) {
		bought, err := stats.Get(tx, userID, stats.ScopeAll, lp.Period, Metric(it.Code))
		if err != nil {
			return err
		}
		if bought+quantity > lp.Limit {
			return &RejectedError{Reason: fmt.Sprintf(LimitReached, lp.Label, lp.Limit, bought)}
		}
	}
	return nil
}

// scale 把一份的价格或商品乘以份数; 单行不超过单笔上限, 乘以份数不会溢出, 超出上限时由 reward.Apply 拒绝
func scale(b reward.Bundle, quantity int64) reward.Bundle {
	var scaled reward.Bundle
	for _, line := range b.Currencies {
		scaled.Currencies = append(scaled.Currencies, reward.CurrencyLine{Currency: line.Currency, Amount: line.Amount * quantity})
	}
	for _, line := range b.Items {
		scaled.Items = append(scaled.Items, reward.ItemLine{ItemID: line.ItemID, Quantity: line.Quantity * quantity})
	}
	return scaled
}

// Refund 管理员退款: 退还价格并收回商品, 恢复库存, 返回退款后的订单; 不恢复限购数量
//
// 价格中已下架或已删除的物品按回收价折成货币退还; 用户的货币或物品不足以收回商品时返回 RefundInsufficient。
func Refund(ctx context.Context, operatorID, orderID uint64) (*Order, error) {
	var (
		order  Order
		result *reward.Result
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(OrderNotFound)
			}
			return err
		}
		if order.Status != StatusPaid {
			return errors.New(OrderRefunded)
		}
		bundle, err := reward.Deliverable(tx, order.Price, nil)
		if err != nil {
			return err
		}
		bundle.Add(order.Goods.Negate())
		bizID := strconv.FormatUint(order.ID, 10)
		if result, _, err = reward.Apply(tx, &reward.ApplyRequest{
			UserID:         order.UserID,
			Bundle:         bundle,
			BizType:        wallet.BizShopRefund,
			BizID:          bizID,
			IdempotencyKey: "shop:" + bizID + ":refund",
			OperatorID:     operatorID,
			Remark:         order.Name,
		}); err != nil {
			switch err.Error() {
			case wallet.InsufficientBalance, inventory.InsufficientQuantity:
				return errors.New(RefundInsufficient)
			case item.ItemNotFound, item.ItemDisabled, wallet.CurrencyNotFound, wallet.CurrencyDisabled:
				return errors.New(RefundUnavailable)
			}
			return err
		}

		now := time.Now()
		order.Status = StatusRefunded
		order.RefundedAt = &now
		order.RefundedBy = operatorID
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		// 商品可能已删除, 用原生 SQL 绕过软删除
		return tx.Exec("UPDATE shop_items SET sold = sold - ?, stock = stock + ?, updated_at = ? WHERE id = ?",
			order.Quantity, order.Quantity, now, order.ShopItemID).Error
	})
	if err != nil {
		return nil, err
	}
	result.Push(ctx)
	return &order, nil
}

// ItemView 玩家看到的商品
type ItemView struct {
	Code          string               `json:"code"`
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	ImageURL      string               `json:"image_url"`
	Tab           string               `json:"tab"`
	Goods         []reward.Line        `json:"goods"`          // 一份商品
	Price         []reward.Line        `json:"price"`          // 一份的价格, 为空时免费
	OriginalPrice []reward.Line        `json:"original_price"` // 原价, 为空时不展示
	Requirements  []condition.Progress `json:"requirements"`   // 每个购买条件的达成情况
	Eligible      bool                 `json:"eligible"`       // 满足全部购买条件
	Limits        []LimitView          `json:"limits"`         // 设置了的限购和已购买的份数
	Stock         *int64               `json:"stock"`          // 全服剩余库存, 为空时不限
	EndsAt        *time.Time           `json:"ends_at"`        // 下架时间, 为空时不限
}

// LimitView 一种限购和用户已购买的份数
type LimitView struct {
	Type   string `json:"type"` // total、daily 或 season
	Limit  int64  `json:"limit"`
	Bought int64  `json:"bought"` // 赛季限购在没有进行中的赛季时为 0, 此时不能购买
}

// QueryItems 获取当前上架的商品, 按排序值和 ID 排列, 附带用户对购买条件的达成情况和已购买的份数
func QueryItems(ctx context.Context, userID uint64) ([]ItemView, error) {
	tx := db.DB(ctx)
	now := time.Now()
	var items []Item
	if err := tx.Where("is_enabled AND (starts_at IS NULL OR starts_at <= ?) AND (ends_at IS NULL OR ends_at > ?)", now, now).
		Order("sort_order, id").Find(&items).Error; err != nil {
		return nil, err
	}
	views := make([]ItemView, len(items))
	if len(items) == 0 {
		return views, nil
	}

	bundles := make([]reward.Bundle, 0, len(items)*3)
	for _, it := range items {
		bundles = append(bundles, it.Goods, it.Price, it.OriginalPrice)
	}
	lines, err := reward.Lines(tx, bundles...)
	if err != nil {
		return nil, err
	}
	current, err := season.Current(tx, now)
	if err != nil {
		return nil, err
	}
	bought, err := queryBought(tx, userID, items, now, current)
	if err != nil {
		return nil, err
	}

	for i := range items {
		it := &items[i]
		progress, err := condition.Evaluate(tx, userID, it.Requirements)
		if err != nil {
			return nil, err
		}
		eligible := true
		for _, p := range progress {
			eligible = eligible && p.Met
		}
		limits := []LimitView{}
		for _, lp := range limitPeriods(it, now, current) {
			limits = append(limits, LimitView{Type: lp.Type, Limit: lp.Limit, Bought: bought[boughtKey{Metric(it.Code), lp.Period}]})
		}
		views[i] = ItemView{
			Code:          it.Code,
			Name:          it.Name,
			Description:   it.Description,
			ImageURL:      it.ImageURL,
			Tab:           it.Tab,
			Goods:         lines[i*3],
			Price:         lines[i*3+1],
			OriginalPrice: lines[i*3+2],
			Requirements:  progress,
			Eligible:      eligible,
			Limits:        limits,
			Stock:         it.Stock,
			EndsAt:        it.EndsAt,
		}
	}
	return views, nil
}

// boughtKey 已购买份数的索引
type boughtKey struct {
	metric, period string
}

// queryBought 一次查出用户对设置了限购的商品在累计、at 当天和当前赛季已购买的份数
func queryBought(tx *gorm.DB, userID uint64, items []Item, at time.Time, current *season.Season) (map[boughtKey]int64, error) {
	metrics := make([]string, 0, len(items))
	for _, it := range items {
		if it.Limits != (Limits{}) {
			metrics = append(metrics, Metric(it.Code))
		}
	}
	bought := map[boughtKey]int64{}
	if len(metrics) == 0 {
		return bought, nil
	}
	periods := []string{stats.PeriodTotal, stats.PeriodDay(at)}
	if current != nil {
		periods = append(periods, season.Period(current.Code))
	}
	var rows []stats.Stat
	if err := tx.Select("period, metric, value").
		Where("user_id = ? AND scope = ? AND period IN ? AND metric IN ?", userID, stats.ScopeAll, periods, metrics).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		bought[boughtKey{row.Metric, row.Period}] = row.Value
	}
	return bought, nil
}

// OrderView 玩家看到的订单, 价格和商品带名称
type OrderView struct {
	ID         uint64        `json:"id"`
	ItemCode   string        `json:"item_code"`
	Name       string        `json:"name"`
	Quantity   int64         `json:"quantity"`
	Price      []reward.Line `json:"price"` // 实际支付的总价
	Goods      []reward.Line `json:"goods"` // 实际得到的全部商品
	Status     string        `json:"status"`
	RefundedAt *time.Time    `json:"refunded_at"`
	CreatedAt  time.Time     `json:"created_at"`
}

// QueryUserOrders 分页获取用户的订单, 从新到旧
func QueryUserOrders(ctx context.Context, userID uint64, offset, limit int) (int64, []OrderView, error) {
	tx := db.DB(ctx)
	query := tx.Model(&Order{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var orders []Order
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return 0, nil, err
	}
	views, err := newOrderViews(tx, orders)
	if err != nil {
		return 0, nil, err
	}
	return total, views, nil
}

// newOrderViews 订单带上价格和商品的名称
func newOrderViews(tx *gorm.DB, orders []Order) ([]OrderView, error) {
	bundles := make([]reward.Bundle, 0, len(orders)*2)
	for _, o := range orders {
		bundles = append(bundles, o.Price, o.Goods)
	}
	lines, err := reward.Lines(tx, bundles...)
	if err != nil {
		return nil, err
	}
	views := make([]OrderView, len(orders))
	for i, o := range orders {
		views[i] = OrderView{
			ID:         o.ID,
			ItemCode:   o.ItemCode,
			Name:       o.Name,
			Quantity:   o.Quantity,
			Price:      lines[i*2],
			Goods:      lines[i*2+1],
			Status:     o.Status,
			RefundedAt: o.RefundedAt,
			CreatedAt:  o.CreatedAt,
		}
	}
	return views, nil
}
