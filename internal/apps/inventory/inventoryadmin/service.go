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

package inventoryadmin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/admin"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
)

// queryUserInventory 获取指定用户背包中的物品
func queryUserInventory(ctx context.Context, userID uint64, req *inventory.ListItemsRequest) (int64, []inventory.UserItemDetail, error) {
	var user auth.User
	if err := user.Exact(db.DB(ctx), userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil, errors.New(auth.UserNotFound)
		}
		return 0, nil, err
	}
	return inventory.QueryUserItems(ctx, userID, req)
}

// doAdjustUserInventory 管理员给其他用户发放或扣除物品, 不允许给自己调整
//
// 幂等键由前端在每次打开表单时生成, 防止重复点击导致重复发放。
func doAdjustUserInventory(ctx context.Context, operatorID, userID uint64, req *AdjustUserInventoryRequest) (*inventory.Transaction, bool, error) {
	if operatorID == userID {
		return nil, false, errors.New(admin.CannotOperateSelf)
	}
	remark, err := admin.NormalizeRemark(req.Remark)
	if err != nil {
		return nil, false, err
	}

	var (
		txn      *inventory.Transaction
		replayed bool
	)
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var user auth.User
		if err := user.Exact(tx, userID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(auth.UserNotFound)
			}
			return err
		}

		var err error
		txn, replayed, err = inventory.Change(tx, &inventory.ChangeRequest{
			UserID:         userID,
			ItemID:         req.ItemID,
			Quantity:       req.Quantity,
			BizType:        inventory.BizAdminAdjust,
			IdempotencyKey: fmt.Sprintf("%s:%d:%s", inventory.BizAdminAdjust, operatorID, req.IdempotencyKey),
			OperatorID:     operatorID,
			Remark:         remark,
		})
		return err
	})
	if err != nil {
		return nil, false, err
	}
	return txn, replayed, nil
}

// InventoryTransactionItem 管理后台展示的背包流水, 带上用户和操作人的用户名
type InventoryTransactionItem struct {
	ID               uint64    `json:"id"`
	UserID           uint64    `json:"user_id"`
	Username         string    `json:"username"` // 用户已不存在时为空
	ItemID           uint64    `json:"item_id"`
	ItemName         string    `json:"item_name"`
	Quantity         int64     `json:"quantity"`
	QuantityAfter    int64     `json:"quantity_after"`
	BizType          string    `json:"biz_type"`
	BizID            string    `json:"biz_id"`
	IdempotencyKey   string    `json:"idempotency_key"`
	OperatorID       uint64    `json:"operator_id"`
	OperatorUsername string    `json:"operator_username"` // 系统操作或用户已不存在时为空
	Remark           string    `json:"remark"`
	CreatedAt        time.Time `json:"created_at"`
}

// QueryInventoryTransactions 获取全部用户的背包流水
func QueryInventoryTransactions(ctx context.Context, req *ListInventoryTransactionsRequest) (int64, []InventoryTransactionItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&inventory.Transaction{})
	if req.UserID != nil {
		query = query.Where("inventory_transactions.user_id = ?", *req.UserID)
	}
	if req.OperatorID != nil {
		query = query.Where("inventory_transactions.operator_id = ?", *req.OperatorID)
	}
	if req.ItemID != nil {
		query = query.Where("inventory_transactions.item_id = ?", *req.ItemID)
	}
	if req.BizType != "" {
		query = query.Where("inventory_transactions.biz_type = ?", req.BizType)
	}
	if req.BizID != "" {
		query = query.Where("inventory_transactions.biz_id = ?", req.BizID)
	}
	if req.StartTime != nil {
		query = query.Where("inventory_transactions.created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("inventory_transactions.created_at < ?", *req.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	items := []InventoryTransactionItem{}
	if err := query.
		Select("inventory_transactions.*, COALESCE(users.username, '') AS username, COALESCE(operators.username, '') AS operator_username").
		Joins("LEFT JOIN users ON users.id = inventory_transactions.user_id").
		Joins("LEFT JOIN users AS operators ON operators.id = inventory_transactions.operator_id").
		Order("inventory_transactions.id DESC").Offset(offset).Limit(req.Size).
		Scan(&items).Error; err != nil {
		return 0, nil, err
	}

	return total, items, nil
}
