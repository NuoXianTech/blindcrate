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

package redeemadmin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/redeem"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 批次列表的状态筛选
const (
	statusActive   = "active"   // 已启用且没有到期
	statusExpired  = "expired"  // 已启用但已到期
	statusDisabled = "disabled" // 已停用
)

// insertBatchSize 一条 INSERT 最多写入的兑换码数
const insertBatchSize = 500

// BatchItem 后台展示的批次, 带上生成人的用户名和已兑换的次数
type BatchItem struct {
	redeem.Batch
	CreatedUsername string `json:"created_username"` // 用户已不存在时为空
	Redeemed        int64  `json:"redeemed"`         // 批次中全部兑换码已兑换的次数
}

// QueryBatches 获取批次列表, 从新到旧
func QueryBatches(ctx context.Context, req *ListBatchesRequest) (int64, []BatchItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&redeem.Batch{})
	if req.Keyword != "" {
		// 关键字也可以是完整的兑换码, 规范化后精确匹配, 用于查一个兑换码属于哪个批次
		query = query.Where(
			"redeem_batches.name ILIKE ? OR EXISTS (SELECT 1 FROM redeem_codes WHERE redeem_codes.batch_id = redeem_batches.id AND redeem_codes.code = ?)",
			"%"+db.EscapeLike(req.Keyword)+"%", redeem.Normalize(req.Keyword),
		)
	}
	now := time.Now()
	switch req.Status {
	case statusActive:
		query = query.Where("redeem_batches.is_enabled AND (redeem_batches.expires_at IS NULL OR redeem_batches.expires_at > ?)", now)
	case statusExpired:
		query = query.Where("redeem_batches.is_enabled AND redeem_batches.expires_at <= ?", now)
	case statusDisabled:
		query = query.Where("NOT redeem_batches.is_enabled")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	items := []BatchItem{}
	if err := query.
		Select("redeem_batches.*, COALESCE(creators.username, '') AS created_username, " +
			"(SELECT COUNT(*) FROM redeem_records WHERE redeem_records.batch_id = redeem_batches.id) AS redeemed").
		Joins("LEFT JOIN users AS creators ON creators.id = redeem_batches.created_by").
		Order("redeem_batches.id DESC").Offset(offset).Limit(req.Size).
		Find(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}

// doCreateBatch 校验并生成一个批次的兑换码: 指定了兑换码时只生成这一个, 否则随机生成 Count 个
//
// 幂等键已生成过时返回已有的批次, replayed 为 true。
func doCreateBatch(ctx context.Context, operatorID uint64, req *CreateBatchRequest) (_ *redeem.Batch, replayed bool, _ error) {
	target := &redeem.Batch{IdempotencyKey: req.IdempotencyKey, CreatedBy: operatorID}
	applyBatch(target, &req.BatchRequest)
	// 修改时不检查, 把到期时间改到过去可以让批次立即到期
	if target.ExpiresAt != nil && !target.ExpiresAt.After(time.Now()) {
		return nil, false, &redeem.InvalidError{Reason: redeem.ExpiresInvalid}
	}
	var codes []string
	if strings.TrimSpace(req.Code) != "" {
		code := redeem.Normalize(req.Code)
		if !redeem.ValidCode(code) {
			return nil, false, &redeem.InvalidError{Reason: redeem.CodeInvalid}
		}
		codes = []string{code}
	} else {
		if req.Count < 1 || req.Count > redeem.MaxBatchCodes {
			return nil, false, &redeem.InvalidError{Reason: redeem.CountInvalid}
		}
		codes = redeem.Generate(req.Count)
	}
	target.CodeCount = int32(len(codes))
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := redeem.Validate(tx, target); err != nil {
			return err
		}
		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "idempotency_key"}}, DoNothing: true}).Create(target)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			replayed = true
			*target = redeem.Batch{}
			// 已有的批次之后被删除了也返回它, 不再生成一次
			return tx.Unscoped().Where("idempotency_key = ?", req.IdempotencyKey).First(target).Error
		}
		rows := make([]redeem.Code, len(codes))
		for i, code := range codes {
			rows[i] = redeem.Code{BatchID: target.ID, Code: code}
		}
		// 与已有的兑换码重复时跳过而不是让唯一索引报错: 报错的 SQL 会连同这些仍可兑换的兑换码一起记进错误日志
		inserted := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoNothing: true}).CreateInBatches(&rows, insertBatchSize)
		if inserted.Error != nil {
			return inserted.Error
		}
		// 包括已删除的批次中的兑换码; 随机生成时几乎不会遇到, 重新提交即可
		if inserted.RowsAffected != int64(len(rows)) {
			return errors.New(redeem.CodeTaken)
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return target, replayed, nil
}

// doUpdateBatch 校验并修改批次, 返回修改后的批次和本次修改的字段, 没有任何字段变化时不写库
func doUpdateBatch(ctx context.Context, batchID uint64, req *BatchRequest) (*redeem.Batch, map[string]interface{}, error) {
	var (
		target  redeem.Batch
		changes map[string]interface{}
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockBatch(tx, &target, batchID); err != nil {
			return err
		}
		old := target
		applyBatch(&target, req)
		if err := redeem.Validate(tx, &target); err != nil {
			return err
		}
		changes = audit.Changes(batchFields(&old), batchFields(&target))
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

// doDeleteBatch 软删除批次, 返回被删除的批次; 批次中的兑换码保留, 不能再兑换
func doDeleteBatch(ctx context.Context, batchID uint64) (*redeem.Batch, error) {
	var target redeem.Batch
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockBatch(tx, &target, batchID); err != nil {
			return err
		}
		return tx.Delete(&target).Error
	})
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// lockBatch 锁定批次, 防止并发修改、删除和兑换互相覆盖
func lockBatch(tx *gorm.DB, target *redeem.Batch, batchID uint64) error {
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), batchID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(redeem.BatchNotFound)
		}
		return err
	}
	return nil
}

// applyBatch 写入批次可修改的字段, 由 redeem.Validate 校验; 到期时间精确到分钟
func applyBatch(target *redeem.Batch, req *BatchRequest) {
	target.Name = strings.TrimSpace(req.Name)
	target.Reward = req.Reward
	target.Requirements = req.Requirements
	target.MaxUses = req.MaxUses
	target.UserLimit = req.UserLimit
	target.ExpiresAt = truncateMinute(req.ExpiresAt)
	target.IsEnabled = req.IsEnabled
}

// truncateMinute 去掉时间中的秒, 为空时返回空
func truncateMinute(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	truncated := t.Truncate(time.Minute)
	return &truncated
}

// batchFields 批次可修改的字段, 用于操作日志; 到期时间统一为 UTC 的 RFC3339 格式, 便于比较
func batchFields(target *redeem.Batch) map[string]interface{} {
	var expiresAt interface{}
	if target.ExpiresAt != nil {
		expiresAt = target.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return map[string]interface{}{
		"name":         target.Name,
		"reward":       target.Reward,
		"requirements": target.Requirements,
		"max_uses":     target.MaxUses,
		"user_limit":   target.UserLimit,
		"expires_at":   expiresAt,
		"is_enabled":   target.IsEnabled,
	}
}

// QueryCodes 获取批次的全部兑换码, 按生成顺序排列; 批次不存在或已删除时返回 BatchNotFound
func QueryCodes(ctx context.Context, batchID uint64) ([]redeem.Code, error) {
	tx := db.DB(ctx)
	var batch redeem.Batch
	if err := batch.Exact(tx, batchID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(redeem.BatchNotFound)
		}
		return nil, err
	}
	codes := []redeem.Code{}
	if err := tx.Where("batch_id = ?", batchID).Order("id").Find(&codes).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

// RecordItem 后台展示的兑换记录, 带上用户名、批次名称和兑换码
type RecordItem struct {
	redeem.Record
	Username  string `json:"username"`   // 用户已不存在时为空
	BatchName string `json:"batch_name"` // 批次已删除时仍然返回
	Code      string `json:"code"`
}

// QueryRecords 获取兑换记录, 从新到旧, 包括已删除的批次的记录
func QueryRecords(ctx context.Context, req *ListRecordsRequest) (int64, []RecordItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&redeem.Record{})
	if req.BatchID != nil {
		query = query.Where("redeem_records.batch_id = ?", *req.BatchID)
	}
	if req.UserID != nil {
		query = query.Where("redeem_records.user_id = ?", *req.UserID)
	}
	if req.Code != "" {
		query = query.Where("redeem_records.code_id IN (SELECT id FROM redeem_codes WHERE code = ?)", redeem.Normalize(req.Code))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	records := []RecordItem{}
	if err := query.
		Select("redeem_records.*, COALESCE(users.username, '') AS username, COALESCE(redeem_batches.name, '') AS batch_name, COALESCE(redeem_codes.code, '') AS code").
		Joins("LEFT JOIN users ON users.id = redeem_records.user_id").
		Joins("LEFT JOIN redeem_batches ON redeem_batches.id = redeem_records.batch_id").
		Joins("LEFT JOIN redeem_codes ON redeem_codes.id = redeem_records.code_id").
		Order("redeem_records.id DESC").Offset(offset).Limit(req.Size).
		Find(&records).Error; err != nil {
		return 0, nil, err
	}
	return total, records, nil
}
