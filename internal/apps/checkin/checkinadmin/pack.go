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

package checkinadmin

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/checkin"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"gorm.io/gorm"
)

// 配置包中的签到奖励
const (
	PackCheckinDays  = "checkin_days"
	OrderCheckinDays = 80 // 排在任务之后, 奖励引用货币和物品
)

func init() {
	pack.RegisterSection(pack.Section{
		Code: PackCheckinDays, Label: "签到奖励", Hint: "按天对应; 包中没有的天数保留", Order: OrderCheckinDays,
		Export: func(_ context.Context, tx *gorm.DB, _ func(string) string) (any, int, error) {
			days, err := checkin.QueryDays(tx)
			if err != nil {
				return nil, 0, err
			}
			list := make([]DayPack, len(days))
			for i := range days {
				list[i] = dayPack(&days[i])
			}
			return list, len(list), nil
		},
		Decode: pack.DecodeList(dayEntry, nil),
		Invalid: func(err error) bool {
			var invalid *checkin.InvalidError
			return errors.As(err, &invalid)
		},
	})
}

// DayPack 配置包中的一天
type DayPack struct {
	Day    int16         `json:"day" binding:"min=1,max=31"`
	Reward reward.Bundle `json:"reward" binding:"-"`
}

// dayPack 某一天的奖励写入配置包的内容
func dayPack(d *checkin.Day) DayPack {
	return DayPack{Day: d.Day, Reward: d.Reward}
}

// dayEntry 包中的一天, 按天对应已有的奖励; 新增的天数需要与已有的连续, 包中按天排列时依次导入即可
func dayEntry(_ *pack.Run, req *DayPack) *pack.Entry {
	var current checkin.Day
	return &pack.Entry{
		Key:  strconv.Itoa(int(req.Day)),
		Name: fmt.Sprintf("第 %d 天", req.Day),
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Where("day = ?", req.Day).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			found := current.Day != 0
			return pack.CompareStatus(found, dayPack(&current), req), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			if err := pack.ValidateFields(req); err != nil {
				return err
			}
			if err := lockDays(tx); err != nil {
				return err
			}
			target := &checkin.Day{Day: req.Day}
			if exists {
				if err := lockDay(tx, &current, req.Day); err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// 比较之后被后台删掉了
						return &checkin.InvalidError{Reason: fmt.Sprintf(checkin.DayGap, req.Day)}
					}
					return err
				}
				target = &current
			} else {
				var before int64
				if err := tx.Model(&checkin.Day{}).Where("day < ?", req.Day).Count(&before).Error; err != nil {
					return err
				}
				if before != int64(req.Day-1) {
					return &checkin.InvalidError{Reason: fmt.Sprintf(checkin.DayGap, req.Day)}
				}
			}
			target.Reward = req.Reward
			if err := checkin.ValidateDay(tx, target); err != nil {
				return err
			}
			if exists {
				return tx.Save(target).Error
			}
			return tx.Create(target).Error
		},
	}
}
