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

package rankedadmin

import (
	"context"
	"strconv"
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"github.com/NuoXianTech/blindcrate/internal/apps/ranked"
	"gorm.io/gorm"
)

// 配置包中的段位
const (
	PackRankedTiers  = "ranked_tiers"
	OrderRankedTiers = 80 // 段位不引用其他配置, 也不被引用
)

func init() {
	pack.RegisterSection(pack.Section{
		Code: PackRankedTiers, Label: "段位", Hint: "按分数线对应", Order: OrderRankedTiers,
		Export: func(_ context.Context, tx *gorm.DB, _ func(string) string) (any, int, error) {
			rows, err := ranked.QueryTiers(tx)
			if err != nil {
				return nil, 0, err
			}
			list := make([]RankedTierRequest, len(rows))
			for i := range rows {
				list[i] = rankedTierPack(&rows[i])
			}
			return list, len(list), nil
		},
		Decode: pack.DecodeList(rankedTierEntry, nil),
	})
}

// rankedTierPack 段位写入配置包的内容
func rankedTierPack(t *ranked.Tier) RankedTierRequest {
	return RankedTierRequest{Name: t.Name, MinRating: t.MinRating, Color: t.Color}
}

// rankedTierEntry 包中的一个段位, 按分数线对应已有段位; 与后台新建、修改的规则相同
func rankedTierEntry(_ *pack.Run, req *RankedTierRequest) *pack.Entry {
	// 与导出一致: 名称去掉首尾空白, 颜色为小写, 包中写成其他形式时比较结果不变
	req.Name, req.Color = strings.TrimSpace(req.Name), strings.ToLower(req.Color)
	var current ranked.Tier
	return &pack.Entry{
		Key:   strconv.Itoa(int(req.MinRating)),
		Name:  req.Name,
		Taken: ranked.TierRatingTaken,
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Where("min_rating = ?", req.MinRating).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			return pack.CompareStatus(current.ID != 0, rankedTierPack(&current), req), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			if err := pack.ValidateFields(req); err != nil {
				return err
			}
			target := &ranked.Tier{}
			if exists {
				if err := lockRankedTier(tx, &current, current.ID); err != nil {
					return err
				}
				target = &current
			}
			if err := applyRankedTier(target, req); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	}
}
