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

package inventory

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ChangeRequest 一次背包变动
type ChangeRequest struct {
	UserID         uint64
	ItemID         uint64
	Quantity       int64  // 正数获得, 负数消耗
	BizType        string // 业务类型, 见 constants.go, 最长 32
	BizID          string // 业务单号, 便于按业务回查流水, 最长 64
	IdempotencyKey string // 非空时同一个 key 只生效一次, 需带业务前缀保证全局唯一, 最长 128
	OperatorID     uint64 // 操作人, 0 表示系统
	Remark         string // 最长 255 个字符
}

// Change 变动背包并写一条流水, 所有背包变动都必须走这里
//
// tx 必须是调用方开启的事务, 背包变动与业务写入一起提交或回滚;
// 返回错误时调用方应回滚整个事务。
//
// 获得物品要求物品已上架, 消耗不受上架状态限制。
// 幂等键命中已有流水时不重复变动, 返回已有流水且 replayed 为 true;
// 同一个幂等键但变动内容不同时返回 IdempotencyConflict。
func Change(tx *gorm.DB, req *ChangeRequest) (*Transaction, bool, error) {
	if committer, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok || committer == nil {
		return nil, false, errors.New("inventory: Change must run inside a transaction")
	}
	if err := validateChange(req); err != nil {
		return nil, false, err
	}

	// 幂等: 同一个 key 的并发请求在咨询锁上排队, 后到的能看到先到的已提交流水
	if req.IdempotencyKey != "" {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "blindcrate:inventory:"+req.IdempotencyKey).Error; err != nil {
			return nil, false, err
		}
		// 查不到是常态, 用 Find 避免每次都打 record not found 日志
		var existing Transaction
		if err := tx.Where("idempotency_key = ?", req.IdempotencyKey).Limit(1).Find(&existing).Error; err != nil {
			return nil, false, err
		}
		if existing.ID != 0 {
			if !existing.sameChange(req) {
				return nil, false, errors.New(IdempotencyConflict)
			}
			return &existing, true, nil
		}
	}

	target, err := checkItem(tx, req)
	if err != nil {
		return nil, false, err
	}
	userItem, err := applyQuantity(tx, req)
	if err != nil {
		return nil, false, err
	}

	txn := &Transaction{
		UserID:         req.UserID,
		ItemID:         req.ItemID,
		ItemName:       target.Name,
		Quantity:       req.Quantity,
		QuantityAfter:  userItem.Quantity,
		BizType:        req.BizType,
		BizID:          req.BizID,
		IdempotencyKey: req.IdempotencyKey,
		OperatorID:     req.OperatorID,
		Remark:         req.Remark,
	}
	if err := tx.Create(txn).Error; err != nil {
		// 咨询锁之外的兜底, 正常不会走到
		if db.IsUniqueViolation(err) {
			return nil, false, errors.New(DuplicateRequest)
		}
		return nil, false, err
	}
	return txn, false, nil
}

// validateChange 校验变动内容, 字段超长属于调用方的编程错误
func validateChange(req *ChangeRequest) error {
	if req.UserID == 0 || req.ItemID == 0 || req.BizType == "" {
		return errors.New("inventory: user_id, item_id and biz_type are required")
	}
	if len(req.BizType) > 32 || len(req.BizID) > 64 || len(req.IdempotencyKey) > 128 || utf8.RuneCountInString(req.Remark) > 255 {
		return errors.New("inventory: biz_type, biz_id, idempotency_key or remark too long")
	}
	if req.Quantity == 0 || req.Quantity < -MaxQuantity || req.Quantity > MaxQuantity {
		return errors.New(QuantityInvalid)
	}
	return nil
}

// checkItem 获得时物品必须存在且已上架; 消耗时已删除的物品按占位物品处理, 流水记录占位名称
//
// 获得时以共享锁读物品, 与删除、修改的行锁互斥, 避免刚被删除或下架的物品仍被发放。
func checkItem(tx *gorm.DB, req *ChangeRequest) (*item.Item, error) {
	if req.Quantity < 0 {
		items, err := item.Resolve(tx, []uint64{req.ItemID})
		if err != nil {
			return nil, err
		}
		return items[req.ItemID], nil
	}

	var target item.Item
	if err := target.Exact(tx.Clauses(clause.Locking{Strength: "SHARE"}), req.ItemID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(item.ItemNotFound)
		}
		return nil, err
	}
	if !target.IsEnabled {
		return nil, errors.New(item.ItemDisabled)
	}
	return &target, nil
}

// applyQuantity 原子地变动数量并返回变动后的背包行
//
// 不先查后改, 而是把"数量够不够"写进 UPDATE 的条件里,
// 并发请求会在行锁上排队, 数量不足时不命中任何行。
func applyQuantity(tx *gorm.DB, req *ChangeRequest) (*UserItem, error) {
	var userItem UserItem
	var result *gorm.DB
	if req.Quantity > 0 {
		// 获得: 背包行不存在时直接创建
		result = tx.Raw(
			`INSERT INTO inventory_items (user_id, item_id, quantity) VALUES (?, ?, ?)
			ON CONFLICT (user_id, item_id) DO UPDATE SET
				quantity = inventory_items.quantity + EXCLUDED.quantity,
				updated_at = NOW()
			RETURNING *`,
			req.UserID, req.ItemID, req.Quantity,
		).Scan(&userItem)
	} else {
		result = tx.Raw(
			`UPDATE inventory_items SET quantity = quantity + ?, updated_at = NOW()
			WHERE user_id = ? AND item_id = ? AND quantity + ? >= 0
			RETURNING *`,
			req.Quantity, req.UserID, req.ItemID, req.Quantity,
		).Scan(&userItem)
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New(InsufficientQuantity)
	}
	return &userItem, nil
}

// sameChange 幂等键重放时校验变动内容一致
func (t *Transaction) sameChange(req *ChangeRequest) bool {
	return t.UserID == req.UserID &&
		t.ItemID == req.ItemID &&
		t.Quantity == req.Quantity &&
		t.BizType == req.BizType &&
		t.BizID == req.BizID &&
		t.Remark == req.Remark
}

// UserItemDetail 背包中的一种物品, 附带物品信息, 不含掉落权重
type UserItemDetail struct {
	ItemID        uint64    `json:"item_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Quality       int16     `json:"quality"`
	ImageURL      string    `json:"image_url"`
	SellCurrency  string    `json:"sell_currency"`
	SellPrice     int64     `json:"sell_price"`
	Width         int16     `json:"width"`
	Height        int16     `json:"height"`
	IsEnabled     bool      `json:"is_enabled"`
	IsPlaceholder bool      `json:"is_placeholder"` // 物品已被删除, 按占位物品展示
	Quantity      int64     `json:"quantity"`
	CreatedAt     time.Time `json:"created_at"` // 首次获得时间
	UpdatedAt     time.Time `json:"updated_at"`
}

// QueryUserItems 获取用户背包中数量大于 0 的物品, 按品质从高到低排序
//
// 已删除的物品按占位物品返回, 品质筛选和排序也按占位物品的品质处理。
func QueryUserItems(ctx context.Context, userID uint64, req *ListItemsRequest) (int64, []UserItemDetail, error) {
	offset := (req.Current - 1) * req.Size

	quality := fmt.Sprintf("COALESCE(items.quality, %d)", item.QualityGray)
	query := db.DB(ctx).Model(&UserItem{}).
		Joins("LEFT JOIN items ON items.id = inventory_items.item_id AND items.deleted_at IS NULL").
		Where("inventory_items.user_id = ? AND inventory_items.quantity > 0", userID)
	if req.Quality != nil {
		query = query.Where(quality+" = ?", *req.Quality)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var rows []UserItem
	if err := query.Select("inventory_items.*").
		Order(quality + " DESC, inventory_items.item_id").Offset(offset).Limit(req.Size).
		Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	ids := make([]uint64, len(rows))
	for i := range rows {
		ids[i] = rows[i].ItemID
	}
	items, err := item.Resolve(db.DB(ctx), ids)
	if err != nil {
		return 0, nil, err
	}

	details := make([]UserItemDetail, len(rows))
	for i, row := range rows {
		target := items[row.ItemID]
		details[i] = UserItemDetail{
			ItemID:        row.ItemID,
			Name:          target.Name,
			Description:   target.Description,
			Quality:       target.Quality,
			ImageURL:      target.ImageURL,
			SellCurrency:  target.SellCurrency,
			SellPrice:     target.SellPrice,
			Width:         target.Width,
			Height:        target.Height,
			IsEnabled:     target.IsEnabled,
			IsPlaceholder: target.IsPlaceholder,
			Quantity:      row.Quantity,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		}
	}
	return total, details, nil
}

// TransactionItem 用户自己看到的流水, 不含操作人和幂等键
type TransactionItem struct {
	ID            uint64    `json:"id"`
	ItemID        uint64    `json:"item_id"`
	ItemName      string    `json:"item_name"`
	Quantity      int64     `json:"quantity"`
	QuantityAfter int64     `json:"quantity_after"`
	BizType       string    `json:"biz_type"`
	BizID         string    `json:"biz_id"`
	Remark        string    `json:"remark"`
	CreatedAt     time.Time `json:"created_at"`
}

// QueryUserTransactions 获取用户自己的背包流水
func QueryUserTransactions(ctx context.Context, userID uint64, req *ListTransactionsRequest) (int64, []TransactionItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&Transaction{}).Where("user_id = ?", userID)
	if req.ItemID != nil {
		query = query.Where("item_id = ?", *req.ItemID)
	}
	if req.BizType != "" {
		query = query.Where("biz_type = ?", req.BizType)
	}
	if req.StartTime != nil {
		query = query.Where("created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("created_at < ?", *req.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	items := []TransactionItem{}
	if err := query.Order("id DESC").Offset(offset).Limit(req.Size).Scan(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}
