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
	"reflect"

	"github.com/NuoXianTech/blindcrate/internal/apps/checkin"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// lockDays 锁住签到奖励表, 整体保存和配置包导入互斥; 不影响签到时读取
func lockDays(tx *gorm.DB) error {
	return tx.Exec("LOCK TABLE " + checkin.Day{}.TableName() + " IN EXCLUSIVE MODE").Error
}

// doUpdateDays 整体保存签到奖励, 第 i 项为第 i+1 天; 多出的天数删除。返回保存后的奖励和本次修改的内容, 没有变化时不写库
func doUpdateDays(ctx context.Context, rewards []reward.Bundle) ([]checkin.Day, map[string]interface{}, error) {
	if len(rewards) < 1 || len(rewards) > checkin.MaxDays {
		return nil, nil, &checkin.InvalidError{Reason: checkin.DaysInvalid}
	}
	var (
		saved   []checkin.Day
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockDays(tx); err != nil {
			return err
		}
		old, err := checkin.QueryDays(tx)
		if err != nil {
			return err
		}
		next := make([]checkin.Day, len(rewards))
		for i := range rewards {
			next[i] = checkin.Day{Day: int16(i + 1), Reward: rewards[i]}
			if i < len(old) {
				// 保留原来的创建时间
				next[i].CreatedAt = old[i].CreatedAt
			}
			if err := checkin.ValidateDay(tx, &next[i]); err != nil {
				return err
			}
		}
		saved = next
		if reflect.DeepEqual(dayRewards(old), dayRewards(next)) {
			saved = old
			return nil
		}
		changes = map[string]interface{}{"days": map[string]interface{}{"old": dayRewards(old), "new": dayRewards(next)}}
		if err := tx.Where("day > ?", len(next)).Delete(&checkin.Day{}).Error; err != nil {
			return err
		}
		for i := range next {
			if i < len(old) {
				if err := tx.Save(&next[i]).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Create(&next[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return saved, changes, nil
}

// dayRewards 每一天的奖励, 用于比较和操作日志
func dayRewards(days []checkin.Day) []reward.Bundle {
	rewards := make([]reward.Bundle, len(days))
	for i := range days {
		rewards[i] = days[i].Reward
	}
	return rewards
}

// lockDay 锁住某一天的奖励, 不存在时返回 gorm.ErrRecordNotFound
func lockDay(tx *gorm.DB, target *checkin.Day, day int16) error {
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("day = ?", day).Take(target).Error
}
