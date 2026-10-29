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

package itemadmin

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/effect"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
		keyword := "%" + db.EscapeLike(req.Keyword) + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ?", keyword, keyword)
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
		if err := ApplyItem(tx, target, req); err != nil {
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
		if err := LockItem(tx, &target, itemID); err != nil {
			return err
		}
		old := target
		if err := ApplyItem(tx, &target, req); err != nil {
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
		if err := LockItem(tx, &target, itemID); err != nil {
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

// LockItem 锁定未删除的物品, 防止并发修改和删除互相覆盖
func LockItem(tx *gorm.DB, target *item.Item, itemID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), itemID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(item.ItemNotFound)
		}
		return err
	}
	return nil
}

// itemCodePattern 物品代码, 小写字母、数字和下划线
var itemCodePattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

// ApplyItem 校验请求并写入物品字段, 品质、类别和卖出货币必须存在, 已停用的货币也可以选
//
// 名称有唯一索引, 撞上时按名称已存在处理; 代码可以重复。
func ApplyItem(tx *gorm.DB, target *item.Item, req *ItemRequest) error {
	if !itemCodePattern.MatchString(req.Code) {
		return errors.New(item.ItemCodeInvalid)
	}
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
	params, err := effect.Validate(tx, req.Category, req.Effect, req.EffectParams)
	if err != nil {
		return err
	}

	target.Code = req.Code
	target.Name = name
	target.Description = strings.TrimSpace(req.Description)
	target.Quality = req.Quality
	target.Category = req.Category
	target.ImageURL = req.ImageURL
	target.SellCurrency = req.SellCurrency
	target.SellPrice = req.SellPrice
	target.Width = req.Width
	target.Height = req.Height
	target.Effect = req.Effect
	target.EffectParams = params
	target.IsEnabled = req.IsEnabled
	return nil
}

// itemFields 物品可修改的字段, 用于操作日志
func itemFields(target *item.Item) map[string]interface{} {
	return map[string]interface{}{
		"code":          target.Code,
		"name":          target.Name,
		"description":   target.Description,
		"quality":       target.Quality,
		"category":      target.Category,
		"image_url":     target.ImageURL,
		"sell_currency": target.SellCurrency,
		"sell_price":    target.SellPrice,
		"width":         target.Width,
		"height":        target.Height,
		"effect":        target.Effect,
		"effect_params": target.EffectParams,
		"is_enabled":    target.IsEnabled,
	}
}

// itemChanges 本次修改了哪些字段, 用于操作日志; 效果参数是 map, 按内容比较
func itemChanges(old, next *item.Item) map[string]interface{} {
	changes := map[string]interface{}{}
	nextFields := itemFields(next)
	for key, oldValue := range itemFields(old) {
		if newValue := nextFields[key]; !reflect.DeepEqual(oldValue, newValue) {
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
	if err := ApplyQuality(target, &req.QualityRequest); err != nil {
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
		if err := LockQuality(tx, &target, level); err != nil {
			return err
		}
		old := target
		if err := ApplyQuality(&target, req); err != nil {
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
		if err := LockQuality(tx, &target, level); err != nil {
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

// LockQuality 锁定品质, 防止并发修改和删除互相覆盖
func LockQuality(tx *gorm.DB, target *item.Quality, level int16) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), level); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(item.QualityNotFound)
		}
		return err
	}
	return nil
}

// ApplyQuality 写入品质字段并校验, 颜色统一为小写
func ApplyQuality(target *item.Quality, req *QualityRequest) error {
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
