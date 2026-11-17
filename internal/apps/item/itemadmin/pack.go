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
	"fmt"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/effect"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"github.com/NuoXianTech/blindcrate/internal/apps/upload"
	"gorm.io/gorm"
)

// ItemPack 配置包中的一个物品, 按 ID 对应, 导入时沿用包中的 ID
type ItemPack struct {
	ID uint64 `json:"id" binding:"required"`
	ItemRequest
}

func init() {
	// 品质按等级对应; 等级相同而代码不同的不能导入, 代码不能修改
	pack.RegisterTable(pack.Table[item.Quality, CreateQualityRequest]{
		Code: pack.PackQualities, Label: "品质", Hint: "按等级对应", Order: pack.OrderQualities,
		Query: func(tx *gorm.DB) *gorm.DB { return tx.Order("level") },
		Pack: func(q *item.Quality, _ func(string) string) CreateQualityRequest {
			return CreateQualityRequest{Level: q.Level, Code: q.Code, QualityRequest: QualityRequest{
				Name:      q.Name,
				Color:     q.Color,
				Effect:    q.Effect,
				ScanMs:    q.ScanMs,
				SortOrder: q.SortOrder,
			}}
		},
		Key:   func(req *CreateQualityRequest) string { return strconv.Itoa(int(req.Level)) },
		Name:  func(req *CreateQualityRequest) string { return req.Name },
		Taken: item.QualityTaken,
		Find: func(tx *gorm.DB, req *CreateQualityRequest, current *item.Quality) (bool, error) {
			err := tx.Where("level = ?", req.Level).Limit(1).Find(current).Error
			return current.Level != 0, err
		},
		Check: func(_ *gorm.DB, current *item.Quality, found bool, req *CreateQualityRequest) (string, error) {
			if found && current.Code != req.Code {
				return fmt.Sprintf(item.QualityCodeDiffers, current.Level, current.Code), nil
			}
			return "", nil
		},
		Lock: func(tx *gorm.DB, current *item.Quality) error { return LockQuality(tx, current, current.Level) },
		Save: func(tx *gorm.DB, _ *pack.Run, current *item.Quality, req *CreateQualityRequest) error {
			if current == nil {
				target := &item.Quality{Level: req.Level, Code: req.Code}
				if err := ApplyQuality(target, &req.QualityRequest); err != nil {
					return err
				}
				return tx.Create(target).Error
			}
			if err := ApplyQuality(current, &req.QualityRequest); err != nil {
				return err
			}
			return tx.Save(current).Error
		},
	})

	// 物品按 ID 对应: 掉落池、收集组和奖励都按 ID 引用物品, 所以不能改名导入
	pack.RegisterTable(pack.Table[item.Item, ItemPack]{
		Code: pack.PackItems, Label: "物品", Hint: "按 ID 对应, 带上物品图片", Order: pack.OrderItems,
		Query: func(tx *gorm.DB) *gorm.DB { return tx.Order("id") },
		Pack: func(it *item.Item, image func(string) string) ItemPack {
			return ItemPack{ID: it.ID, ItemRequest: ItemRequest{
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
		},
		Key:   func(req *ItemPack) string { return strconv.FormatUint(req.ID, 10) },
		Name:  func(req *ItemPack) string { return req.Name },
		Taken: item.ItemNameTaken,
		// 效果参数中引用的掉落池(如箱子的掉落池)跟着掉落池一起改名
		Renamed: func(req *ItemPack, section, old, next string) {
			if section == pack.PackLootPools {
				effect.Renamed(req.Effect, req.EffectParams, loot.SourcePool, old, next)
			}
		},
		Find: func(tx *gorm.DB, req *ItemPack, current *item.Item) (bool, error) {
			err := tx.Unscoped().Where("id = ?", req.ID).Limit(1).Find(current).Error
			return current.ID != 0, err
		},
		Deleted: func(it *item.Item) bool { return it.DeletedAt.Valid },
		Image:   func(req *ItemPack) (string, *string) { return upload.PurposeItem, &req.ImageURL },
		Lock:    func(tx *gorm.DB, current *item.Item) error { return LockItem(tx, current, current.ID) },
		Save: func(tx *gorm.DB, r *pack.Run, current *item.Item, req *ItemPack) error {
			if current != nil {
				if err := ApplyItem(tx, current, &req.ItemRequest); err != nil {
					return err
				}
				return tx.Save(current).Error
			}
			target := &item.Item{ID: req.ID}
			if err := ApplyItem(tx, target, &req.ItemRequest); err != nil {
				return err
			}
			if !r.DryRun() {
				// 沿用包中的 ID, 先把序列推到这个 ID 之后, 之后新建的物品不会撞上
				if err := tx.Exec("SELECT setval(pg_get_serial_sequence('items', 'id'), GREATEST(?, nextval(pg_get_serial_sequence('items', 'id'))))", req.ID).Error; err != nil {
					return err
				}
			}
			return tx.Create(target).Error
		},
	})
}
