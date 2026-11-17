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

package settings

import (
	"context"
	"encoding/json"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/fault"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
)

func init() {
	audit.RegisterAction(ActionSettingsUpdate, "修改系统设置")
	audit.RegisterAction(ActionSettingsReset, "恢复系统设置默认值")
	audit.RegisterAction(ActionMailTest, "发送测试邮件")
}

// GroupView 管理后台看到的设置组, 包含表单描述和当前值
type GroupView struct {
	Key           string                 `json:"key"`
	Label         string                 `json:"label"`
	Description   string                 `json:"description"`
	Category      string                 `json:"category"`
	Fields        []Field                `json:"fields"`
	Value         map[string]interface{} `json:"value"`          // 密钥字段固定为空字符串
	SecretsSet    map[string]bool        `json:"secrets_set"`    // 密钥字段是否已设置
	SecretsBroken []string               `json:"secrets_broken"` // 无法解密的密钥字段, 需要重新填写或清空才能保存
	UpdatedBy     uint64                 `json:"updated_by"`
	UpdatedAt     *time.Time             `json:"updated_at"` // 未保存过为空
}

// Get 读取当前值, 未保存过时返回默认值
func (h *Handle[T]) Get(ctx context.Context) (*T, error) {
	var row Setting
	if err := db.DB(ctx).Where("key = ?", h.g.key).Limit(1).Find(&row).Error; err != nil {
		return nil, err
	}
	value, _ := h.g.decode(ctx, &row)
	return value.(*T), nil
}

// Load 在调用方的事务中读取当前值, 未保存过时返回默认值; 用于需要看到同一事务中刚导入的设置的场景
func (h *Handle[T]) Load(tx *gorm.DB) (*T, error) {
	var row Setting
	if err := tx.Where("key = ?", h.g.key).Limit(1).Find(&row).Error; err != nil {
		return nil, err
	}
	value, _ := h.g.decode(tx.Statement.Context, &row)
	return value.(*T), nil
}

// UpdatedAt 上次保存的时间, 未保存过时为空; 用于判断按这组设置算出的结果是否过期
func (h *Handle[T]) UpdatedAt(ctx context.Context) (*time.Time, error) {
	var row Setting
	if err := db.DB(ctx).Select("key", "updated_at").Where("key = ?", h.g.key).Limit(1).Find(&row).Error; err != nil {
		return nil, err
	}
	if row.Key == "" {
		return nil, nil
	}
	return &row.UpdatedAt, nil
}

// QueryGroups 获取设置组, category 为空时返回全部
func QueryGroups(ctx context.Context, category string) ([]GroupView, error) {
	rows, err := queryRows(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]GroupView, 0, len(groups))
	for _, g := range groups {
		if category != "" && g.category != category {
			continue
		}
		fields, err := g.formFields(db.DB(ctx))
		if err != nil {
			return nil, err
		}
		value, broken := g.decode(ctx, rows[g.key])
		views = append(views, g.view(fields, rows[g.key], value, broken))
	}
	return views, nil
}

// QueryGroup 获取一个设置组, 不存在时返回 GroupNotFound
func QueryGroup(ctx context.Context, key string) (*GroupView, error) {
	g := findGroup(key)
	if g == nil {
		return nil, fault.NotFound(GroupNotFound)
	}
	var row Setting
	if err := db.DB(ctx).Where("key = ?", key).Limit(1).Find(&row).Error; err != nil {
		return nil, err
	}
	fields, err := g.formFields(db.DB(ctx))
	if err != nil {
		return nil, err
	}
	value, broken := g.decode(ctx, &row)
	view := g.view(fields, &row, value, broken)
	return &view, nil
}

// QueryPublic 获取全部公开字段, 按设置组分组
func QueryPublic(ctx context.Context) (map[string]map[string]interface{}, error) {
	rows, err := queryRows(ctx)
	if err != nil {
		return nil, err
	}
	result := map[string]map[string]interface{}{}
	for _, g := range groups {
		value, _ := g.decode(ctx, rows[g.key])
		if values := g.values(value, true); len(values) > 0 {
			result[g.key] = values
		}
	}
	return result, nil
}

// queryRows 读取全部设置组, 未保存过的设置组为空记录
func queryRows(ctx context.Context) (map[string]*Setting, error) {
	var rows []Setting
	if err := db.DB(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := map[string]*Setting{}
	for _, g := range groups {
		result[g.key] = &Setting{}
	}
	for i := range rows {
		result[rows[i].Key] = &rows[i]
	}
	return result, nil
}

// Update 保存一个设置组, 返回保存后的设置组和本次修改的字段
//
// 合并规则见 group.merge; 多人同时保存时依次合并, 改了同一个字段的以后保存的为准。
// 返回的错误写明了类别的可以直接展示(GroupNotFound 或校验失败), 其余为数据库错误, 见 fault。
func Update(ctx context.Context, operatorID uint64, key string, raw json.RawMessage, clearSecrets []string) (*GroupView, map[string]interface{}, error) {
	g := findGroup(key)
	if g == nil {
		return nil, nil, fault.NotFound(GroupNotFound)
	}
	return g.save(ctx, operatorID, func(tx *gorm.DB, cur any, broken []string) (any, error) {
		return g.merge(tx, cur, broken, raw, clearSecrets)
	})
}

// Reset 把设置组恢复为代码中的默认值, 密钥字段会被清空
func Reset(ctx context.Context, operatorID uint64, key string) (*GroupView, map[string]interface{}, error) {
	g := findGroup(key)
	if g == nil {
		return nil, nil, fault.NotFound(GroupNotFound)
	}
	return g.save(ctx, operatorID, func(_ *gorm.DB, _ any, _ []string) (any, error) {
		return g.defaults(), nil
	})
}

// save 在新事务中用 build 生成新值后保存
//
// build 返回的错误和数据库错误都原样返回; 其余规则见 saveTx。
func (g *group) save(ctx context.Context, operatorID uint64,
	build func(tx *gorm.DB, cur any, broken []string) (any, error)) (*GroupView, map[string]interface{}, error) {
	var (
		view    *GroupView
		changes map[string]interface{}
		failure error // 可以直接展示的错误
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		view, changes, failure, err = g.saveTx(ctx, tx, operatorID, build)
		if failure != nil {
			return failure
		}
		return err
	})
	if failure != nil {
		return nil, nil, failure
	}
	if err != nil {
		return nil, nil, err
	}
	return view, changes, nil
}

// saveTx 在调用方的事务中用 build 生成新值后保存
//
// 同一个设置组的保存在咨询锁上排队, build 拿到的总是最新的已保存值; 没有任何字段变化时不写库。
// failure 为可以直接展示的错误(build 返回的错误等), err 为数据库错误。
func (g *group) saveTx(ctx context.Context, tx *gorm.DB, operatorID uint64,
	build func(tx *gorm.DB, cur any, broken []string) (any, error)) (view *GroupView, changes map[string]interface{}, failure, err error) {
	if err = db.Lock(tx, "settings:"+g.key); err != nil {
		return nil, nil, nil, err
	}
	var row Setting
	if err = tx.Where("key = ?", g.key).Limit(1).Find(&row).Error; err != nil {
		return nil, nil, nil, err
	}
	cur, broken := g.decode(ctx, &row)
	next, err := build(tx, cur, broken)
	if err != nil {
		return nil, nil, err, nil
	}
	fields, err := g.formFields(tx)
	if err != nil {
		return nil, nil, nil, err
	}
	changes = g.diff(cur, broken, next)
	if len(changes) == 0 {
		unchanged := g.view(fields, &row, cur, broken)
		return &unchanged, changes, nil, nil
	}
	stored, err := g.encode(next)
	if err != nil {
		if fault.KindOf(err) != fault.KindInternal {
			return nil, nil, err, nil
		}
		return nil, nil, nil, err
	}

	// 数据库的时间精确到微秒, 返回的保存时间与之后读到的一致, 前端按它判断有没有保存
	now := time.Now().Truncate(time.Microsecond)
	if row.Key == "" {
		row = Setting{Key: g.key, Value: stored, UpdatedBy: operatorID, CreatedAt: now, UpdatedAt: now}
		if err = tx.Create(&row).Error; err != nil {
			return nil, nil, nil, err
		}
	} else {
		row.Value, row.UpdatedBy, row.UpdatedAt = stored, operatorID, now
		if err = tx.Model(&Setting{}).Where("key = ?", g.key).Updates(map[string]interface{}{
			"value":      row.Value,
			"updated_by": row.UpdatedBy,
			"updated_at": row.UpdatedAt,
		}).Error; err != nil {
			return nil, nil, nil, err
		}
	}
	saved := g.view(fields, &row, next, nil)
	return &saved, changes, nil, nil
}

// ExportValues 可以导出的设置组(游戏分类)的当前值, 按设置组的 key, 不含密钥字段
func ExportValues(ctx context.Context) (map[string]map[string]interface{}, error) {
	rows, err := queryRows(ctx)
	if err != nil {
		return nil, err
	}
	result := map[string]map[string]interface{}{}
	for _, g := range groups {
		if g.category != CategoryGame {
			continue
		}
		value, _ := g.decode(ctx, rows[g.key])
		values := g.values(value, false)
		for _, f := range g.fields {
			if f.Secret {
				delete(values, f.Key)
			}
		}
		result[g.key] = values
	}
	return result, nil
}

// ImportLabel 可以导入的设置组的名称, 只有游戏分类的设置组可以导入
func ImportLabel(key string) (string, bool) {
	g := findGroup(key)
	if g == nil || g.category != CategoryGame {
		return "", false
	}
	return g.label, true
}

// Import 在调用方的事务中导入一个设置组, 返回本次修改的字段, 没有任何字段变化时不写库
//
// 提交的字段合并到当前值上并按保存的规则校验, 没有提交的字段和密钥字段保持当前值。
// 返回的错误中 failure 可以直接展示, err 为数据库错误。
func Import(ctx context.Context, tx *gorm.DB, operatorID uint64, key string, raw json.RawMessage) (changes map[string]interface{}, failure, err error) {
	if _, ok := ImportLabel(key); !ok {
		return nil, fault.Invalid(GroupNotImportable), nil
	}
	g := findGroup(key)
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fault.Invalid(ValueInvalid), nil
	}
	for _, f := range g.fields {
		if f.Secret {
			delete(values, f.Key)
		}
	}
	raw, err = json.Marshal(values)
	if err != nil {
		return nil, nil, err
	}
	build := func(tx *gorm.DB, cur any, broken []string) (any, error) {
		return g.merge(tx, cur, broken, raw, nil)
	}
	_, changes, failure, err = g.saveTx(ctx, tx, operatorID, build)
	return changes, failure, err
}
