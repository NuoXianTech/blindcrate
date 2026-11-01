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

package bot

import (
	"maps"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"gorm.io/gorm"
)

// Name 人机的一个昵称, 开局时从启用的昵称中随机选取; 删除为物理删除, 已开局的对局中保存的是昵称本身
type Name struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:32;not null;uniqueIndex:uniq_bot_names_name" json:"name"`
	IsEnabled bool      `gorm:"not null;default:false" json:"is_enabled"` // 停用后开局时不再选取
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Name) TableName() string { return "bot_names" }

// Exact 按 ID 获取昵称
func (n *Name) Exact(tx *gorm.DB, id uint64) error {
	return tx.Where("id = ?", id).First(n).Error
}

// Persona 人机风格: 一种决策策略和它的参数, 地图按模式配置出场的风格和权重
//
// 删除为软删除, 代码在已删除的风格中也唯一; 被地图引用时不能删除。
type Persona struct {
	ID          uint64         `gorm:"primaryKey" json:"id"`
	Code        string         `gorm:"size:32;not null;uniqueIndex:uniq_bot_personas_code" json:"code"` // 自然键, 地图按它引用, 创建后不能修改
	Name        string         `gorm:"size:32;not null" json:"name"`
	Description string         `gorm:"size:255;not null;default:''" json:"description"`
	Strategy    string         `gorm:"size:32;not null" json:"strategy"`                               // 决策策略的代码, 见 RegisterStrategy
	Params      map[string]any `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"params"` // 策略参数, 按策略的参数结构体校验并重新编码
	CreatedAt   time.Time      `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 自定义表名
func (Persona) TableName() string { return "bot_personas" }

// Exact 按 ID 获取未删除的风格
func (p *Persona) Exact(tx *gorm.DB, id uint64) error {
	return tx.Where("id = ?", id).First(p).Error
}

// Snapshot 风格的快照, 开局时存在人机的座位上
func (p *Persona) Snapshot() Snapshot {
	return Snapshot{Code: p.Code, Name: p.Name, Strategy: p.Strategy, Params: maps.Clone(p.Params)}
}

// Snapshot 开局时人机风格的快照, 之后修改或删除风格不影响进行中的对局; 真人为空
type Snapshot struct {
	Code     string         `json:"code,omitempty"`
	Name     string         `json:"name,omitempty"`
	Strategy string         `json:"strategy,omitempty"`
	Params   map[string]any `json:"params,omitempty"`
}

// Weight 地图上一种出场的风格和它的权重, 开局时每个人机按权重随机选一种
type Weight struct {
	Persona string `json:"persona" label:"风格" binding:"required,max=32"` // 风格代码
	Weight  int32  `json:"weight" label:"权重" binding:"min=1,max=100"`
}

// StrategyView 后台看到的决策策略
type StrategyView struct {
	Code        string            `json:"code"`
	Label       string            `json:"label"`
	Description string            `json:"description"`
	Fields      []condition.Field `json:"fields"`   // 参数的表单描述, 与条件类型相同
	Defaults    map[string]any    `json:"defaults"` // 新建风格时的默认参数
}
