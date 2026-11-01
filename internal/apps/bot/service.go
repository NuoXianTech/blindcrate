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
	"errors"
	"fmt"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	audit.RegisterAction(ActionNameCreate, "新建人机昵称")
	audit.RegisterAction(ActionNameUpdate, "修改人机昵称")
	audit.RegisterAction(ActionNameDelete, "删除人机昵称")
	audit.RegisterAction(ActionPersonaCreate, "新建人机风格")
	audit.RegisterAction(ActionPersonaUpdate, "修改人机风格")
	audit.RegisterAction(ActionPersonaDelete, "删除人机风格")
}

// codePattern 风格代码, 小写字母、数字和下划线
var codePattern = regexp.MustCompile(`^[a-z0-9_]{1,32}$`)

// InvalidError 人机配置不合法, Error 返回的提示可以直接展示给管理员
type InvalidError struct {
	Reason string
}

func (e *InvalidError) Error() string { return e.Reason }

// ValidateName 校验人机昵称, 需要调用方先去掉首尾空白
func ValidateName(name string) error {
	if n := utf8.RuneCountInString(name); n < 1 || n > maxNameLength {
		return &InvalidError{Reason: NameInvalid}
	}
	return nil
}

// Validate 保存风格前校验配置, 校验后参数按策略的参数结构体重新编码, 缺少的参数取默认值
//
// 配置不合法时返回 *InvalidError; 其他错误为查库失败等内部错误。名称等文本字段需要调用方先去掉首尾空白。
func Validate(tx *gorm.DB, p *Persona) error {
	if !codePattern.MatchString(p.Code) {
		return &InvalidError{Reason: CodeInvalid}
	}
	if n := utf8.RuneCountInString(p.Name); n < 1 || n > maxPersonaNameLength {
		return &InvalidError{Reason: PersonaNameInvalid}
	}
	if utf8.RuneCountInString(p.Description) > maxDescriptionLength {
		return &InvalidError{Reason: DescriptionInvalid}
	}
	params, err := validateParams(tx, p.Strategy, p.Params)
	if err != nil {
		return err
	}
	p.Params = params
	return nil
}

// CheckWeights 地图保存前校验出场的风格: 不能重复, 需为策略仍然存在的风格; 返回非空的 reason 表示不合法
//
// 风格加共享锁, 防止保存地图的同时风格被删除; 删除风格时先加排他锁再检查是否被地图引用。
func CheckWeights(tx *gorm.DB, weights []Weight) (string, error) {
	codes := make([]string, 0, len(weights))
	for _, w := range weights {
		if slices.Contains(codes, w.Persona) {
			return fmt.Sprintf(PersonasDuplicated, w.Persona), nil
		}
		codes = append(codes, w.Persona)
	}
	if len(codes) == 0 {
		return "", nil
	}
	var personas []Persona
	if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Select("code", "strategy").Where("code IN ?", codes).Find(&personas).Error; err != nil {
		return "", err
	}
	var missing, broken []string
	for _, code := range codes {
		i := slices.IndexFunc(personas, func(p Persona) bool { return p.Code == code })
		switch {
		case i < 0:
			missing = append(missing, code)
		case strategies[personas[i].Strategy] == nil:
			broken = append(broken, code)
		}
	}
	switch {
	case len(missing) > 0:
		return fmt.Sprintf(PersonasMissing, strings.Join(missing, ", ")), nil
	case len(broken) > 0:
		return fmt.Sprintf(PersonasBroken, strings.Join(broken, ", ")), nil
	}
	return "", nil
}

// Assign 为一局的 n 个人机按权重随机选择风格, 返回风格的快照
//
// 地图保存时已校验风格, 被地图引用的风格不能删除, 这里风格不存在或策略没有注册说明数据不一致, 返回错误。
func Assign(tx *gorm.DB, weights []Weight, n int) ([]Snapshot, error) {
	if n <= 0 {
		return nil, nil
	}
	if len(weights) == 0 {
		return nil, errors.New("bot: no persona configured")
	}
	codes := make([]string, len(weights))
	picks := make([]int64, len(weights))
	for i, w := range weights {
		if w.Weight <= 0 {
			return nil, fmt.Errorf("bot: invalid weight of persona %s", w.Persona)
		}
		codes[i], picks[i] = w.Persona, int64(w.Weight)
	}
	var personas []Persona
	if err := tx.Where("code IN ?", codes).Find(&personas).Error; err != nil {
		return nil, err
	}
	byCode := make(map[string]*Persona, len(personas))
	for i := range personas {
		if strategies[personas[i].Strategy] == nil {
			return nil, fmt.Errorf("bot: strategy %s of persona %s not registered", personas[i].Strategy, personas[i].Code)
		}
		byCode[personas[i].Code] = &personas[i]
	}
	for _, code := range codes {
		if byCode[code] == nil {
			return nil, fmt.Errorf("bot: persona %s not found", code)
		}
	}
	snapshots := make([]Snapshot, n)
	for i := range snapshots {
		snapshots[i] = byCode[codes[loot.PickIndex(picks)]].Snapshot()
	}
	return snapshots, nil
}

// PickNames 为一局的 n 个人机随机选取启用的昵称, 同一局不重复; 不够时用 "玩家" 加四位数字补齐
func PickNames(tx *gorm.DB, n int) ([]string, error) {
	if n <= 0 {
		return nil, nil
	}
	var names []string
	if err := tx.Model(&Name{}).Where("is_enabled").Order("RANDOM()").Limit(n).Pluck("name", &names).Error; err != nil {
		return nil, err
	}
	for len(names) < n {
		name := fmt.Sprintf("%s%04d", fallbackNamePrefix, rand.IntN(10000))
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return names, nil
}
