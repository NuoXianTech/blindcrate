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
	"errors"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"gorm.io/gorm/clause"
)

// GroupView 管理后台看到的设置组, 包含表单描述和当前值
type GroupView struct {
	Key           string                 `json:"key"`
	Label         string                 `json:"label"`
	Description   string                 `json:"description"`
	Fields        []Field                `json:"fields"`
	Value         map[string]interface{} `json:"value"`          // 密钥字段固定为空字符串
	SecretsSet    map[string]bool        `json:"secrets_set"`    // 密钥字段是否已设置
	SecretsBroken []string               `json:"secrets_broken"` // 无法解密的密钥字段, 需要重新填写或清空才能保存
	UpdatedBy     uint64                 `json:"updated_by"`
	UpdatedAt     *time.Time             `json:"updated_at"` // 未保存过为空
}

// Site 当前站点设置
func Site(ctx context.Context) (*SiteSettings, error) {
	return get[SiteSettings](ctx, KeySite)
}

// Auth 当前注册与登录设置
func Auth(ctx context.Context) (*AuthSettings, error) {
	return get[AuthSettings](ctx, KeyAuth)
}

// Mail 当前邮件设置
func Mail(ctx context.Context) (*MailSettings, error) {
	return get[MailSettings](ctx, KeyMail)
}

// get 读取一个设置组, 未保存过时返回默认值
func get[T any](ctx context.Context, key string) (*T, error) {
	var row Setting
	if err := db.DB(ctx).Where("key = ?", key).Limit(1).Find(&row).Error; err != nil {
		return nil, err
	}
	value, _ := findGroup(key).decode(ctx, &row)
	return value.(*T), nil
}

// QueryGroups 获取全部设置组
func QueryGroups(ctx context.Context) ([]GroupView, error) {
	rows, err := queryRows(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]GroupView, 0, len(groups))
	for _, g := range groups {
		value, broken := g.decode(ctx, rows[g.key])
		views = append(views, g.view(rows[g.key], value, broken))
	}
	return views, nil
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
// 合并规则见 group.merge, 没有任何字段变化时不写库; 多人同时修改时以后保存的为准。
// 返回的错误都可以直接展示: 数据库错误记录日志后返回 SaveFailed, 其余为 GroupNotFound 或校验失败。
func Update(ctx context.Context, operatorID uint64, key string, raw json.RawMessage, clearSecrets []string) (*GroupView, map[string]interface{}, error) {
	g := findGroup(key)
	if g == nil {
		return nil, nil, errors.New(GroupNotFound)
	}

	var row Setting
	if err := db.DB(ctx).Where("key = ?", key).Limit(1).Find(&row).Error; err != nil {
		logger.ErrorF(ctx, "[Settings] query %s failed: %v", key, err)
		return nil, nil, errors.New(SaveFailed)
	}
	cur, broken := g.decode(ctx, &row)
	next, err := g.merge(cur, broken, raw, clearSecrets)
	if err != nil {
		return nil, nil, err
	}
	changes := g.diff(cur, broken, next)
	if len(changes) == 0 {
		view := g.view(&row, cur, broken)
		return &view, changes, nil
	}

	stored, err := g.encode(next)
	if err != nil {
		if err.Error() == SecretKeyMissing {
			return nil, nil, err
		}
		logger.ErrorF(ctx, "[Settings] encode %s failed: %v", key, err)
		return nil, nil, errors.New(SaveFailed)
	}
	row.Key, row.Value, row.UpdatedBy, row.UpdatedAt = key, stored, operatorID, time.Now()
	if err = db.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_by", "updated_at"}),
	}).Create(&row).Error; err != nil {
		logger.ErrorF(ctx, "[Settings] save %s failed: %v", key, err)
		return nil, nil, errors.New(SaveFailed)
	}
	view := g.view(&row, next, nil)
	return &view, changes, nil
}
