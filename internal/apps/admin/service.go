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

package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/apps/ranked"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/mail"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// likeEscaper 转义 LIKE 通配符, 用户名里常见的下划线也是通配符
var likeEscaper = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`)

// QueryUsersList 获取用户列表
func QueryUsersList(ctx context.Context, req *ListUsersRequest) (int64, []auth.User, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&auth.User{})
	if req.Keyword != "" {
		keyword := likeEscaper.Replace(strings.ToLower(req.Keyword)) + "%"
		query = query.Where("LOWER(username) LIKE ? OR LOWER(email) LIKE ?", keyword, keyword)
	}
	if req.IsAdmin != nil {
		query = query.Where("is_admin = ?", *req.IsAdmin)
	}
	if req.IsDisabled != nil {
		query = query.Where("is_disabled = ?", *req.IsDisabled)
	}
	if req.IsBanned != nil {
		if *req.IsBanned {
			query = query.Where("banned_until > ?", time.Now())
		} else {
			query = query.Where("banned_until IS NULL OR banned_until <= ?", time.Now())
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	users := []auth.User{}
	if err := query.Order("id DESC").Offset(offset).Limit(req.Size).Find(&users).Error; err != nil {
		return 0, nil, err
	}

	return total, users, nil
}

// updateOtherUser 管理员修改其他用户的字段, 不允许操作自己
func updateOtherUser(ctx context.Context, operatorID, userID uint64, values map[string]interface{}) error {
	if operatorID == userID {
		return errors.New(CannotOperateSelf)
	}
	tx := db.DB(ctx).Model(&auth.User{}).Where("id = ?", userID).Updates(values)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return errors.New(auth.UserNotFound)
	}
	return nil
}

// doBanUser 封禁用户到指定时间
func doBanUser(ctx context.Context, operatorID, userID uint64, until time.Time, reason string) error {
	if !until.After(time.Now()) {
		return errors.New(BanUntilInvalid)
	}
	return updateOtherUser(ctx, operatorID, userID, map[string]interface{}{"banned_until": until, "ban_reason": reason})
}

// doUnbanUser 解除封禁
func doUnbanUser(ctx context.Context, operatorID, userID uint64) error {
	return updateOtherUser(ctx, operatorID, userID, map[string]interface{}{"banned_until": nil, "ban_reason": ""})
}

// doSetUserDisabled 停用或启用用户
func doSetUserDisabled(ctx context.Context, operatorID, userID uint64, disabled bool) error {
	return updateOtherUser(ctx, operatorID, userID, map[string]interface{}{"is_disabled": disabled})
}

// AuditLogItem 管理后台展示的操作日志, 带上操作人用户名
type AuditLogItem struct {
	ID            uint64          `json:"id"`
	ActorID       uint64          `json:"actor_id"`
	ActorUsername string          `json:"actor_username"` // 操作人用户名, 未登录或用户已不存在时为空
	Scope         string          `json:"scope"`
	Action        string          `json:"action"`
	TargetType    string          `json:"target_type"`
	TargetID      uint64          `json:"target_id"`
	Detail        json.RawMessage `json:"detail" swaggertype:"object"`
	IP            string          `json:"ip"`
	UserAgent     string          `json:"user_agent"`
	CreatedAt     time.Time       `json:"created_at"`
}

// QueryAuditLogs 获取操作日志列表
func QueryAuditLogs(ctx context.Context, req *ListAuditLogsRequest) (int64, []AuditLogItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&audit.Log{})
	if req.ActorID != nil {
		query = query.Where("audit_logs.actor_id = ?", *req.ActorID)
	}
	if req.Scope != "" {
		query = query.Where("audit_logs.scope = ?", req.Scope)
	}
	if req.Action != "" {
		query = query.Where("audit_logs.action = ?", req.Action)
	}
	if req.TargetType != "" {
		query = query.Where("audit_logs.target_type = ?", req.TargetType)
	}
	if req.TargetID != nil {
		query = query.Where("audit_logs.target_id = ?", *req.TargetID)
	}
	if req.StartTime != nil {
		query = query.Where("audit_logs.created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("audit_logs.created_at < ?", *req.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	logs := []AuditLogItem{}
	if err := query.
		Select("audit_logs.*, COALESCE(users.username, '') AS actor_username").
		Joins("LEFT JOIN users ON users.id = audit_logs.actor_id").
		Order("audit_logs.id DESC").Offset(offset).Limit(req.Size).
		Scan(&logs).Error; err != nil {
		return 0, nil, err
	}

	return total, logs, nil
}

// WalletTransactionItem 管理后台展示的余额流水, 带上用户和操作人的用户名
type WalletTransactionItem struct {
	ID               uint64    `json:"id"`
	UserID           uint64    `json:"user_id"`
	Username         string    `json:"username"` // 用户已不存在时为空
	Currency         string    `json:"currency"`
	Amount           int64     `json:"amount"`
	FrozenAmount     int64     `json:"frozen_amount"`
	BalanceAfter     int64     `json:"balance_after"`
	FrozenAfter      int64     `json:"frozen_after"`
	BizType          string    `json:"biz_type"`
	BizID            string    `json:"biz_id"`
	IdempotencyKey   string    `json:"idempotency_key"`
	OperatorID       uint64    `json:"operator_id"`
	OperatorUsername string    `json:"operator_username"` // 系统操作或用户已不存在时为空
	Remark           string    `json:"remark"`
	CreatedAt        time.Time `json:"created_at"`
}

// QueryWalletTransactions 获取全部用户的余额流水
func QueryWalletTransactions(ctx context.Context, req *ListWalletTransactionsRequest) (int64, []WalletTransactionItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&wallet.Transaction{})
	if req.UserID != nil {
		query = query.Where("wallet_transactions.user_id = ?", *req.UserID)
	}
	if req.OperatorID != nil {
		query = query.Where("wallet_transactions.operator_id = ?", *req.OperatorID)
	}
	if req.Currency != "" {
		query = query.Where("wallet_transactions.currency = ?", req.Currency)
	}
	if req.BizType != "" {
		query = query.Where("wallet_transactions.biz_type = ?", req.BizType)
	}
	if req.BizID != "" {
		query = query.Where("wallet_transactions.biz_id = ?", req.BizID)
	}
	if req.StartTime != nil {
		query = query.Where("wallet_transactions.created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("wallet_transactions.created_at < ?", *req.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	items := []WalletTransactionItem{}
	if err := query.
		Select("wallet_transactions.*, COALESCE(users.username, '') AS username, COALESCE(operators.username, '') AS operator_username").
		Joins("LEFT JOIN users ON users.id = wallet_transactions.user_id").
		Joins("LEFT JOIN users AS operators ON operators.id = wallet_transactions.operator_id").
		Order("wallet_transactions.id DESC").Offset(offset).Limit(req.Size).
		Scan(&items).Error; err != nil {
		return 0, nil, err
	}

	return total, items, nil
}

// queryUserBalances 获取指定用户的各货币余额(含已停用货币)
func queryUserBalances(ctx context.Context, userID uint64) ([]wallet.BalanceItem, error) {
	var user auth.User
	if err := user.Exact(db.DB(ctx), userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(auth.UserNotFound)
		}
		return nil, err
	}
	return wallet.QueryBalances(ctx, userID, true)
}

// doAdjustUserBalance 管理员给其他用户调账, 不允许给自己调账
//
// 幂等键由前端在每次打开调账表单时生成, 防止重复点击导致重复入账。
func doAdjustUserBalance(ctx context.Context, operatorID, userID uint64, req *AdjustUserBalanceRequest) (*wallet.Transaction, bool, error) {
	if operatorID == userID {
		return nil, false, errors.New(CannotOperateSelf)
	}
	remark, err := normalizeRemark(req.Remark)
	if err != nil {
		return nil, false, err
	}

	var (
		txn      *wallet.Transaction
		replayed bool
	)
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var user auth.User
		if err := user.Exact(tx, userID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(auth.UserNotFound)
			}
			return err
		}

		var err error
		txn, replayed, err = wallet.Change(tx, &wallet.ChangeRequest{
			UserID:         userID,
			Currency:       req.Currency,
			Amount:         req.Amount,
			BizType:        wallet.BizAdminAdjust,
			IdempotencyKey: fmt.Sprintf("%s:%d:%s", wallet.BizAdminAdjust, operatorID, req.IdempotencyKey),
			OperatorID:     operatorID,
			Remark:         remark,
		})
		return err
	})
	if err != nil {
		return nil, false, err
	}
	return txn, replayed, nil
}

// normalizeRemark 去除首尾空白并校验调整备注, 用户可在流水中看到
func normalizeRemark(remark string) (string, error) {
	remark = strings.TrimSpace(remark)
	if n := utf8.RuneCountInString(remark); n < 1 || n > 255 || strings.ContainsRune(remark, 0) {
		return "", errors.New(RemarkInvalid)
	}
	return remark, nil
}

// holderCountSQL 持有某物品(数量大于 0)的用户数
const holderCountSQL = "SELECT COUNT(*) FROM inventory_items WHERE inventory_items.item_id = items.id AND inventory_items.quantity > 0"

// ItemListItem 管理后台展示的物品, 带上持有人数
type ItemListItem struct {
	item.Item
	HolderCount int64 `json:"holder_count"` // 持有该物品的用户数, 删除前用于提示
}

// QueryItems 获取未删除的物品列表
func QueryItems(ctx context.Context, req *ListItemsRequest) (int64, []ItemListItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&item.Item{})
	if req.Keyword != "" {
		query = query.Where("name ILIKE ?", "%"+likeEscaper.Replace(req.Keyword)+"%")
	}
	if req.Quality != nil {
		query = query.Where("quality = ?", *req.Quality)
	}
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}
	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	items := []ItemListItem{}
	if err := query.Select("items.*, (" + holderCountSQL + ") AS holder_count").
		Order("id DESC").Offset(offset).Limit(req.Size).
		Scan(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}

// doCreateItem 新建物品
func doCreateItem(ctx context.Context, req *ItemRequest) (*item.Item, error) {
	target := &item.Item{}
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := applyItem(tx, target, req); err != nil {
			return err
		}
		return tx.Create(target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(item.ItemNameTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateItem 修改物品, 返回修改后的物品和本次修改的字段, 没有任何字段变化时不写库
func doUpdateItem(ctx context.Context, itemID uint64, req *ItemRequest) (*item.Item, map[string]interface{}, error) {
	var (
		target  item.Item
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockItem(tx, &target, itemID); err != nil {
			return err
		}
		old := target
		if err := applyItem(tx, &target, req); err != nil {
			return err
		}
		changes = itemChanges(&old, &target)
		if len(changes) == 0 {
			return nil
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, nil, errors.New(item.ItemNameTaken)
		}
		return nil, nil, err
	}
	return &target, changes, nil
}

// doDeleteItem 软删除物品, 返回被删除的物品和当时的持有人数
//
// 已持有的玩家背包保留该物品, 之后按占位物品展示和结算; 物品 ID 不会被新物品复用。
func doDeleteItem(ctx context.Context, itemID uint64) (*item.Item, int64, error) {
	var (
		target  item.Item
		holders int64
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockItem(tx, &target, itemID); err != nil {
			return err
		}
		if err := tx.Model(&inventory.UserItem{}).Where("item_id = ? AND quantity > 0", itemID).Count(&holders).Error; err != nil {
			return err
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, 0, err
	}
	return &target, holders, nil
}

// lockItem 锁定未删除的物品, 防止并发修改和删除互相覆盖
func lockItem(tx *gorm.DB, target *item.Item, itemID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), itemID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(item.ItemNotFound)
		}
		return err
	}
	return nil
}

// applyItem 校验请求并写入物品字段, 品质、类别和卖出货币必须存在, 已停用的货币也可以选
func applyItem(tx *gorm.DB, target *item.Item, req *ItemRequest) error {
	name := strings.TrimSpace(req.Name)
	if n := utf8.RuneCountInString(name); n < 1 || n > 64 {
		return errors.New(item.ItemNameInvalid)
	}
	if err := item.CheckQuality(tx, req.Quality); err != nil {
		return err
	}
	if item.FindCategory(req.Category) == nil {
		return errors.New(item.CategoryNotFound)
	}
	var currency wallet.Currency
	if err := currency.Exact(tx, req.SellCurrency); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(wallet.CurrencyNotFound)
		}
		return err
	}

	target.Name = name
	target.Description = strings.TrimSpace(req.Description)
	target.Quality = req.Quality
	target.Category = req.Category
	target.ImageURL = req.ImageURL
	target.SellCurrency = req.SellCurrency
	target.SellPrice = req.SellPrice
	target.Width = req.Width
	target.Height = req.Height
	target.IsEnabled = req.IsEnabled
	return nil
}

// itemFields 物品可修改的字段, 用于操作日志
func itemFields(target *item.Item) map[string]interface{} {
	return map[string]interface{}{
		"name":          target.Name,
		"description":   target.Description,
		"quality":       target.Quality,
		"category":      target.Category,
		"image_url":     target.ImageURL,
		"sell_currency": target.SellCurrency,
		"sell_price":    target.SellPrice,
		"width":         target.Width,
		"height":        target.Height,
		"is_enabled":    target.IsEnabled,
	}
}

// itemChanges 本次修改了哪些字段, 用于操作日志
func itemChanges(old, next *item.Item) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := itemFields(next)
	for key, oldValue := range itemFields(old) {
		if newValue := nextFields[key]; oldValue != newValue {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}

// qualityItemCountSQL 使用某品质的未删除物品数
const qualityItemCountSQL = "SELECT COUNT(*) FROM items WHERE items.quality = qualities.level AND items.deleted_at IS NULL"

// QualityListItem 管理后台展示的品质, 带上使用该品质的物品数
type QualityListItem struct {
	item.Quality
	ItemCount int64 `json:"item_count"` // 使用该品质的未删除物品数, 不为 0 时不能删除
}

// queryQualities 获取全部品质, 按排序值和等级排列
func queryQualities(ctx context.Context) ([]QualityListItem, error) {
	qualities := []QualityListItem{}
	if err := db.DB(ctx).Model(&item.Quality{}).
		Select("qualities.*, (" + qualityItemCountSQL + ") AS item_count").
		Order("sort_order, level").Scan(&qualities).Error; err != nil {
		return nil, err
	}
	return qualities, nil
}

// doCreateQuality 新建品质
func doCreateQuality(ctx context.Context, req *CreateQualityRequest) (*item.Quality, error) {
	target := &item.Quality{Level: req.Level, Code: req.Code}
	if err := applyQuality(target, &req.QualityRequest); err != nil {
		return nil, err
	}
	if err := db.DB(ctx).Create(target).Error; err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(item.QualityTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateQuality 修改品质, 返回修改后的品质和本次修改的字段, 没有任何字段变化时不写库
func doUpdateQuality(ctx context.Context, level int16, req *QualityRequest) (*item.Quality, map[string]interface{}, error) {
	var (
		target  item.Quality
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockQuality(tx, &target, level); err != nil {
			return err
		}
		old := target
		if err := applyQuality(&target, req); err != nil {
			return err
		}
		changes = qualityChanges(&old, &target)
		if len(changes) == 0 {
			return nil
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return &target, changes, nil
}

// doDeleteQuality 删除品质, 还有未删除的物品使用该品质时返回 QualityInUse
//
// 保存物品时会对品质加共享锁, 这里先锁住品质再统计, 不会漏掉并发保存的物品。
func doDeleteQuality(ctx context.Context, level int16) (*item.Quality, error) {
	var target item.Quality
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockQuality(tx, &target, level); err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&item.Item{}).Where("quality = ?", level).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New(item.QualityInUse)
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// lockQuality 锁定品质, 防止并发修改和删除互相覆盖
func lockQuality(tx *gorm.DB, target *item.Quality, level int16) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), level); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(item.QualityNotFound)
		}
		return err
	}
	return nil
}

// applyQuality 写入品质字段并校验, 颜色统一为小写
func applyQuality(target *item.Quality, req *QualityRequest) error {
	target.Name = strings.TrimSpace(req.Name)
	target.Color = strings.ToLower(req.Color)
	target.Effect = req.Effect
	target.ScanMs = req.ScanMs
	target.SortOrder = req.SortOrder
	return item.ValidateQuality(target)
}

// qualityFields 品质的字段, 用于操作日志
func qualityFields(target *item.Quality) map[string]interface{} {
	return map[string]interface{}{
		"code":       target.Code,
		"name":       target.Name,
		"color":      target.Color,
		"effect":     target.Effect,
		"scan_ms":    target.ScanMs,
		"sort_order": target.SortOrder,
	}
}

// qualityChanges 本次修改了哪些字段, 用于操作日志
func qualityChanges(old, next *item.Quality) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := qualityFields(next)
	for key, oldValue := range qualityFields(old) {
		if newValue := nextFields[key]; oldValue != newValue {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}

// doCreateCurrency 新建货币
func doCreateCurrency(ctx context.Context, req *CreateCurrencyRequest) (*wallet.Currency, error) {
	target := &wallet.Currency{Code: req.Code}
	applyCurrency(target, &req.CurrencyRequest)
	if err := wallet.ValidateCurrency(target); err != nil {
		return nil, err
	}
	if err := db.DB(ctx).Create(target).Error; err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(wallet.CurrencyTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateCurrency 修改货币, 返回修改后的货币和本次修改的字段, 没有任何字段变化时不写库
//
// 金额按最小单位存储, 小数位只影响展示; 货币已被使用时修改小数位会让现有金额整体变大或变小, 因此只能修改未使用的货币。
func doUpdateCurrency(ctx context.Context, code string, req *CurrencyRequest) (*wallet.Currency, map[string]interface{}, error) {
	var (
		target  wallet.Currency
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), code); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(wallet.CurrencyNotFound)
			}
			return err
		}
		old := target
		applyCurrency(&target, req)
		if err := wallet.ValidateCurrency(&target); err != nil {
			return err
		}
		if target.Decimals != old.Decimals {
			used, err := currencyUsed(tx, code)
			if err != nil {
				return err
			}
			if used {
				return errors.New(wallet.CurrencyInUse)
			}
		}
		if old.IsEnabled && !target.IsEnabled {
			if err := wallet.CheckDisable(tx, code); err != nil {
				return err
			}
		}
		changes = currencyChanges(&old, &target)
		if len(changes) == 0 {
			return nil
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return &target, changes, nil
}

// currencyUsed 货币是否已被余额、物品、地图或掉落池使用, 已删除的物品和地图也算
//
// 对局、出售单和流水都会先写余额, 有余额行就说明已经产生过金额。
func currencyUsed(tx *gorm.DB, code string) (bool, error) {
	var used bool
	err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM wallet_balances WHERE currency = @code)
		OR EXISTS (SELECT 1 FROM items WHERE sell_currency = @code)
		OR EXISTS (SELECT 1 FROM venues WHERE bid_currency = @code)
		OR EXISTS (SELECT 1 FROM loot_pool_entries WHERE currency = @code)`, map[string]interface{}{"code": code}).
		Scan(&used).Error
	return used, err
}

// applyCurrency 写入货币字段, 名称去掉首尾空白
func applyCurrency(target *wallet.Currency, req *CurrencyRequest) {
	target.Name = strings.TrimSpace(req.Name)
	target.Decimals = req.Decimals
	target.SortOrder = req.SortOrder
	target.IsEnabled = req.IsEnabled
}

// currencyFields 货币的字段, 用于操作日志
func currencyFields(target *wallet.Currency) map[string]interface{} {
	return map[string]interface{}{
		"code":       target.Code,
		"name":       target.Name,
		"decimals":   target.Decimals,
		"sort_order": target.SortOrder,
		"is_enabled": target.IsEnabled,
	}
}

// currencyChanges 本次修改了哪些字段, 用于操作日志
func currencyChanges(old, next *wallet.Currency) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := currencyFields(next)
	for key, oldValue := range currencyFields(old) {
		if newValue := nextFields[key]; oldValue != newValue {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}

// lootPoolCodePattern 掉落池代码, 小写字母、数字和下划线
var lootPoolCodePattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

// entryCountSQL 掉落池的条目数
const entryCountSQL = "SELECT COUNT(*) FROM loot_pool_entries WHERE loot_pool_entries.pool_id = loot_pools.id"

// LootPoolListItem 管理后台展示的掉落池, 带上条目数
type LootPoolListItem struct {
	loot.Pool
	EntryCount int64 `json:"entry_count"`
}

// QueryLootPools 获取掉落池列表
func QueryLootPools(ctx context.Context, req *ListLootPoolsRequest) (int64, []LootPoolListItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&loot.Pool{})
	if req.Keyword != "" {
		keyword := "%" + likeEscaper.Replace(req.Keyword) + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", keyword, keyword)
	}
	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	pools := []LootPoolListItem{}
	if err := query.Select("loot_pools.*, (" + entryCountSQL + ") AS entry_count").
		Order("id DESC").Offset(offset).Limit(req.Size).
		Scan(&pools).Error; err != nil {
		return 0, nil, err
	}
	return total, pools, nil
}

// doCreateLootPool 新建掉落池, 新建后没有条目
func doCreateLootPool(ctx context.Context, req *LootPoolRequest) (*loot.Pool, error) {
	target := &loot.Pool{}
	if err := applyLootPool(target, req); err != nil {
		return nil, err
	}
	if err := db.DB(ctx).Create(target).Error; err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(loot.PoolCodeTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateLootPool 修改掉落池, 返回修改后的掉落池和本次修改的字段, 没有任何字段变化时不写库
//
// 修改代码后, 仍使用旧代码的玩法会找不到掉落池。
func doUpdateLootPool(ctx context.Context, poolID uint64, req *LootPoolRequest) (*loot.Pool, map[string]interface{}, error) {
	var (
		target  loot.Pool
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockLootPool(tx, &target, poolID); err != nil {
			return err
		}
		old := target
		if err := applyLootPool(&target, req); err != nil {
			return err
		}
		changes = lootPoolChanges(&old, &target)
		if len(changes) == 0 {
			return nil
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, nil, errors.New(loot.PoolCodeTaken)
		}
		return nil, nil, err
	}
	return &target, changes, nil
}

// doDeleteLootPool 删除掉落池及其条目, 返回被删除的掉落池和条目数
func doDeleteLootPool(ctx context.Context, poolID uint64) (*loot.Pool, int64, error) {
	var (
		target  loot.Pool
		entries int64
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockLootPool(tx, &target, poolID); err != nil {
			return err
		}
		result := tx.Where("pool_id = ?", poolID).Delete(&loot.Entry{})
		if result.Error != nil {
			return result.Error
		}
		entries = result.RowsAffected
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, 0, err
	}
	return &target, entries, nil
}

// queryLootPoolEntries 获取掉落池及其全部条目
func queryLootPoolEntries(ctx context.Context, poolID uint64) (*loot.Pool, []loot.EntryDetail, error) {
	var target loot.Pool
	if err := target.Exact(db.DB(ctx), poolID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New(loot.PoolNotFound)
		}
		return nil, nil, err
	}
	entries, err := loot.QueryEntries(db.DB(ctx), poolID)
	if err != nil {
		return nil, nil, err
	}
	return &target, entries, nil
}

// doReplaceLootPoolEntries 用提交的条目整体替换掉落池的条目, 返回掉落池、替换后的条目和用于操作日志的新旧条目
//
// 物品需未删除, 已下架的物品也可以加入, 上架后才参与抽取; 货币需存在, 已停用的货币同理。
func doReplaceLootPoolEntries(ctx context.Context, poolID uint64, req *ReplaceLootPoolEntriesRequest) (*loot.Pool, []loot.EntryDetail, map[string]interface{}, error) {
	entries, err := buildLootEntries(poolID, req.Entries)
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		target  loot.Pool
		old     []loot.Entry
		details []loot.EntryDetail
	)
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockLootPool(tx, &target, poolID); err != nil {
			return err
		}
		if err := checkLootTargets(tx, entries); err != nil {
			return err
		}
		if err := tx.Where("pool_id = ?", poolID).Order("id").Find(&old).Error; err != nil {
			return err
		}
		if err := tx.Where("pool_id = ?", poolID).Delete(&loot.Entry{}).Error; err != nil {
			return err
		}
		if len(entries) > 0 {
			if err := tx.Create(&entries).Error; err != nil {
				return err
			}
		}
		var err error
		details, err = loot.QueryEntries(tx, poolID)
		return err
	})
	if err != nil {
		return nil, nil, nil, err
	}
	changes := map[string]interface{}{
		"old": lootEntryFields(old),
		"new": lootEntryFields(entries),
	}
	return &target, details, changes, nil
}

// buildLootEntries 校验提交的条目并转换为待写入的条目
func buildLootEntries(poolID uint64, reqEntries []LootPoolEntryRequest) ([]loot.Entry, error) {
	if len(reqEntries) > loot.MaxEntries {
		return nil, errors.New(loot.EntryTooMany)
	}
	entries := make([]loot.Entry, len(reqEntries))
	for i, reqEntry := range reqEntries {
		if (reqEntry.ItemID != 0) == (reqEntry.Currency != "") {
			return nil, errors.New(loot.EntryTargetInvalid)
		}
		if reqEntry.MinQuantity > reqEntry.MaxQuantity {
			return nil, errors.New(loot.EntryRangeInvalid)
		}
		if reqEntry.ItemID != 0 && reqEntry.MaxQuantity > inventory.MaxQuantity {
			return nil, errors.New(inventory.QuantityInvalid)
		}
		entries[i] = loot.Entry{
			PoolID:      poolID,
			ItemID:      reqEntry.ItemID,
			Currency:    reqEntry.Currency,
			Weight:      reqEntry.Weight,
			MinQuantity: reqEntry.MinQuantity,
			MaxQuantity: reqEntry.MaxQuantity,
		}
	}
	return entries, nil
}

// checkLootTargets 条目中的物品需未删除, 货币需存在
func checkLootTargets(tx *gorm.DB, entries []loot.Entry) error {
	itemIDs := map[uint64]struct{}{}
	codes := map[string]struct{}{}
	for _, entry := range entries {
		if entry.ItemID != 0 {
			itemIDs[entry.ItemID] = struct{}{}
		} else {
			codes[entry.Currency] = struct{}{}
		}
	}

	if len(itemIDs) > 0 {
		var count int64
		if err := tx.Model(&item.Item{}).Where("id IN ?", slices.Collect(maps.Keys(itemIDs))).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(itemIDs)) {
			return errors.New(item.ItemNotFound)
		}
	}
	if len(codes) > 0 {
		var count int64
		if err := tx.Model(&wallet.Currency{}).Where("code IN ?", slices.Collect(maps.Keys(codes))).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(codes)) {
			return errors.New(wallet.CurrencyNotFound)
		}
	}
	return nil
}

// lockLootPool 锁定掉落池, 防止并发修改、替换条目和删除互相覆盖
func lockLootPool(tx *gorm.DB, target *loot.Pool, poolID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), poolID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(loot.PoolNotFound)
		}
		return err
	}
	return nil
}

// applyLootPool 校验请求并写入掉落池字段
func applyLootPool(target *loot.Pool, req *LootPoolRequest) error {
	if !lootPoolCodePattern.MatchString(req.Code) {
		return errors.New(loot.PoolCodeInvalid)
	}
	name := strings.TrimSpace(req.Name)
	if n := utf8.RuneCountInString(name); n < 1 || n > 64 {
		return errors.New(loot.PoolNameInvalid)
	}

	target.Code = req.Code
	target.Name = name
	target.Description = strings.TrimSpace(req.Description)
	target.IsEnabled = req.IsEnabled
	return nil
}

// lootPoolFields 掉落池可修改的字段, 用于操作日志
func lootPoolFields(target *loot.Pool) map[string]interface{} {
	return map[string]interface{}{
		"code":        target.Code,
		"name":        target.Name,
		"description": target.Description,
		"is_enabled":  target.IsEnabled,
	}
}

// lootPoolChanges 本次修改了哪些字段, 用于操作日志
func lootPoolChanges(old, next *loot.Pool) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := lootPoolFields(next)
	for key, oldValue := range lootPoolFields(old) {
		if newValue := nextFields[key]; oldValue != newValue {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}

// lootEntryFields 掉落池条目的内容, 用于操作日志
func lootEntryFields(entries []loot.Entry) []map[string]interface{} {
	fields := make([]map[string]interface{}, len(entries))
	for i, entry := range entries {
		fields[i] = map[string]interface{}{
			"item_id":      entry.ItemID,
			"currency":     entry.Currency,
			"weight":       entry.Weight,
			"min_quantity": entry.MinQuantity,
			"max_quantity": entry.MaxQuantity,
		}
	}
	return fields
}

// QueryVenues 获取地图列表, 不含已删除的地图
func QueryVenues(ctx context.Context, req *ListVenuesRequest) (int64, []venue.Venue, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&venue.Venue{})
	if req.Keyword != "" {
		keyword := "%" + likeEscaper.Replace(req.Keyword) + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", keyword, keyword)
	}
	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	venues := []venue.Venue{}
	if err := query.Order("sort_order, id").Offset(offset).Limit(req.Size).Find(&venues).Error; err != nil {
		return 0, nil, err
	}
	return total, venues, nil
}

// doCreateVenue 校验并新建地图
func doCreateVenue(ctx context.Context, req *VenueRequest) (*venue.Venue, error) {
	target := &venue.Venue{}
	applyVenue(target, req)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := venue.Validate(tx, target); err != nil {
			return err
		}
		return tx.Create(target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(venue.CodeTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateVenue 校验并修改地图, 返回修改后的地图和本次修改的字段; 没有任何字段变化时只更新模拟结果
func doUpdateVenue(ctx context.Context, venueID uint64, req *VenueRequest) (*venue.Venue, map[string]interface{}, error) {
	var (
		target  venue.Venue
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockVenue(tx, &target, venueID); err != nil {
			return err
		}
		if req.Code != target.Code {
			return errors.New(venue.CodeImmutable)
		}
		old := target
		applyVenue(&target, req)
		if err := venue.Validate(tx, &target); err != nil {
			return err
		}
		changes = venueChanges(&old, &target)
		return tx.Save(&target).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return &target, changes, nil
}

// doDeleteVenue 软删除地图, 返回被删除的地图
func doDeleteVenue(ctx context.Context, venueID uint64) (*venue.Venue, error) {
	var target venue.Venue
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockVenue(tx, &target, venueID); err != nil {
			return err
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// doSimulateVenue 按保存时的规则校验并模拟仓库, 不保存
func doSimulateVenue(ctx context.Context, req *VenueRequest) (*venue.Simulation, error) {
	target := &venue.Venue{}
	applyVenue(target, req)
	if err := venue.Validate(db.DB(ctx), target); err != nil {
		return nil, err
	}
	return &target.Simulation, nil
}

// lockVenue 锁定地图, 防止并发修改和删除互相覆盖
func lockVenue(tx *gorm.DB, target *venue.Venue, venueID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), venueID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(venue.VenueNotFound)
		}
		return err
	}
	return nil
}

// applyVenue 写入地图字段, 由 venue.Validate 校验
func applyVenue(target *venue.Venue, req *VenueRequest) {
	target.Code = req.Code
	target.Name = strings.TrimSpace(req.Name)
	target.Description = strings.TrimSpace(req.Description)
	target.ImageURL = req.ImageURL
	target.SortOrder = req.SortOrder
	target.IsEnabled = req.IsEnabled
	target.PveEnabled = req.PveEnabled
	target.RankedEnabled = req.RankedEnabled
	target.BidCurrency = req.BidCurrency
	target.EntryCost = req.EntryCost
	target.Requirements = req.Requirements
	target.Warehouse = req.Warehouse
	target.Auction = req.Auction
	target.Bots = req.Bots
	target.Pve = req.Pve
}

// venueFields 地图可修改的字段, 用于操作日志
func venueFields(target *venue.Venue) map[string]interface{} {
	return map[string]interface{}{
		"code":           target.Code,
		"name":           target.Name,
		"description":    target.Description,
		"image_url":      target.ImageURL,
		"sort_order":     target.SortOrder,
		"is_enabled":     target.IsEnabled,
		"pve_enabled":    target.PveEnabled,
		"ranked_enabled": target.RankedEnabled,
		"bid_currency":   target.BidCurrency,
		"entry_cost":     target.EntryCost,
		"requirements":   target.Requirements,
		"warehouse":      target.Warehouse,
		"auction":        target.Auction,
		"bots":           target.Bots,
		"pve":            target.Pve,
	}
}

// venueChanges 本次修改了哪些字段, 用于操作日志; 规则中有切片, 按内容比较
func venueChanges(old, next *venue.Venue) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := venueFields(next)
	for key, oldValue := range venueFields(old) {
		if newValue := nextFields[key]; !reflect.DeepEqual(oldValue, newValue) {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}

// QueryMatches 获取对局列表
func QueryMatches(ctx context.Context, req *ListMatchesRequest) (int64, []match.MatchDetail, error) {
	offset := (req.Current - 1) * req.Size

	query := match.FilterMode(db.DB(ctx).Model(&match.Match{}), req.Prefix, req.Suffix)
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.UserID != nil {
		query = query.Where("id IN (SELECT match_id FROM match_players WHERE user_id = ?)", *req.UserID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var matches []match.Match
	if err := query.Order("id DESC").Offset(offset).Limit(req.Size).Find(&matches).Error; err != nil {
		return 0, nil, err
	}
	details, err := match.LoadDetails(db.DB(ctx), matches)
	if err != nil {
		return 0, nil, err
	}
	return total, details, nil
}

// QueryLeaderboards 获取榜单列表, 按排序值排列
func QueryLeaderboards(ctx context.Context, req *ListLeaderboardsRequest) (int64, []leaderboard.Board, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&leaderboard.Board{})
	if req.Keyword != "" {
		keyword := "%" + likeEscaper.Replace(req.Keyword) + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", keyword, keyword)
	}
	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	boards := []leaderboard.Board{}
	if err := query.Order("sort_order, id").Offset(offset).Limit(req.Size).Find(&boards).Error; err != nil {
		return 0, nil, err
	}
	return total, boards, nil
}

// doCreateLeaderboard 新建榜单
func doCreateLeaderboard(ctx context.Context, req *LeaderboardRequest) (*leaderboard.Board, error) {
	target := &leaderboard.Board{}
	tx := db.DB(ctx)
	if err := applyLeaderboard(tx, target, req); err != nil {
		return nil, err
	}
	if err := tx.Create(target).Error; err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(leaderboard.CodeTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateLeaderboard 修改榜单, 返回修改后的榜单和本次修改的字段, 没有任何字段变化时不写库
//
// 修改后缓存的前 N 名随即失效。
func doUpdateLeaderboard(ctx context.Context, boardID uint64, req *LeaderboardRequest) (*leaderboard.Board, map[string]interface{}, error) {
	var (
		target  leaderboard.Board
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockLeaderboard(tx, &target, boardID); err != nil {
			return err
		}
		old := target
		if err := applyLeaderboard(tx, &target, req); err != nil {
			return err
		}
		changes = leaderboardChanges(&old, &target)
		if len(changes) == 0 {
			return nil
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, nil, errors.New(leaderboard.CodeTaken)
		}
		return nil, nil, err
	}
	return &target, changes, nil
}

// doDeleteLeaderboard 删除榜单, 统计数据不受影响
func doDeleteLeaderboard(ctx context.Context, boardID uint64) (*leaderboard.Board, error) {
	var target leaderboard.Board
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockLeaderboard(tx, &target, boardID); err != nil {
			return err
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// lockLeaderboard 锁定榜单, 防止并发修改和删除互相覆盖
func lockLeaderboard(tx *gorm.DB, target *leaderboard.Board, boardID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), boardID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(leaderboard.BoardNotFound)
		}
		return err
	}
	return nil
}

// applyLeaderboard 写入榜单字段并校验
func applyLeaderboard(tx *gorm.DB, target *leaderboard.Board, req *LeaderboardRequest) error {
	target.Code = req.Code
	target.Name = strings.TrimSpace(req.Name)
	target.Description = strings.TrimSpace(req.Description)
	target.Metric = req.Metric
	target.Scope = req.Scope
	target.Period = req.Period
	target.Size = req.Size
	target.MinMatches = req.MinMatches
	target.SortOrder = req.SortOrder
	target.IsEnabled = req.IsEnabled
	return leaderboard.Validate(tx, target)
}

// leaderboardFields 榜单可修改的字段, 用于操作日志
func leaderboardFields(target *leaderboard.Board) map[string]interface{} {
	return map[string]interface{}{
		"code":        target.Code,
		"name":        target.Name,
		"description": target.Description,
		"metric":      target.Metric,
		"scope":       target.Scope,
		"period":      target.Period,
		"size":        target.Size,
		"min_matches": target.MinMatches,
		"sort_order":  target.SortOrder,
		"is_enabled":  target.IsEnabled,
	}
}

// leaderboardChanges 本次修改了哪些字段, 用于操作日志
func leaderboardChanges(old, next *leaderboard.Board) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := leaderboardFields(next)
	for key, oldValue := range leaderboardFields(old) {
		if newValue := nextFields[key]; oldValue != newValue {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}

// QuerySeasons 获取赛季列表, 按开始时间从晚到早排列
func QuerySeasons(ctx context.Context, req *ListSeasonsRequest) (int64, []season.View, error) {
	offset := (req.Current - 1) * req.Size
	now := time.Now()

	query := db.DB(ctx).Model(&season.Season{})
	if req.Keyword != "" {
		keyword := "%" + likeEscaper.Replace(req.Keyword) + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", keyword, keyword)
	}
	// 与 season.Season.StatusAt 一致
	switch req.Status {
	case season.StatusUpcoming:
		query = query.Where("settled_at IS NULL AND starts_at > ?", now)
	case season.StatusActive:
		query = query.Where("settled_at IS NULL AND starts_at <= ? AND ends_at > ?", now, now)
	case season.StatusEnded:
		query = query.Where("settled_at IS NULL AND ends_at <= ?", now)
	case season.StatusSettled:
		query = query.Where("settled_at IS NOT NULL")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var seasons []season.Season
	if err := query.Order("starts_at DESC, id DESC").Offset(offset).Limit(req.Size).Find(&seasons).Error; err != nil {
		return 0, nil, err
	}
	views := make([]season.View, len(seasons))
	for i := range seasons {
		views[i] = season.NewView(&seasons[i], now)
	}
	return total, views, nil
}

// doCreateSeason 新建赛季, 时间不能与其他赛季重叠
func doCreateSeason(ctx context.Context, req *CreateSeasonRequest) (*season.View, error) {
	now := time.Now()
	target := &season.Season{
		Code:        req.Code,
		Name:        strings.TrimSpace(req.Name),
		StartsAt:    req.StartsAt.Truncate(time.Minute),
		EndsAt:      req.EndsAt.Truncate(time.Minute),
		RatingCarry: req.RatingCarry,
	}
	if err := season.Validate(target); err != nil {
		return nil, err
	}
	if !target.EndsAt.After(now) {
		return nil, errors.New(season.TimeInvalid)
	}
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := season.Lock(tx); err != nil {
			return err
		}
		if err := season.CheckOverlap(tx, target); err != nil {
			return err
		}
		return tx.Create(target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(season.CodeTaken)
		}
		return nil, err
	}
	view := season.NewView(target, now)
	return &view, nil
}

// doUpdateSeason 修改赛季, 返回修改后的赛季和本次修改的字段, 没有任何字段变化时不写库
//
// 时间按分钟比较, 与原来在同一分钟内视为没有修改; 赛季开始后不能修改开始时间和段位分继承比例, 结束后只能修改名称。
func doUpdateSeason(ctx context.Context, seasonID uint64, req *SeasonRequest) (*season.View, map[string]interface{}, error) {
	var (
		target  season.Season
		changes map[string]interface{}
		now     = time.Now()
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := season.Lock(tx); err != nil {
			return err
		}
		if err := lockSeason(tx, &target, seasonID); err != nil {
			return err
		}
		old := target
		startsAt, endsAt := req.StartsAt.Truncate(time.Minute), req.EndsAt.Truncate(time.Minute)
		startChanged := !startsAt.Equal(target.StartsAt.Truncate(time.Minute))
		endChanged := !endsAt.Equal(target.EndsAt.Truncate(time.Minute))
		carryChanged := req.RatingCarry != target.RatingCarry
		switch target.StatusAt(now) {
		case season.StatusUpcoming:
		case season.StatusActive:
			if startChanged || carryChanged {
				return errors.New(season.StartLocked)
			}
		default:
			if startChanged || endChanged || carryChanged {
				return errors.New(season.SeasonEnded)
			}
		}

		target.Name = strings.TrimSpace(req.Name)
		target.RatingCarry = req.RatingCarry
		if startChanged {
			target.StartsAt = startsAt
		}
		if endChanged {
			if !endsAt.After(now) {
				return errors.New(season.TimeInvalid)
			}
			target.EndsAt = endsAt
		}
		if err := season.Validate(&target); err != nil {
			return err
		}
		changes = seasonChanges(&old, &target)
		if len(changes) == 0 {
			return nil
		}
		if startChanged || endChanged {
			if err := season.CheckOverlap(tx, &target); err != nil {
				return err
			}
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		return nil, nil, err
	}
	view := season.NewView(&target, now)
	return &view, changes, nil
}

// doDeleteSeason 删除未开始的赛季
func doDeleteSeason(ctx context.Context, seasonID uint64) (*season.Season, error) {
	var target season.Season
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSeason(tx, &target, seasonID); err != nil {
			return err
		}
		if target.StatusAt(time.Now()) != season.StatusUpcoming {
			return errors.New(season.DeleteStarted)
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// doEndSeason 提前结束进行中的赛季, 结束时间改为当前时间, 返回结束后的赛季和修改的字段
func doEndSeason(ctx context.Context, seasonID uint64) (*season.View, map[string]interface{}, error) {
	var (
		target  season.Season
		changes map[string]interface{}
		now     = time.Now()
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSeason(tx, &target, seasonID); err != nil {
			return err
		}
		if target.StatusAt(now) != season.StatusActive {
			return errors.New(season.EndNotActive)
		}
		old := target
		target.EndsAt = now
		changes = seasonChanges(&old, &target)
		return tx.Save(&target).Error
	})
	if err != nil {
		return nil, nil, err
	}
	view := season.NewView(&target, now)
	return &view, changes, nil
}

// lockSeason 锁定赛季, 防止并发修改、删除和结束互相覆盖
func lockSeason(tx *gorm.DB, target *season.Season, seasonID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), seasonID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(season.SeasonNotFound)
		}
		return err
	}
	return nil
}

// seasonFields 赛季可修改的字段, 用于操作日志; 时间统一为 UTC 的 RFC3339 格式, 便于比较
func seasonFields(target *season.Season) map[string]interface{} {
	return map[string]interface{}{
		"code":         target.Code,
		"name":         target.Name,
		"starts_at":    target.StartsAt.UTC().Format(time.RFC3339),
		"ends_at":      target.EndsAt.UTC().Format(time.RFC3339),
		"rating_carry": target.RatingCarry,
	}
}

// seasonChanges 本次修改了哪些字段, 用于操作日志
func seasonChanges(old, next *season.Season) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := seasonFields(next)
	for key, oldValue := range seasonFields(old) {
		if newValue := nextFields[key]; oldValue != newValue {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}

// queryRankedTiers 获取全部段位, 按分数线从低到高
func queryRankedTiers(ctx context.Context) ([]ranked.Tier, error) {
	return ranked.QueryTiers(db.DB(ctx))
}

// doCreateRankedTier 新建段位
func doCreateRankedTier(ctx context.Context, req *RankedTierRequest) (*ranked.Tier, error) {
	target := &ranked.Tier{}
	if err := applyRankedTier(target, req); err != nil {
		return nil, err
	}
	if err := db.DB(ctx).Create(target).Error; err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(ranked.TierRatingTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdateRankedTier 修改段位, 返回修改后的段位和本次修改的字段, 没有任何字段变化时不写库
func doUpdateRankedTier(ctx context.Context, tierID uint64, req *RankedTierRequest) (*ranked.Tier, map[string]interface{}, error) {
	var (
		target  ranked.Tier
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockRankedTier(tx, &target, tierID); err != nil {
			return err
		}
		old := target
		if err := applyRankedTier(&target, req); err != nil {
			return err
		}
		changes = rankedTierChanges(&old, &target)
		if len(changes) == 0 {
			return nil
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, nil, errors.New(ranked.TierRatingTaken)
		}
		return nil, nil, err
	}
	return &target, changes, nil
}

// doDeleteRankedTier 删除段位, 段位分不受影响
func doDeleteRankedTier(ctx context.Context, tierID uint64) (*ranked.Tier, error) {
	var target ranked.Tier
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockRankedTier(tx, &target, tierID); err != nil {
			return err
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// lockRankedTier 锁定段位, 防止并发修改和删除互相覆盖
func lockRankedTier(tx *gorm.DB, target *ranked.Tier, tierID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), tierID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(ranked.TierNotFound)
		}
		return err
	}
	return nil
}

// applyRankedTier 写入段位字段并校验, 颜色统一为小写
func applyRankedTier(target *ranked.Tier, req *RankedTierRequest) error {
	target.Name = strings.TrimSpace(req.Name)
	target.MinRating = req.MinRating
	target.Color = strings.ToLower(req.Color)
	return ranked.ValidateTier(target)
}

// rankedTierFields 段位可修改的字段, 用于操作日志
func rankedTierFields(target *ranked.Tier) map[string]interface{} {
	return map[string]interface{}{
		"name":       target.Name,
		"min_rating": target.MinRating,
		"color":      target.Color,
	}
}

// rankedTierChanges 本次修改了哪些字段, 用于操作日志
func rankedTierChanges(old, next *ranked.Tier) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := rankedTierFields(next)
	for key, oldValue := range rankedTierFields(old) {
		if newValue := nextFields[key]; oldValue != newValue {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}

// queryUserInventory 获取指定用户背包中的物品
func queryUserInventory(ctx context.Context, userID uint64, req *inventory.ListItemsRequest) (int64, []inventory.UserItemDetail, error) {
	var user auth.User
	if err := user.Exact(db.DB(ctx), userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil, errors.New(auth.UserNotFound)
		}
		return 0, nil, err
	}
	return inventory.QueryUserItems(ctx, userID, req)
}

// doAdjustUserInventory 管理员给其他用户发放或扣除物品, 不允许给自己调整
//
// 幂等键由前端在每次打开表单时生成, 防止重复点击导致重复发放。
func doAdjustUserInventory(ctx context.Context, operatorID, userID uint64, req *AdjustUserInventoryRequest) (*inventory.Transaction, bool, error) {
	if operatorID == userID {
		return nil, false, errors.New(CannotOperateSelf)
	}
	remark, err := normalizeRemark(req.Remark)
	if err != nil {
		return nil, false, err
	}

	var (
		txn      *inventory.Transaction
		replayed bool
	)
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var user auth.User
		if err := user.Exact(tx, userID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(auth.UserNotFound)
			}
			return err
		}

		var err error
		txn, replayed, err = inventory.Change(tx, &inventory.ChangeRequest{
			UserID:         userID,
			ItemID:         req.ItemID,
			Quantity:       req.Quantity,
			BizType:        inventory.BizAdminAdjust,
			IdempotencyKey: fmt.Sprintf("%s:%d:%s", inventory.BizAdminAdjust, operatorID, req.IdempotencyKey),
			OperatorID:     operatorID,
			Remark:         remark,
		})
		return err
	})
	if err != nil {
		return nil, false, err
	}
	return txn, replayed, nil
}

// InventoryTransactionItem 管理后台展示的背包流水, 带上用户和操作人的用户名
type InventoryTransactionItem struct {
	ID               uint64    `json:"id"`
	UserID           uint64    `json:"user_id"`
	Username         string    `json:"username"` // 用户已不存在时为空
	ItemID           uint64    `json:"item_id"`
	ItemName         string    `json:"item_name"`
	Quantity         int64     `json:"quantity"`
	QuantityAfter    int64     `json:"quantity_after"`
	BizType          string    `json:"biz_type"`
	BizID            string    `json:"biz_id"`
	IdempotencyKey   string    `json:"idempotency_key"`
	OperatorID       uint64    `json:"operator_id"`
	OperatorUsername string    `json:"operator_username"` // 系统操作或用户已不存在时为空
	Remark           string    `json:"remark"`
	CreatedAt        time.Time `json:"created_at"`
}

// QueryInventoryTransactions 获取全部用户的背包流水
func QueryInventoryTransactions(ctx context.Context, req *ListInventoryTransactionsRequest) (int64, []InventoryTransactionItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&inventory.Transaction{})
	if req.UserID != nil {
		query = query.Where("inventory_transactions.user_id = ?", *req.UserID)
	}
	if req.OperatorID != nil {
		query = query.Where("inventory_transactions.operator_id = ?", *req.OperatorID)
	}
	if req.ItemID != nil {
		query = query.Where("inventory_transactions.item_id = ?", *req.ItemID)
	}
	if req.BizType != "" {
		query = query.Where("inventory_transactions.biz_type = ?", req.BizType)
	}
	if req.BizID != "" {
		query = query.Where("inventory_transactions.biz_id = ?", req.BizID)
	}
	if req.StartTime != nil {
		query = query.Where("inventory_transactions.created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("inventory_transactions.created_at < ?", *req.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	items := []InventoryTransactionItem{}
	if err := query.
		Select("inventory_transactions.*, COALESCE(users.username, '') AS username, COALESCE(operators.username, '') AS operator_username").
		Joins("LEFT JOIN users ON users.id = inventory_transactions.user_id").
		Joins("LEFT JOIN users AS operators ON operators.id = inventory_transactions.operator_id").
		Order("inventory_transactions.id DESC").Offset(offset).Limit(req.Size).
		Scan(&items).Error; err != nil {
		return 0, nil, err
	}

	return total, items, nil
}

// SettingRevisionItem 设置组的一个版本, 不含快照内容
type SettingRevisionItem struct {
	Version           int64                  `json:"version"`
	Action            string                 `json:"action"`
	FromVersion       int64                  `json:"from_version"`
	Changes           map[string]interface{} `gorm:"serializer:json" json:"changes"`
	UpdatedBy         uint64                 `json:"updated_by"`
	UpdatedByUsername string                 `json:"updated_by_username"` // 系统操作或用户已不存在时为空
	CreatedAt         time.Time              `json:"created_at"`
}

// QuerySettingRevisions 获取设置组的历史版本, 按版本从新到旧
func QuerySettingRevisions(ctx context.Context, key string, req *ListSettingRevisionsRequest) (int64, []SettingRevisionItem, error) {
	offset := (req.Current - 1) * req.Size
	query := db.DB(ctx).Model(&settings.Revision{}).Where("setting_revisions.key = ?", key)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	revisions := []SettingRevisionItem{}
	if err := query.
		Select("setting_revisions.version, setting_revisions.action, setting_revisions.from_version, setting_revisions.changes, " +
			"setting_revisions.updated_by, setting_revisions.created_at, COALESCE(users.username, '') AS updated_by_username").
		Joins("LEFT JOIN users ON users.id = setting_revisions.updated_by").
		Order("setting_revisions.version DESC").Offset(offset).Limit(req.Size).
		Scan(&revisions).Error; err != nil {
		return 0, nil, err
	}

	return total, revisions, nil
}

// prepareTestMail 校验测试邮件的收件人, 并取出已保存的邮件设置
//
// 只使用已保存的设置, 不接受草稿: 否则改掉 SMTP 服务器后沿用已保存的密码, 就能把密码发到任意服务器。
func prepareTestMail(ctx context.Context, req *TestMailRequest) (string, *settings.MailSettings, error) {
	to, err := mail.NormalizeAddress(req.To)
	if err != nil {
		return "", nil, errors.New(auth.EmailInvalid)
	}
	mailConfig, err := settings.Mail.Get(ctx)
	if err != nil {
		return "", nil, err
	}
	if mailConfig.Host == "" || mailConfig.FromAddress == "" {
		return "", nil, errors.New(MailTestIncomplete)
	}
	return to, mailConfig, nil
}
