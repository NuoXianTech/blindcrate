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

package shopadmin

import (
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"github.com/NuoXianTech/blindcrate/internal/apps/shop"
	"gorm.io/gorm"
)

func init() {
	pack.RegisterTable(pack.Table[shop.Item, ShopItemRequest]{
		Code: pack.PackShopItems, Label: "商品", Hint: "按代码对应, 带上商品图片; 导入时已有商品的剩余库存不变", Order: pack.OrderShopItems,
		Query: func(tx *gorm.DB) *gorm.DB { return tx.Order("sort_order, id") },
		Pack: func(it *shop.Item, image func(string) string) ShopItemRequest {
			return ShopItemRequest{
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
		},
		Key:    func(req *ShopItemRequest) string { return req.Code },
		Name:   func(req *ShopItemRequest) string { return req.Name },
		Taken:  shop.CodeTaken,
		Rename: func(req *ShopItemRequest, code string) { req.Code = code },
		Renamed: func(req *ShopItemRequest, section, old, next string) {
			if section == pack.PackVenues {
				pack.RenameConditionScopes(req.Requirements, old, next)
			}
		},
		// 包中的时间与已有配置按分钟比较
		Prepare: func(req *ShopItemRequest) {
			req.StartsAt, req.EndsAt = utcMinute(req.StartsAt), utcMinute(req.EndsAt)
		},
		Find: func(tx *gorm.DB, req *ShopItemRequest, current *shop.Item) (bool, error) {
			err := tx.Unscoped().Where("code = ?", req.Code).Limit(1).Find(current).Error
			return current.ID != 0, err
		},
		Deleted: func(it *shop.Item) bool { return it.DeletedAt.Valid },
		// 剩余库存随玩家购买变化, 不随配置覆盖
		Keep:  func(current *shop.Item, req *ShopItemRequest) { req.Stock = current.Stock },
		Image: func(req *ShopItemRequest) (string, *string) { return shop.ImagePurpose, &req.ImageURL },
		Lock:  func(tx *gorm.DB, current *shop.Item) error { return LockShopItem(tx, current, current.ID) },
		Save: func(tx *gorm.DB, _ *pack.Run, current *shop.Item, req *ShopItemRequest) error {
			target := current
			if target == nil {
				target = &shop.Item{}
			}
			ApplyShopItem(target, req)
			if err := shop.Validate(tx, target); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	})
}

// utcMinute 时间去掉秒并转为 UTC, 包中的时间与已有配置按这个格式比较
func utcMinute(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	utc := t.UTC().Truncate(time.Minute)
	return &utc
}
