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

package venueadmin

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// QueryVenues 获取地图列表, 不含已删除的地图
func QueryVenues(ctx context.Context, req *ListVenuesRequest) (int64, []venue.Venue, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&venue.Venue{})
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

	venues := []venue.Venue{}
	if err := query.Order("sort_order, id").Offset(offset).Limit(req.Size).Find(&venues).Error; err != nil {
		return 0, nil, err
	}
	return total, venues, nil
}

// doCreateVenue 校验并新建地图
func doCreateVenue(ctx context.Context, req *VenueRequest) (*venue.Venue, error) {
	target := &venue.Venue{}
	ApplyVenue(target, req)
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
		if err := LockVenue(tx, &target, venueID); err != nil {
			return err
		}
		if req.Code != target.Code {
			return errors.New(venue.CodeImmutable)
		}
		old := target
		ApplyVenue(&target, req)
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
		if err := LockVenue(tx, &target, venueID); err != nil {
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
	ApplyVenue(target, req)
	if err := venue.Validate(db.DB(ctx), target); err != nil {
		return nil, err
	}
	return &target.Simulation, nil
}

// LockVenue 锁定地图, 防止并发修改和删除互相覆盖
func LockVenue(tx *gorm.DB, target *venue.Venue, venueID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), venueID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(venue.VenueNotFound)
		}
		return err
	}
	return nil
}

// ApplyVenue 写入地图字段, 由 venue.Validate 校验
func ApplyVenue(target *venue.Venue, req *VenueRequest) {
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
	target.Consumables = req.Consumables
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
		"consumables":    target.Consumables,
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
