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
	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"gorm.io/gorm"
)

func init() {
	pack.RegisterTable(pack.Table[venue.Venue, VenueRequest]{
		Code: pack.PackVenues, Label: "地图", Hint: "按代码对应, 带上地图图片", Order: pack.OrderVenues,
		Query: func(tx *gorm.DB) *gorm.DB { return tx.Order("sort_order, id") },
		Pack: func(v *venue.Venue, image func(string) string) VenueRequest {
			return VenueRequest{
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
				Lots:          v.Lots,
				Auction:       v.Auction,
				Bots:          v.Bots,
				Pve:           v.Pve,
				Consumables:   v.Consumables,
				Host:          v.Host,
			}
		},
		Key:     func(req *VenueRequest) string { return req.Code },
		Name:    func(req *VenueRequest) string { return req.Name },
		Taken:   venue.CodeTaken,
		Rename:  func(req *VenueRequest, code string) { req.Code = code },
		Renamed: venueRenamed,
		Find: func(tx *gorm.DB, req *VenueRequest, current *venue.Venue) (bool, error) {
			err := tx.Unscoped().Where("code = ?", req.Code).Limit(1).Find(current).Error
			return current.ID != 0, err
		},
		Deleted: func(v *venue.Venue) bool { return v.DeletedAt.Valid },
		Image:   func(req *VenueRequest) (string, *string) { return venue.ImagePurpose, &req.ImageURL },
		Lock:    func(tx *gorm.DB, current *venue.Venue) error { return LockVenue(tx, current, current.ID) },
		Save: func(tx *gorm.DB, _ *pack.Run, current *venue.Venue, req *VenueRequest) error {
			target := current
			if target == nil {
				target = &venue.Venue{}
			}
			ApplyVenue(target, req)
			if err := venue.Validate(tx, target); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	})
}

// venueRenamed 地图引用的掉落池和人机风格改名, 以及入场条件中引用的配置改名
func venueRenamed(req *VenueRequest, section, old, next string) {
	switch section {
	case pack.PackLootPools:
		for t := range req.Lots.Tiers {
			slots := req.Lots.Tiers[t].Depot.Slots
			for i := range slots {
				if slots[i].Pool == old {
					slots[i].Pool = next
				}
			}
		}
	case bot.PackPersonas:
		for _, weights := range [][]bot.Weight{req.Bots.Pve, req.Bots.Ranked} {
			for i := range weights {
				if weights[i].Persona == old {
					weights[i].Persona = next
				}
			}
		}
	}
	pack.RenameConditions(req.Requirements, section, old, next)
}
