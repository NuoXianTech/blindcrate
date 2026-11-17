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

package pack

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"gorm.io/gorm"
)

// 设置是这里唯一自己注册的配置种类: 它不是一张表, 旧版本的包升级时还要改它的内容, 见 packUpgrades; 其他配置种类由各模块在自己的 init 中注册
func init() {
	audit.RegisterAction(ActionPackExport, "导出配置包")
	audit.RegisterAction(ActionPackImport, "导入配置包")
	RegisterSection(Section{
		Code: PackSettings, Label: "设置", Hint: "游戏分类的设置组, 不含密钥字段", Order: OrderSettings,
		Export: func(ctx context.Context, _ *gorm.DB, _ func(string) string) (any, int, error) {
			values, err := settings.ExportValues(ctx)
			if err != nil {
				return nil, 0, err
			}
			return values, len(values), nil
		},
		Decode: func(data []byte) (Content, error) {
			var c settingsContent
			if err := decodePackJSON(data, &c); err != nil {
				return nil, err
			}
			return c, nil
		},
	})
}

// settingsContent 设置组的值, 按设置组的 key
type settingsContent map[string]json.RawMessage

func (c settingsContent) Len() int { return len(c) }

// Entries 按 key 排序
func (c settingsContent) Entries(r *Run) []*Entry {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	entries := make([]*Entry, len(keys))
	for i, key := range keys {
		entries[i] = settingsEntry(r, key, c[key])
	}
	return entries
}

func (settingsContent) Renamed(string, string, string) {}

// settingsEntry 设置组没有新增, 先在保存点中试着导入一次, 按修改的字段判断是否相同
func settingsEntry(r *Run, key string, raw json.RawMessage) *Entry {
	label, _ := settings.ImportLabel(key)
	importGroup := func(tx *gorm.DB) (map[string]interface{}, error) {
		changes, failure, err := settings.Import(r.ctx, tx, r.operatorID, key, raw)
		if failure != nil {
			return nil, failure
		}
		return changes, err
	}
	return &Entry{
		Key:  key,
		Name: label,
		Compare: func(tx *gorm.DB) (string, string, error) {
			if err := tx.SavePoint(packSavePoint).Error; err != nil {
				return "", "", err
			}
			changes, err := importGroup(tx)
			if err := tx.RollbackTo(packSavePoint).Error; err != nil {
				return "", "", err
			}
			if err := releasePackSavePoint(tx); err != nil {
				return "", "", err
			}
			switch {
			case err != nil && packFailure(err):
				return PackStatusConflict, err.Error(), nil
			case err != nil:
				return "", "", err
			case len(changes) == 0:
				return PackStatusSame, "", nil
			}
			return PackStatusChanged, "", nil
		},
		Apply: func(tx *gorm.DB, _ bool) error {
			_, err := importGroup(tx)
			return err
		},
	}
}
