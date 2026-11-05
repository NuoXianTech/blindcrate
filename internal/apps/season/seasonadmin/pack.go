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

package seasonadmin

import (
	"context"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"gorm.io/gorm"
)

// 配置包中的赛季
const (
	PackSeasons  = "seasons"
	OrderSeasons = 90 // 排在物品之后, 赛季奖励引用货币和物品
)

func init() {
	pack.RegisterSection(pack.Section{
		Code: PackSeasons, Label: "赛季", Hint: "按代码对应, 含时间、段位分继承比例和奖励; 已结束的赛季只导出, 导入时只能新增或修改未结束的赛季", Order: OrderSeasons,
		Export: func(_ context.Context, tx *gorm.DB, _ func(string) string) (any, int, error) {
			var rows []season.Season
			if err := tx.Order("starts_at, id").Find(&rows).Error; err != nil {
				return nil, 0, err
			}
			list := make([]CreateSeasonRequest, len(rows))
			for i := range rows {
				list[i] = seasonPack(&rows[i])
			}
			return list, len(list), nil
		},
		Decode: pack.DecodeList(seasonEntry, nil),
	})
}

// seasonPack 赛季写入配置包的内容, 时间统一为 UTC, 便于比较
func seasonPack(s *season.Season) CreateSeasonRequest {
	return CreateSeasonRequest{
		Code: s.Code,
		SeasonRequest: SeasonRequest{
			Name:        s.Name,
			StartsAt:    s.StartsAt.UTC(),
			EndsAt:      s.EndsAt.UTC(),
			RatingCarry: s.RatingCarry,
			Rewards:     s.Rewards,
		},
	}
}

// seasonEntry 包中的一个赛季, 按代码对应已有赛季; 与后台新建、修改的规则相同, 如时间不能重叠、已开始的赛季不能改开始时间
func seasonEntry(_ *pack.Run, req *CreateSeasonRequest) *pack.Entry {
	// 与导出一致: 精确到分钟的 UTC 时间, 包中写成其他时区时比较结果不变
	req.StartsAt = req.StartsAt.Truncate(time.Minute).UTC()
	req.EndsAt = req.EndsAt.Truncate(time.Minute).UTC()
	var current season.Season
	return &pack.Entry{
		Key:   req.Code,
		Name:  req.Name,
		Taken: season.CodeTaken,
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Where("code = ?", req.Code).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			found := current.ID != 0
			status := pack.CompareStatus(found, seasonPack(&current), req)
			// 已结束的赛季不能再新建, 提前标出, 不用等到写入时才失败
			if !found && !req.EndsAt.After(time.Now()) {
				return pack.PackStatusConflict, season.TimeInvalid, nil
			}
			return status, "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			if err := pack.ValidateFields(req); err != nil {
				return err
			}
			now := time.Now()
			if !exists {
				_, err := createSeason(tx, req, now)
				return err
			}
			if err := season.Lock(tx); err != nil {
				return err
			}
			if err := lockSeason(tx, &current, current.ID); err != nil {
				return err
			}
			_, err := updateSeason(tx, &current, &req.SeasonRequest, now)
			return err
		},
	}
}
