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
	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"gorm.io/gorm"
)

// 配置包中的人机配置, 代码见 bot.PackPersonas 和 bot.PackNames
const (
	OrderBotPersonas = 45 // 排在地图之前, 地图按代码引用风格
	OrderBotNames    = 46
)

func init() {
	// 风格按代码对应; 同代码的风格已删除时只能改名导入
	pack.RegisterTable(pack.Table[bot.Persona, PersonaRequest]{
		Code: bot.PackPersonas, Label: "人机风格", Hint: "按代码对应", Order: OrderBotPersonas,
		Query: func(tx *gorm.DB) *gorm.DB { return tx.Order("id") },
		Pack: func(p *bot.Persona, _ func(string) string) PersonaRequest {
			return PersonaRequest{Code: p.Code, Name: p.Name, Description: p.Description, Strategy: p.Strategy, Params: p.Params}
		},
		Key:    func(req *PersonaRequest) string { return req.Code },
		Name:   func(req *PersonaRequest) string { return req.Name },
		Taken:  bot.CodeTaken,
		Rename: func(req *PersonaRequest, code string) { req.Code = code },
		Find: func(tx *gorm.DB, req *PersonaRequest, current *bot.Persona) (bool, error) {
			err := tx.Unscoped().Where("code = ?", req.Code).Limit(1).Find(current).Error
			return current.ID != 0, err
		},
		Deleted: func(p *bot.Persona) bool { return p.DeletedAt.Valid },
		Lock:    func(tx *gorm.DB, current *bot.Persona) error { return LockPersona(tx, current, current.ID) },
		Save: func(tx *gorm.DB, _ *pack.Run, current *bot.Persona, req *PersonaRequest) error {
			target := current
			if target == nil {
				target = &bot.Persona{}
			}
			ApplyPersona(target, req)
			if err := bot.Validate(tx, target); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	})

	// 昵称按昵称对应
	pack.RegisterTable(pack.Table[bot.Name, NameRequest]{
		Code: bot.PackNames, Label: "人机昵称", Hint: "按昵称对应", Order: OrderBotNames,
		Query: func(tx *gorm.DB) *gorm.DB { return tx.Order("id") },
		Pack: func(n *bot.Name, _ func(string) string) NameRequest {
			return NameRequest{Name: n.Name, IsEnabled: n.IsEnabled}
		},
		Key:   func(req *NameRequest) string { return req.Name },
		Name:  func(req *NameRequest) string { return req.Name },
		Taken: bot.NameTaken,
		Find: func(tx *gorm.DB, req *NameRequest, current *bot.Name) (bool, error) {
			err := tx.Where("name = ?", req.Name).Limit(1).Find(current).Error
			return current.ID != 0, err
		},
		Lock: func(tx *gorm.DB, current *bot.Name) error { return LockName(tx, current, current.ID) },
		Save: func(tx *gorm.DB, _ *pack.Run, current *bot.Name, req *NameRequest) error {
			target := current
			if target == nil {
				target = &bot.Name{}
			}
			ApplyName(target, req)
			if err := bot.ValidateName(target.Name); err != nil {
				return err
			}
			return tx.Save(target).Error
		},
	})
}
