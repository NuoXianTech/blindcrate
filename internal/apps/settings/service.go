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
	"fmt"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
	Version       int64                  `json:"version"`        // 保存时原样提交, 未保存过为 0
	UpdatedBy     uint64                 `json:"updated_by"`
	UpdatedAt     *time.Time             `json:"updated_at"` // 未保存过为空
}

// RevisionView 一个版本的快照
type RevisionView struct {
	Revision
	Value         map[string]interface{} `json:"value"`          // 密钥字段固定为空字符串
	SecretsSet    map[string]bool        `json:"secrets_set"`    // 密钥字段是否已设置
	SecretsBroken []string               `json:"secrets_broken"` // 无法解密的密钥字段, 有这类字段时不能回滚到此版本
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

// HasGroup 设置组是否存在
func HasGroup(key string) bool {
	return findGroup(key) != nil
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
		value, broken := g.decode(ctx, rows[g.key])
		views = append(views, g.view(rows[g.key], value, broken))
	}
	return views, nil
}

// QueryGroup 获取一个设置组, 不存在时返回 GroupNotFound
func QueryGroup(ctx context.Context, key string) (*GroupView, error) {
	g := findGroup(key)
	if g == nil {
		return nil, errors.New(GroupNotFound)
	}
	var row Setting
	if err := db.DB(ctx).Where("key = ?", key).Limit(1).Find(&row).Error; err != nil {
		return nil, err
	}
	value, broken := g.decode(ctx, &row)
	view := g.view(&row, value, broken)
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

// QueryRevision 获取一个版本的快照, 不存在时返回 GroupNotFound 或 RevisionNotFound
func QueryRevision(ctx context.Context, key string, version int64) (*RevisionView, error) {
	g := findGroup(key)
	if g == nil {
		return nil, errors.New(GroupNotFound)
	}
	revision, err := findRevision(db.DB(ctx), key, version)
	if err != nil {
		return nil, err
	}
	value, broken := g.decodeValue(ctx, revision.Value)
	return &RevisionView{
		Revision:      *revision,
		Value:         g.values(value, false),
		SecretsSet:    g.secretsSet(value),
		SecretsBroken: broken,
	}, nil
}

// findRevision 按版本号查找版本, 不存在时返回 RevisionNotFound
func findRevision(tx *gorm.DB, key string, version int64) (*Revision, error) {
	var revision Revision
	if err := tx.Where("key = ? AND version = ?", key, version).Limit(1).Find(&revision).Error; err != nil {
		return nil, err
	}
	if revision.ID == 0 {
		return nil, errors.New(RevisionNotFound)
	}
	return &revision, nil
}

// Update 保存一个设置组, 返回保存后的设置组和本次修改的字段
//
// 合并规则见 group.merge; version 为修改前读到的版本, 已被别人修改时返回 VersionConflict。
// 返回的错误都可以直接展示: 数据库错误记录日志后返回 SaveFailed, 其余为 GroupNotFound、VersionConflict 或校验失败。
func Update(ctx context.Context, operatorID uint64, key string, version int64, raw json.RawMessage, clearSecrets []string) (*GroupView, map[string]interface{}, error) {
	g := findGroup(key)
	if g == nil {
		return nil, nil, errors.New(GroupNotFound)
	}
	return g.save(ctx, operatorID, version, RevisionUpdate, 0, func(_ *gorm.DB, cur any, broken []string) (any, error) {
		return g.merge(cur, broken, raw, clearSecrets)
	})
}

// Rollback 把设置组恢复为 target 版本的内容, 回滚也会生成一个新版本
//
// 快照中的字段按当前的规则重新校验; 快照中有无法解密的密钥字段时返回 RevisionBroken, 需要手动修改。
func Rollback(ctx context.Context, operatorID uint64, key string, version, target int64) (*GroupView, map[string]interface{}, error) {
	g := findGroup(key)
	if g == nil {
		return nil, nil, errors.New(GroupNotFound)
	}
	return g.save(ctx, operatorID, version, RevisionRollback, target, func(tx *gorm.DB, _ any, _ []string) (any, error) {
		revision, err := findRevision(tx, key, target)
		if err != nil {
			return nil, err
		}
		next, broken := g.decodeValue(ctx, revision.Value)
		if len(broken) > 0 {
			return nil, fmt.Errorf(RevisionBroken, target, g.field(broken[0]).Label)
		}
		if err = g.validate(next); err != nil {
			return nil, err
		}
		return next, nil
	})
}

// Reset 把设置组恢复为代码中的默认值, 密钥字段会被清空, 同样生成一个新版本
func Reset(ctx context.Context, operatorID uint64, key string, version int64) (*GroupView, map[string]interface{}, error) {
	g := findGroup(key)
	if g == nil {
		return nil, nil, errors.New(GroupNotFound)
	}
	return g.save(ctx, operatorID, version, RevisionReset, 0, func(_ *gorm.DB, _ any, _ []string) (any, error) {
		return g.defaults(), nil
	})
}

// save 锁定当前值, 用 build 生成新值后保存, 并写入一个版本
//
// 没有任何字段变化时不写库, 也不生成版本; 启用版本记录前保存过的设置组, 第一次保存时先把原内容补记为版本 0。
// build 返回的错误原样返回, 数据库错误记录日志后返回 SaveFailed。
func (g *group) save(ctx context.Context, operatorID uint64, version int64, action string, fromVersion int64,
	build func(tx *gorm.DB, cur any, broken []string) (any, error)) (*GroupView, map[string]interface{}, error) {
	var (
		view    GroupView
		changes map[string]interface{}
		failure error // 可以直接展示的错误
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var row Setting
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("key = ?", g.key).Limit(1).Find(&row).Error; err != nil {
			return err
		}
		if row.Version != version {
			failure = errors.New(VersionConflict)
			return failure
		}
		cur, broken := g.decode(ctx, &row)
		next, err := build(tx, cur, broken)
		if err != nil {
			failure = err
			return err
		}
		changes = g.diff(cur, broken, next)
		if len(changes) == 0 {
			view = g.view(&row, cur, broken)
			return nil
		}
		stored, err := g.encode(next)
		if err != nil {
			if err.Error() == SecretKeyMissing {
				failure = err
			}
			return err
		}

		now := time.Now()
		if row.Key == "" {
			row = Setting{Key: g.key, Value: stored, Version: 1, UpdatedBy: operatorID, CreatedAt: now, UpdatedAt: now}
			if err = tx.Create(&row).Error; err != nil {
				// 别人同时第一次保存
				if db.IsUniqueViolation(err) {
					failure = errors.New(VersionConflict)
				}
				return err
			}
		} else {
			if row.Version == 0 {
				baseline := Revision{
					Key:       g.key,
					Value:     row.Value,
					Changes:   map[string]interface{}{},
					Action:    RevisionBaseline,
					UpdatedBy: row.UpdatedBy,
					CreatedAt: row.UpdatedAt,
				}
				if err = tx.Create(&baseline).Error; err != nil {
					return err
				}
			}
			row.Value, row.Version, row.UpdatedBy, row.UpdatedAt = stored, row.Version+1, operatorID, now
			if err = tx.Model(&Setting{}).Where("key = ?", g.key).Updates(map[string]interface{}{
				"value":      row.Value,
				"version":    row.Version,
				"updated_by": row.UpdatedBy,
				"updated_at": row.UpdatedAt,
			}).Error; err != nil {
				return err
			}
		}

		revision := Revision{
			Key:         g.key,
			Version:     row.Version,
			Value:       stored,
			Changes:     changes,
			Action:      action,
			FromVersion: fromVersion,
			UpdatedBy:   operatorID,
			CreatedAt:   now,
		}
		if err = tx.Create(&revision).Error; err != nil {
			return err
		}
		view = g.view(&row, next, nil)
		return nil
	})
	if failure != nil {
		return nil, nil, failure
	}
	if err != nil {
		logger.ErrorF(ctx, "[Settings] save %s failed: %v", g.key, err)
		return nil, nil, errors.New(SaveFailed)
	}
	return &view, changes, nil
}
