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

package lootadmin

import (
	"context"
	"errors"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
		keyword := "%" + db.EscapeLike(req.Keyword) + "%"
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
	if err := ApplyLootPool(target, req); err != nil {
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
		if err := LockLootPool(tx, &target, poolID); err != nil {
			return err
		}
		old := target
		if err := ApplyLootPool(&target, req); err != nil {
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
		if err := LockLootPool(tx, &target, poolID); err != nil {
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
	entries, err := BuildLootEntries(poolID, req.Entries)
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		target  loot.Pool
		old     []loot.Entry
		details []loot.EntryDetail
	)
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockLootPool(tx, &target, poolID); err != nil {
			return err
		}
		if err := CheckLootTargets(tx, entries); err != nil {
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

// BuildLootEntries 校验提交的条目并转换为待写入的条目
func BuildLootEntries(poolID uint64, reqEntries []LootPoolEntryRequest) ([]loot.Entry, error) {
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

// CheckLootTargets 条目中的物品需未删除, 货币需存在
func CheckLootTargets(tx *gorm.DB, entries []loot.Entry) error {
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

// LockLootPool 锁定掉落池, 防止并发修改、替换条目和删除互相覆盖
func LockLootPool(tx *gorm.DB, target *loot.Pool, poolID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), poolID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(loot.PoolNotFound)
		}
		return err
	}
	return nil
}

// ApplyLootPool 校验请求并写入掉落池字段
func ApplyLootPool(target *loot.Pool, req *LootPoolRequest) error {
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
