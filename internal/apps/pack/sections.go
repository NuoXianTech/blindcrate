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

package pack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/item/itemadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard"
	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard/leaderboardadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot/lootadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/shop"
	"github.com/NuoXianTech/blindcrate/internal/apps/shop/shopadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/upload"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue/venueadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet/walletadmin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 内置的配置种类, 其他模块的配置种类在各自的 init 中注册
func init() {
	RegisterSection(Section{
		Code: PackCurrencies, Label: "货币", Hint: "按代码对应", Order: OrderCurrencies,
		Export: func(_ context.Context, tx *gorm.DB, _ func(string) string) (any, int, error) {
			var rows []wallet.Currency
			if err := tx.Order("sort_order, code").Find(&rows).Error; err != nil {
				return nil, 0, err
			}
			list := make([]packCurrency, len(rows))
			for i := range rows {
				list[i] = currencyPack(&rows[i])
			}
			return list, len(list), nil
		},
		Decode: DecodeList(currencyEntry, nil),
	})
	RegisterSection(Section{
		Code: PackQualities, Label: "品质", Hint: "按等级对应", Order: OrderQualities,
		Export: func(_ context.Context, tx *gorm.DB, _ func(string) string) (any, int, error) {
			var rows []item.Quality
			if err := tx.Order("level").Find(&rows).Error; err != nil {
				return nil, 0, err
			}
			list := make([]itemadmin.CreateQualityRequest, len(rows))
			for i := range rows {
				list[i] = qualityPack(&rows[i])
			}
			return list, len(list), nil
		},
		Decode: DecodeList(qualityEntry, nil),
	})
	RegisterSection(Section{
		Code: PackItems, Label: "物品", Hint: "按 ID 对应, 带上物品图片", Order: OrderItems,
		Export: func(_ context.Context, tx *gorm.DB, image func(string) string) (any, int, error) {
			var rows []item.Item
			if err := tx.Order("id").Find(&rows).Error; err != nil {
				return nil, 0, err
			}
			list := make([]packItem, len(rows))
			for i := range rows {
				list[i] = itemPack(&rows[i], image)
			}
			return list, len(list), nil
		},
		Decode: DecodeList(itemEntry, nil),
	})
	RegisterSection(Section{
		Code: PackLootPools, Label: "掉落池", Hint: "按代码对应, 含条目; 已删除物品的条目不导出", Order: OrderLootPools, Renamable: true,
		Export: func(_ context.Context, tx *gorm.DB, _ func(string) string) (any, int, error) {
			var (
				rows    []loot.Pool
				entries []loot.Entry
			)
			if err := tx.Order("id").Find(&rows).Error; err != nil {
				return nil, 0, err
			}
			if err := liveLootEntries(tx).Order("pool_id, id").Find(&entries).Error; err != nil {
				return nil, 0, err
			}
			byPool := map[uint64][]loot.Entry{}
			for _, entry := range entries {
				byPool[entry.PoolID] = append(byPool[entry.PoolID], entry)
			}
			list := make([]packLootPool, len(rows))
			for i := range rows {
				list[i] = lootPoolPack(&rows[i], byPool[rows[i].ID])
			}
			return list, len(list), nil
		},
		Decode: DecodeList(lootPoolEntry, nil),
	})
	RegisterSection(Section{
		Code: PackVenues, Label: "地图", Hint: "按代码对应, 带上地图图片", Order: OrderVenues, Renamable: true,
		Export: func(_ context.Context, tx *gorm.DB, image func(string) string) (any, int, error) {
			var rows []venue.Venue
			if err := tx.Order("sort_order, id").Find(&rows).Error; err != nil {
				return nil, 0, err
			}
			list := make([]venueadmin.VenueRequest, len(rows))
			for i := range rows {
				list[i] = venuePack(&rows[i], image)
			}
			return list, len(list), nil
		},
		Decode: DecodeList(venueEntry, venueRenamed),
		Invalid: func(err error) bool {
			var invalid *venue.InvalidError
			return errors.As(err, &invalid)
		},
	})
	RegisterSection(Section{
		Code: PackShopItems, Label: "商品", Hint: "按代码对应, 带上商品图片; 导入时已有商品的剩余库存不变", Order: OrderShopItems, Renamable: true,
		Export: func(_ context.Context, tx *gorm.DB, image func(string) string) (any, int, error) {
			var rows []shop.Item
			if err := tx.Order("sort_order, id").Find(&rows).Error; err != nil {
				return nil, 0, err
			}
			list := make([]shopadmin.ShopItemRequest, len(rows))
			for i := range rows {
				list[i] = shopItemPack(&rows[i], image)
			}
			return list, len(list), nil
		},
		Decode: DecodeList(shopItemEntry, func(req *shopadmin.ShopItemRequest, section, old, next string) {
			if section == PackVenues {
				RenameConditionScopes(req.Requirements, old, next)
			}
		}),
		Invalid: func(err error) bool {
			var invalid *shop.InvalidError
			return errors.As(err, &invalid)
		},
	})
	RegisterSection(Section{
		Code: PackLeaderboards, Label: "榜单", Hint: "按代码对应", Order: OrderLeaderboards, Renamable: true,
		Export: func(_ context.Context, tx *gorm.DB, _ func(string) string) (any, int, error) {
			var rows []leaderboard.Board
			if err := tx.Order("sort_order, id").Find(&rows).Error; err != nil {
				return nil, 0, err
			}
			list := make([]leaderboardadmin.LeaderboardRequest, len(rows))
			for i := range rows {
				list[i] = leaderboardPack(&rows[i])
			}
			return list, len(list), nil
		},
		Decode: DecodeList(leaderboardEntry, func(req *leaderboardadmin.LeaderboardRequest, section, old, next string) {
			if section == PackVenues {
				req.Scope = RenameScope(req.Scope, old, next)
			}
		}),
	})
	RegisterSection(Section{
		Code: PackSettings, Label: "设置", Hint: "游戏分类的设置组, 不含密钥字段", Order: OrderSettings,
		Export: func(ctx context.Context, _ *gorm.DB, _ func(string) string) (any, int, error) {
			values, err := settings.ExportValues(ctx)
			if err != nil {
				return nil, 0, err
			}
			return values, len(values), nil
		},
		Decode: func(data []byte) (Content, error) {
			var c settingsContent
			if err := decodePackJSON(data, &c); err != nil {
				return nil, err
			}
			return c, nil
		},
	})
}

// venueRenamed 地图引用的掉落池改名, 以及条件中统计范围的地图代码改名
func venueRenamed(req *venueadmin.VenueRequest, section, old, next string) {
	switch section {
	case PackLootPools:
		for i := range req.Warehouse.Slots {
			if slot := &req.Warehouse.Slots[i]; slot.Pool == old {
				slot.Pool = next
			}
		}
	case PackVenues:
		RenameConditionScopes(req.Requirements, old, next)
	}
}

// settingsContent 设置组的值, 按设置组的 key
type settingsContent map[string]json.RawMessage

func (c settingsContent) Len() int { return len(c) }

// Entries 按 key 排序
func (c settingsContent) Entries(r *Run) []*Entry {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	entries := make([]*Entry, len(keys))
	for i, key := range keys {
		entries[i] = settingsEntry(r, key, c[key])
	}
	return entries
}

func (settingsContent) Renamed(string, string, string) {}

type (
	packCurrency struct {
		Code string `json:"code" binding:"required,max=32"`
		walletadmin.CurrencyRequest
	}
	packItem struct {
		ID uint64 `json:"id" binding:"required"` // 物品以 ID 对应, 导入时沿用
		itemadmin.ItemRequest
	}
	packLootPool struct {
		lootadmin.LootPoolRequest
		Entries []lootadmin.LootPoolEntryRequest `json:"entries" binding:"dive"`
	}
)

// utcMinute 时间去掉秒并转为 UTC, 包中的时间与已有配置按这个格式比较
func utcMinute(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	utc := t.UTC().Truncate(time.Minute)
	return &utc
}

func currencyPack(c *wallet.Currency) packCurrency {
	return packCurrency{Code: c.Code, CurrencyRequest: walletadmin.CurrencyRequest{
		Name:      c.Name,
		Decimals:  c.Decimals,
		SortOrder: c.SortOrder,
		IsEnabled: c.IsEnabled,
	}}
}

func qualityPack(q *item.Quality) itemadmin.CreateQualityRequest {
	return itemadmin.CreateQualityRequest{Level: q.Level, Code: q.Code, QualityRequest: itemadmin.QualityRequest{
		Name:      q.Name,
		Color:     q.Color,
		Effect:    q.Effect,
		ScanMs:    q.ScanMs,
		SortOrder: q.SortOrder,
	}}
}

func itemPack(it *item.Item, image func(string) string) packItem {
	return packItem{ID: it.ID, ItemRequest: itemadmin.ItemRequest{
		Code:         it.Code,
		Name:         it.Name,
		Description:  it.Description,
		Quality:      it.Quality,
		Category:     it.Category,
		ImageURL:     image(it.ImageURL),
		SellCurrency: it.SellCurrency,
		SellPrice:    it.SellPrice,
		Width:        it.Width,
		Height:       it.Height,
		Effect:       it.Effect,
		EffectParams: it.EffectParams,
		IsEnabled:    it.IsEnabled,
	}}
}

func lootPoolPack(pool *loot.Pool, entries []loot.Entry) packLootPool {
	result := packLootPool{
		LootPoolRequest: lootadmin.LootPoolRequest{Code: pool.Code, Name: pool.Name, Description: pool.Description, IsEnabled: pool.IsEnabled},
		Entries:         make([]lootadmin.LootPoolEntryRequest, len(entries)),
	}
	for i, entry := range entries {
		result.Entries[i] = lootadmin.LootPoolEntryRequest{
			ItemID:      entry.ItemID,
			Currency:    entry.Currency,
			Weight:      entry.Weight,
			MinQuantity: entry.MinQuantity,
			MaxQuantity: entry.MaxQuantity,
		}
	}
	return result
}

func venuePack(v *venue.Venue, image func(string) string) venueadmin.VenueRequest {
	return venueadmin.VenueRequest{
		Code:          v.Code,
		Name:          v.Name,
		Description:   v.Description,
		ImageURL:      image(v.ImageURL),
		SortOrder:     v.SortOrder,
		IsEnabled:     v.IsEnabled,
		PveEnabled:    v.PveEnabled,
		RankedEnabled: v.RankedEnabled,
		BidCurrency:   v.BidCurrency,
		EntryCost:     v.EntryCost,
		Requirements:  v.Requirements,
		Warehouse:     v.Warehouse,
		Auction:       v.Auction,
		Bots:          v.Bots,
		Pve:           v.Pve,
		Consumables:   v.Consumables,
	}
}

func shopItemPack(it *shop.Item, image func(string) string) shopadmin.ShopItemRequest {
	return shopadmin.ShopItemRequest{
		Code:          it.Code,
		Name:          it.Name,
		Description:   it.Description,
		ImageURL:      image(it.ImageURL),
		Tab:           it.Tab,
		SortOrder:     it.SortOrder,
		IsEnabled:     it.IsEnabled,
		Goods:         it.Goods,
		Price:         it.Price,
		OriginalPrice: it.OriginalPrice,
		Requirements:  it.Requirements,
		Limits:        it.Limits,
		Stock:         it.Stock,
		StartsAt:      utcMinute(it.StartsAt),
		EndsAt:        utcMinute(it.EndsAt),
	}
}

func leaderboardPack(b *leaderboard.Board) leaderboardadmin.LeaderboardRequest {
	return leaderboardadmin.LeaderboardRequest{
		Code:        b.Code,
		Name:        b.Name,
		Description: b.Description,
		Metric:      b.Metric,
		Scope:       b.Scope,
		Period:      b.Period,
		Size:        b.Size,
		MinMatches:  b.MinMatches,
		SortOrder:   b.SortOrder,
		IsEnabled:   b.IsEnabled,
	}
}

// liveLootEntries 掉落池中的条目, 不含已删除物品的条目; 这些条目不参与抽取, 也不导出, 否则导入时找不到物品
func liveLootEntries(tx *gorm.DB) *gorm.DB {
	return tx.Where("item_id = 0 OR item_id IN (?)", tx.Session(&gorm.Session{NewDB: true}).Model(&item.Item{}).Select("id"))
}

func currencyEntry(_ *Run, req *packCurrency) *Entry {
	var current wallet.Currency
	return &Entry{
		Key:   req.Code,
		Name:  req.Name,
		Taken: wallet.CurrencyTaken,
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Where("code = ?", req.Code).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			found := current.Code != ""
			return CompareStatus(found, currencyPack(&current), req), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			if err := ValidateFields(req); err != nil {
				return err
			}
			if !exists {
				target := &wallet.Currency{Code: req.Code}
				walletadmin.ApplyCurrency(target, &req.CurrencyRequest)
				if err := wallet.ValidateCurrency(target); err != nil {
					return err
				}
				return tx.Create(target).Error
			}
			if err := lockCurrency(tx, &current, req.Code); err != nil {
				return err
			}
			old := current
			walletadmin.ApplyCurrency(&current, &req.CurrencyRequest)
			if err := walletadmin.CheckCurrencyChange(tx, &old, &current); err != nil {
				return err
			}
			return tx.Save(&current).Error
		},
	}
}

func qualityEntry(_ *Run, req *itemadmin.CreateQualityRequest) *Entry {
	var current item.Quality
	return &Entry{
		Key:   strconv.Itoa(int(req.Level)),
		Name:  req.Name,
		Taken: item.QualityTaken,
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Where("level = ?", req.Level).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			found := current.Level != 0
			if found && current.Code != req.Code {
				return PackStatusConflict, fmt.Sprintf(PackQualityCodeDiffers, current.Level, current.Code), nil
			}
			return CompareStatus(found, qualityPack(&current), req), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			if err := ValidateFields(req); err != nil {
				return err
			}
			if !exists {
				target := &item.Quality{Level: req.Level, Code: req.Code}
				if err := itemadmin.ApplyQuality(target, &req.QualityRequest); err != nil {
					return err
				}
				return tx.Create(target).Error
			}
			if err := itemadmin.LockQuality(tx, &current, req.Level); err != nil {
				return err
			}
			if err := itemadmin.ApplyQuality(&current, &req.QualityRequest); err != nil {
				return err
			}
			return tx.Save(&current).Error
		},
	}
}

func itemEntry(r *Run, req *packItem) *Entry {
	var current item.Item
	return &Entry{
		Key:   strconv.FormatUint(req.ID, 10),
		Name:  req.Name,
		Taken: item.ItemNameTaken,
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Unscoped().Where("id = ?", req.ID).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			found := current.ID != 0
			if found && current.DeletedAt.Valid {
				return PackStatusConflict, PackItemDeleted, nil
			}
			next := *req
			reason, err := r.CompareImage(upload.PurposeItem, &next.ImageURL)
			if err != nil {
				return "", "", err
			}
			if reason != "" {
				return PackStatusConflict, reason, nil
			}
			return CompareStatus(found, itemPack(&current, SameImage), &next), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			next := req.ItemRequest
			var err error
			if next.ImageURL, err = r.Image(upload.PurposeItem, next.ImageURL); err != nil {
				return err
			}
			if err := ValidateFields(&packItem{ID: req.ID, ItemRequest: next}); err != nil {
				return err
			}
			if !exists {
				target := &item.Item{ID: req.ID}
				if err := itemadmin.ApplyItem(tx, target, &next); err != nil {
					return err
				}
				if !r.dryRun {
					// 沿用包中的 ID, 先把序列推到这个 ID 之后, 之后新建的物品不会撞上
					if err := tx.Exec("SELECT setval(pg_get_serial_sequence('items', 'id'), GREATEST(?, nextval(pg_get_serial_sequence('items', 'id'))))", req.ID).Error; err != nil {
						return err
					}
				}
				return tx.Create(target).Error
			}
			if err := itemadmin.LockItem(tx, &current, req.ID); err != nil {
				return err
			}
			if err := itemadmin.ApplyItem(tx, &current, &next); err != nil {
				return err
			}
			return tx.Save(&current).Error
		},
	}
}

func lootPoolEntry(_ *Run, req *packLootPool) *Entry {
	var current loot.Pool
	return &Entry{
		Key:    req.Code,
		Name:   req.Name,
		Taken:  loot.PoolCodeTaken,
		Rename: func(code string) { req.Code = code },
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Where("code = ?", req.Code).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			if current.ID == 0 {
				return PackStatusNew, "", nil
			}
			var entries []loot.Entry
			if err := liveLootEntries(tx).Where("pool_id = ?", current.ID).Order("id").Find(&entries).Error; err != nil {
				return "", "", err
			}
			return CompareStatus(true, lootPoolPack(&current, entries), req), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			if err := ValidateFields(req); err != nil {
				return err
			}
			target := &loot.Pool{}
			if exists {
				if err := lootadmin.LockLootPool(tx, &current, current.ID); err != nil {
					return err
				}
				target = &current
			}
			if err := lootadmin.ApplyLootPool(target, &req.LootPoolRequest); err != nil {
				return err
			}
			if err := tx.Save(target).Error; err != nil {
				return err
			}
			entries, err := lootadmin.BuildLootEntries(target.ID, req.Entries)
			if err != nil {
				return err
			}
			if err := lootadmin.CheckLootTargets(tx, entries); err != nil {
				return err
			}
			if err := tx.Where("pool_id = ?", target.ID).Delete(&loot.Entry{}).Error; err != nil {
				return err
			}
			if len(entries) == 0 {
				return nil
			}
			return tx.Create(&entries).Error
		},
	}
}

func venueEntry(r *Run, req *venueadmin.VenueRequest) *Entry {
	var current venue.Venue
	return &Entry{
		Key:    req.Code,
		Name:   req.Name,
		Taken:  venue.CodeTaken,
		Rename: func(code string) { req.Code = code },
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Unscoped().Where("code = ?", req.Code).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			found := current.ID != 0
			if found && current.DeletedAt.Valid {
				return PackStatusConflict, fmt.Sprintf(PackCodeDeleted, sectionLabel(PackVenues)), nil
			}
			next := *req
			reason, err := r.CompareImage(upload.PurposeVenue, &next.ImageURL)
			if err != nil {
				return "", "", err
			}
			if reason != "" {
				return PackStatusConflict, reason, nil
			}
			return CompareStatus(found, venuePack(&current, SameImage), &next), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			next := *req
			var err error
			if next.ImageURL, err = r.Image(upload.PurposeVenue, next.ImageURL); err != nil {
				return err
			}
			if err := ValidateFields(&next); err != nil {
				return err
			}
			target := &venue.Venue{}
			if exists {
				if err := venueadmin.LockVenue(tx, &current, current.ID); err != nil {
					return err
				}
				target = &current
			}
			venueadmin.ApplyVenue(target, &next)
			if err := venue.Validate(tx, target); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	}
}

func shopItemEntry(r *Run, req *shopadmin.ShopItemRequest) *Entry {
	// 包中的时间与已有配置按分钟比较
	req.StartsAt, req.EndsAt = utcMinute(req.StartsAt), utcMinute(req.EndsAt)
	var current shop.Item
	return &Entry{
		Key:    req.Code,
		Name:   req.Name,
		Taken:  shop.CodeTaken,
		Rename: func(code string) { req.Code = code },
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Unscoped().Where("code = ?", req.Code).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			found := current.ID != 0
			if found && current.DeletedAt.Valid {
				return PackStatusConflict, fmt.Sprintf(PackCodeDeleted, sectionLabel(PackShopItems)), nil
			}
			next := *req
			if found {
				next.Stock = current.Stock
			}
			reason, err := r.CompareImage(upload.PurposeShop, &next.ImageURL)
			if err != nil {
				return "", "", err
			}
			if reason != "" {
				return PackStatusConflict, reason, nil
			}
			return CompareStatus(found, shopItemPack(&current, SameImage), &next), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			next := *req
			var err error
			if next.ImageURL, err = r.Image(upload.PurposeShop, next.ImageURL); err != nil {
				return err
			}
			if err := ValidateFields(&next); err != nil {
				return err
			}
			target := &shop.Item{}
			if exists {
				if err := shopadmin.LockShopItem(tx, &current, current.ID); err != nil {
					return err
				}
				// 剩余库存随玩家购买变化, 不随配置覆盖
				target, next.Stock = &current, current.Stock
			}
			shopadmin.ApplyShopItem(target, &next)
			if err := shop.Validate(tx, target); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	}
}

func leaderboardEntry(_ *Run, req *leaderboardadmin.LeaderboardRequest) *Entry {
	var current leaderboard.Board
	return &Entry{
		Key:    req.Code,
		Name:   req.Name,
		Taken:  leaderboard.CodeTaken,
		Rename: func(code string) { req.Code = code },
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Where("code = ?", req.Code).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			found := current.ID != 0
			return CompareStatus(found, leaderboardPack(&current), req), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			if err := ValidateFields(req); err != nil {
				return err
			}
			target := &leaderboard.Board{}
			if exists {
				if err := leaderboardadmin.LockLeaderboard(tx, &current, current.ID); err != nil {
					return err
				}
				target = &current
			}
			if err := leaderboardadmin.ApplyLeaderboard(tx, target, req); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	}
}

// settingsEntry 设置组没有新增, 先在保存点中试着导入一次, 按修改的字段判断是否相同
func settingsEntry(r *Run, key string, raw json.RawMessage) *Entry {
	label, _ := settings.ImportLabel(key)
	importGroup := func(tx *gorm.DB) (map[string]interface{}, error) {
		changes, failure, err := settings.Import(r.ctx, tx, r.operatorID, key, raw)
		if failure != nil {
			return nil, failure
		}
		return changes, err
	}
	return &Entry{
		Key:  key,
		Name: label,
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.SavePoint(packSavePoint).Error; err != nil {
				return "", "", err
			}
			changes, err := importGroup(tx)
			if err := tx.RollbackTo(packSavePoint).Error; err != nil {
				return "", "", err
			}
			if err := releasePackSavePoint(tx); err != nil {
				return "", "", err
			}
			switch {
			case err != nil && packFailure(err):
				return PackStatusConflict, err.Error(), nil
			case err != nil:
				return "", "", err
			case len(changes) == 0:
				return PackStatusSame, "", nil
			}
			return PackStatusChanged, "", nil
		},
		Apply: func(tx *gorm.DB, _ bool) error {
			_, err := importGroup(tx)
			return err
		},
	}
}

// lockCurrency 锁定货币, 防止并发修改互相覆盖
func lockCurrency(tx *gorm.DB, target *wallet.Currency, code string) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), code); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(wallet.CurrencyNotFound)
		}
		return err
	}
	return nil
}
