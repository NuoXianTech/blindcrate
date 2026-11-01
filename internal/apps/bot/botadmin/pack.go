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

	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"gorm.io/gorm"
)

// 配置包中的人机配置, 代码见 bot.PackPersonas 和 bot.PackNames
const (
	OrderBotPersonas = 45 // 排在地图之前, 地图按代码引用风格
	OrderBotNames    = 46
	personaLabel     = "人机风格"
)

func init() {
	pack.RegisterSection(pack.Section{
		Code: bot.PackPersonas, Label: personaLabel, Hint: "按代码对应", Order: OrderBotPersonas, Renamable: true,
		Export: func(_ context.Context, tx *gorm.DB, _ func(string) string) (any, int, error) {
			var rows []bot.Persona
			if err := tx.Order("id").Find(&rows).Error; err != nil {
				return nil, 0, err
			}
			list := make([]PersonaRequest, len(rows))
			for i := range rows {
				list[i] = personaPack(&rows[i])
			}
			return list, len(list), nil
		},
		Decode:  pack.DecodeList(personaEntry, nil),
		Invalid: invalid,
	})
	pack.RegisterSection(pack.Section{
		Code: bot.PackNames, Label: "人机昵称", Hint: "按昵称对应", Order: OrderBotNames,
		Export: func(_ context.Context, tx *gorm.DB, _ func(string) string) (any, int, error) {
			var rows []bot.Name
			if err := tx.Order("id").Find(&rows).Error; err != nil {
				return nil, 0, err
			}
			list := make([]NameRequest, len(rows))
			for i := range rows {
				list[i] = namePack(&rows[i])
			}
			return list, len(list), nil
		},
		Decode:  pack.DecodeList(nameEntry, nil),
		Invalid: invalid,
	})
}

// invalid 写入时返回的错误是否为人机配置的校验错误
func invalid(err error) bool {
	var target *bot.InvalidError
	return errors.As(err, &target)
}

// personaPack 风格写入配置包的内容
func personaPack(p *bot.Persona) PersonaRequest {
	return PersonaRequest{Code: p.Code, Name: p.Name, Description: p.Description, Strategy: p.Strategy, Params: p.Params}
}

// personaEntry 包中的一个风格, 按代码对应已有风格; 同代码的风格已删除时只能改名导入
func personaEntry(_ *pack.Run, req *PersonaRequest) *pack.Entry {
	var current bot.Persona
	return &pack.Entry{
		Key:    req.Code,
		Name:   req.Name,
		Taken:  bot.CodeTaken,
		Rename: func(code string) { req.Code = code },
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Unscoped().Where("code = ?", req.Code).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			found := current.ID != 0
			if found && current.DeletedAt.Valid {
				return pack.PackStatusConflict, fmt.Sprintf(pack.PackCodeDeleted, personaLabel), nil
			}
			return pack.CompareStatus(found, personaPack(&current), req), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			if err := pack.ValidateFields(req); err != nil {
				return err
			}
			target := &bot.Persona{}
			if exists {
				if err := LockPersona(tx, &current, current.ID); err != nil {
					return err
				}
				target = &current
			}
			ApplyPersona(target, req)
			if err := bot.Validate(tx, target); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	}
}

// namePack 昵称写入配置包的内容
func namePack(n *bot.Name) NameRequest {
	return NameRequest{Name: n.Name, IsEnabled: n.IsEnabled}
}

// nameEntry 包中的一个昵称, 按昵称对应已有昵称
func nameEntry(_ *pack.Run, req *NameRequest) *pack.Entry {
	var current bot.Name
	return &pack.Entry{
		Key:   req.Name,
		Name:  req.Name,
		Taken: bot.NameTaken,
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.Where("name = ?", req.Name).Limit(1).Find(&current).Error; err != nil {
				return "", "", err
			}
			return pack.CompareStatus(current.ID != 0, namePack(&current), req), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			if err := pack.ValidateFields(req); err != nil {
				return err
			}
			target := &bot.Name{}
			if exists {
				if err := LockName(tx, &current, current.ID); err != nil {
					return err
				}
				target = &current
			}
			ApplyName(target, req)
			if err := bot.ValidateName(target.Name); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	}
}
