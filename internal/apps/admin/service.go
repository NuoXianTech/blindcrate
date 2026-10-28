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
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
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

// applyItem 校验请求并写入物品字段, 卖出货币必须存在, 已停用的货币也可以选
func applyItem(tx *gorm.DB, target *item.Item, req *ItemRequest) error {
	name := strings.TrimSpace(req.Name)
	if n := utf8.RuneCountInString(name); n < 1 || n > 64 {
		return errors.New(item.ItemNameInvalid)
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
	target.ImageURL = req.ImageURL
	target.SellCurrency = req.SellCurrency
	target.SellPrice = req.SellPrice
	target.Weight = req.Weight
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
		"image_url":     target.ImageURL,
		"sell_currency": target.SellCurrency,
		"sell_price":    target.SellPrice,
		"weight":        target.Weight,
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

// prepareTestMail 校验测试邮件的收件人, 并取出已保存的邮件设置
//
// 只使用已保存的设置, 不接受草稿: 否则改掉 SMTP 服务器后沿用已保存的密码, 就能把密码发到任意服务器。
func prepareTestMail(ctx context.Context, req *TestMailRequest) (string, *settings.MailSettings, error) {
	to, err := mail.NormalizeAddress(req.To)
	if err != nil {
		return "", nil, errors.New(auth.EmailInvalid)
	}
	mailConfig, err := settings.Mail(ctx)
	if err != nil {
		return "", nil, err
	}
	if mailConfig.Host == "" || mailConfig.FromAddress == "" {
		return "", nil, errors.New(MailTestIncomplete)
	}
	return to, mailConfig, nil
}
