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

package botadmin

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// QueryNames 获取人机昵称列表, 按 ID 倒序
func QueryNames(ctx context.Context, req *ListNamesRequest) (int64, []bot.Name, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&bot.Name{})
	if req.Keyword != "" {
		query = query.Where("name ILIKE ?", "%"+db.EscapeLike(req.Keyword)+"%")
	}
	if req.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *req.IsEnabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	names := []bot.Name{}
	if err := query.Order("id DESC").Offset(offset).Limit(req.Size).Find(&names).Error; err != nil {
		return 0, nil, err
	}
	return total, names, nil
}

// cleanNames 去掉首尾空白并校验, 返回去重后的昵称和本次重复的昵称
func cleanNames(raw []string) ([]string, []string, error) {
	switch {
	case len(raw) == 0:
		return nil, nil, &bot.InvalidError{Reason: bot.NamesEmpty}
	case len(raw) > bot.MaxBatchNames:
		return nil, nil, &bot.InvalidError{Reason: bot.NamesTooMany}
	}
	names := make([]string, 0, len(raw))
	var repeated []string
	for i, name := range raw {
		name = strings.TrimSpace(name)
		if bot.ValidateName(name) != nil {
			return nil, nil, &bot.InvalidError{Reason: fmt.Sprintf(bot.NamesInvalid, i+1)}
		}
		if slices.Contains(names, name) {
			repeated = append(repeated, name)
			continue
		}
		names = append(names, name)
	}
	return names, repeated, nil
}

// doCreateNames 批量新建人机昵称, 已存在的和本次重复的跳过
func doCreateNames(ctx context.Context, req *CreateNamesRequest) (*CreateNamesResponseData, error) {
	names, repeated, err := cleanNames(req.Names)
	if err != nil {
		return nil, err
	}
	result := &CreateNamesResponseData{Created: []bot.Name{}, Skipped: append([]string{}, repeated...)}
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var existing []string
		if err := tx.Model(&bot.Name{}).Where("name IN ?", names).Pluck("name", &existing).Error; err != nil {
			return err
		}
		for _, name := range names {
			if slices.Contains(existing, name) {
				result.Skipped = append(result.Skipped, name)
				continue
			}
			result.Created = append(result.Created, bot.Name{Name: name, IsEnabled: req.IsEnabled})
		}
		if len(result.Created) == 0 {
			return nil
		}
		return tx.Create(&result.Created).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(bot.NameTaken)
		}
		return nil, err
	}
	return result, nil
}

// doUpdateName 校验并修改人机昵称, 返回修改后的昵称和本次修改的字段, 没有任何字段变化时不写库
func doUpdateName(ctx context.Context, nameID uint64, req *NameRequest) (*bot.Name, map[string]interface{}, error) {
	var (
		target  bot.Name
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockName(tx, &target, nameID); err != nil {
			return err
		}
		old := target
		ApplyName(&target, req)
		if err := bot.ValidateName(target.Name); err != nil {
			return err
		}
		changes = fieldChanges(nameFields(&old), nameFields(&target))
		if len(changes) == 0 {
			return nil
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, nil, errors.New(bot.NameTaken)
		}
		return nil, nil, err
	}
	return &target, changes, nil
}

// doDeleteName 删除人机昵称, 返回被删除的昵称
func doDeleteName(ctx context.Context, nameID uint64) (*bot.Name, error) {
	var target bot.Name
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockName(tx, &target, nameID); err != nil {
			return err
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// LockName 锁定人机昵称, 防止并发修改和删除互相覆盖
func LockName(tx *gorm.DB, target *bot.Name, nameID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), nameID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(bot.NameNotFound)
		}
		return err
	}
	return nil
}

// ApplyName 写入人机昵称字段, 由 bot.ValidateName 校验
func ApplyName(target *bot.Name, req *NameRequest) {
	target.Name = strings.TrimSpace(req.Name)
	target.IsEnabled = req.IsEnabled
}

// nameFields 人机昵称可修改的字段, 用于操作日志
func nameFields(target *bot.Name) map[string]interface{} {
	return map[string]interface{}{
		"name":       target.Name,
		"is_enabled": target.IsEnabled,
	}
}

// QueryPersonas 获取人机风格列表, 按 ID 排列
func QueryPersonas(ctx context.Context, req *ListPersonasRequest) (int64, []bot.Persona, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&bot.Persona{})
	if req.Keyword != "" {
		keyword := "%" + db.EscapeLike(req.Keyword) + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", keyword, keyword)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	personas := []bot.Persona{}
	if err := query.Order("id").Offset(offset).Limit(req.Size).Find(&personas).Error; err != nil {
		return 0, nil, err
	}
	return total, personas, nil
}

// doCreatePersona 校验并新建人机风格
func doCreatePersona(ctx context.Context, req *PersonaRequest) (*bot.Persona, error) {
	target := &bot.Persona{}
	ApplyPersona(target, req)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := bot.Validate(tx, target); err != nil {
			return err
		}
		return tx.Create(target).Error
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return nil, errors.New(bot.CodeTaken)
		}
		return nil, err
	}
	return target, nil
}

// doUpdatePersona 校验并修改人机风格, 返回修改后的风格和本次修改的字段, 没有任何字段变化时不写库
func doUpdatePersona(ctx context.Context, personaID uint64, req *PersonaRequest) (*bot.Persona, map[string]interface{}, error) {
	var (
		target  bot.Persona
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockPersona(tx, &target, personaID); err != nil {
			return err
		}
		if req.Code != target.Code {
			return errors.New(bot.CodeImmutable)
		}
		old := target
		ApplyPersona(&target, req)
		if err := bot.Validate(tx, &target); err != nil {
			return err
		}
		changes = fieldChanges(personaFields(&old), personaFields(&target))
		if len(changes) == 0 {
			return nil
		}
		return tx.Save(&target).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return &target, changes, nil
}

// doDeletePersona 软删除人机风格, 返回被删除的风格; 被地图引用时返回 *InUseError
//
// 保存地图时会对引用的风格加共享锁, 这里先锁住风格再查地图, 不会漏掉并发保存的地图。
func doDeletePersona(ctx context.Context, personaID uint64) (*bot.Persona, error) {
	var target bot.Persona
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockPersona(tx, &target, personaID); err != nil {
			return err
		}
		using, err := venue.UsingPersona(tx, target.Code)
		if err != nil {
			return err
		}
		if using != "" {
			return &InUseError{Venue: using}
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// LockPersona 锁定人机风格, 防止并发修改和删除互相覆盖
func LockPersona(tx *gorm.DB, target *bot.Persona, personaID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), personaID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(bot.PersonaNotFound)
		}
		return err
	}
	return nil
}

// ApplyPersona 写入人机风格字段, 由 bot.Validate 校验
func ApplyPersona(target *bot.Persona, req *PersonaRequest) {
	target.Code = req.Code
	target.Name = strings.TrimSpace(req.Name)
	target.Description = strings.TrimSpace(req.Description)
	target.Strategy = req.Strategy
	target.Params = req.Params
}

// personaFields 人机风格可修改的字段, 用于操作日志
func personaFields(target *bot.Persona) map[string]interface{} {
	return map[string]interface{}{
		"code":        target.Code,
		"name":        target.Name,
		"description": target.Description,
		"strategy":    target.Strategy,
		"params":      target.Params,
	}
}

// fieldChanges 本次修改了哪些字段, 用于操作日志; 参数是 map, 按内容比较
func fieldChanges(old, next map[string]interface{}) map[string]interface{} {
	changes := map[string]interface{}{}
	for key, oldValue := range old {
		if newValue := next[key]; !reflect.DeepEqual(oldValue, newValue) {
			changes[key] = map[string]interface{}{"old": oldValue, "new": newValue}
		}
	}
	return changes
}
