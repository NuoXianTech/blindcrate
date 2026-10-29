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

package season

import (
	"errors"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"gorm.io/gorm"
)

func init() {
	// 进行中的赛季在结算对局时额外写入赛季周期
	stats.RegisterPeriod(func(tx *gorm.DB, at time.Time) (string, error) {
		current, err := Current(tx, at)
		if err != nil || current == nil {
			return "", err
		}
		return Period(current.Code), nil
	})
}

// codePattern 赛季代码, 小写字母、数字和下划线; 拼上前缀后不能超过 stats.period 的长度
var codePattern = regexp.MustCompile(`^[a-z0-9_]{1,24}$`)

// Period 赛季的统计周期, 如 season:s1
func Period(code string) string {
	return periodPrefix + code
}

// Validate 保存前校验赛季, 名称需已去掉首尾空白; 不检查与其他赛季是否重叠, 见 CheckOverlap
func Validate(s *Season) error {
	if !codePattern.MatchString(s.Code) {
		return errors.New(CodeInvalid)
	}
	if n := utf8.RuneCountInString(s.Name); n < 1 || n > 64 {
		return errors.New(NameInvalid)
	}
	if !s.EndsAt.After(s.StartsAt) {
		return errors.New(TimeInvalid)
	}
	return nil
}

// Lock 锁定赛季表直到事务结束, 新建和修改赛季时间前调用, 防止并发保存的两个赛季互相重叠
//
// 只阻塞其他写入, 不影响读取。
func Lock(tx *gorm.DB) error {
	return tx.Exec("LOCK TABLE seasons IN SHARE ROW EXCLUSIVE MODE").Error
}

// CheckOverlap 检查赛季的时间是否与其他赛季重叠, 需要先调用 Lock
func CheckOverlap(tx *gorm.DB, s *Season) error {
	var count int64
	if err := tx.Model(&Season{}).
		Where("id <> ? AND starts_at < ? AND ends_at > ?", s.ID, s.EndsAt, s.StartsAt).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New(TimeOverlapped)
	}
	return nil
}

// Current 获取 at 时进行中的赛季, 没有时返回 nil
func Current(tx *gorm.DB, at time.Time) (*Season, error) {
	return first(tx.Where("starts_at <= ? AND ends_at > ?", at, at).Order("starts_at DESC"))
}

// Latest 获取 at 时最近一个已开始的赛季, 可能已经结束; 没有时返回 nil
//
// 用于赛季排行榜: 赛季之间的间隔中展示刚结束的赛季。
func Latest(tx *gorm.DB, at time.Time) (*Season, error) {
	return first(tx.Where("starts_at <= ?", at).Order("starts_at DESC"))
}

// Next 获取 at 之后最早开始的赛季, 没有时返回 nil
func Next(tx *gorm.DB, at time.Time) (*Season, error) {
	return first(tx.Where("starts_at > ?", at).Order("starts_at"))
}

// first 获取查询到的第一个赛季, 没有时返回 nil
func first(query *gorm.DB) (*Season, error) {
	var seasons []Season
	if err := query.Limit(1).Find(&seasons).Error; err != nil {
		return nil, err
	}
	if len(seasons) == 0 {
		return nil, nil
	}
	return &seasons[0], nil
}
