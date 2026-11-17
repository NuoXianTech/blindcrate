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
	// 赛季按代码对应; 与后台新建、修改的规则相同, 如时间不能重叠、已开始的赛季不能改开始时间
	pack.RegisterTable(pack.Table[season.Season, CreateSeasonRequest]{
		Code: PackSeasons, Label: "赛季", Hint: "按代码对应, 含时间、段位分继承比例和奖励; 已结束的赛季只导出, 导入时只能新增或修改未结束的赛季", Order: OrderSeasons,
		Query: func(tx *gorm.DB) *gorm.DB { return tx.Order("starts_at, id") },
		// 时间统一为 UTC, 便于比较
		Pack: func(s *season.Season, _ func(string) string) CreateSeasonRequest {
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
		},
		Key:   func(req *CreateSeasonRequest) string { return req.Code },
		Name:  func(req *CreateSeasonRequest) string { return req.Name },
		Taken: season.CodeTaken,
		// 与导出一致: 精确到分钟的 UTC 时间, 包中写成其他时区时比较结果不变
		Prepare: func(req *CreateSeasonRequest) {
			req.StartsAt = req.StartsAt.Truncate(time.Minute).UTC()
			req.EndsAt = req.EndsAt.Truncate(time.Minute).UTC()
		},
		Find: func(tx *gorm.DB, req *CreateSeasonRequest, current *season.Season) (bool, error) {
			err := tx.Where("code = ?", req.Code).Limit(1).Find(current).Error
			return current.ID != 0, err
		},
		// 已结束的赛季不能再新建, 提前标出, 不用等到写入时才失败
		Check: func(_ *gorm.DB, _ *season.Season, found bool, req *CreateSeasonRequest) (string, error) {
			if !found && !req.EndsAt.After(time.Now()) {
				return season.TimeInvalid, nil
			}
			return "", nil
		},
		// 修改时先锁赛季表再锁这一行, 与后台修改的顺序相同
		Save: func(tx *gorm.DB, _ *pack.Run, current *season.Season, req *CreateSeasonRequest) error {
			now := time.Now()
			if current == nil {
				_, err := createSeason(tx, req, now)
				return err
			}
			if err := season.Lock(tx); err != nil {
				return err
			}
			if err := lockSeason(tx, current, current.ID); err != nil {
				return err
			}
			_, err := updateSeason(tx, current, &req.SeasonRequest, now)
			return err
		},
	})
}
