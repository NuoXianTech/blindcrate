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

package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/audit"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
)

// likeEscaper 转义 LIKE 通配符, 用户名里常见的下划线也是通配符
var likeEscaper = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`)

// QueryUsersList 获取用户列表
func QueryUsersList(ctx context.Context, req *ListUsersRequest) (int64, []auth.User, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&auth.User{})
	if req.Keyword != "" {
		keyword := likeEscaper.Replace(strings.ToLower(req.Keyword)) + "%"
		query = query.Where("LOWER(username) LIKE ? OR LOWER(email) LIKE ?", keyword, keyword)
	}
	if req.IsAdmin != nil {
		query = query.Where("is_admin = ?", *req.IsAdmin)
	}
	if req.IsDisabled != nil {
		query = query.Where("is_disabled = ?", *req.IsDisabled)
	}
	if req.IsBanned != nil {
		if *req.IsBanned {
			query = query.Where("banned_until > ?", time.Now())
		} else {
			query = query.Where("banned_until IS NULL OR banned_until <= ?", time.Now())
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	users := []auth.User{}
	if err := query.Order("id DESC").Offset(offset).Limit(req.Size).Find(&users).Error; err != nil {
		return 0, nil, err
	}

	return total, users, nil
}

// updateOtherUser 管理员修改其他用户的字段, 不允许操作自己
func updateOtherUser(ctx context.Context, operatorID, userID uint64, values map[string]interface{}) error {
	if operatorID == userID {
		return errors.New(CannotOperateSelf)
	}
	tx := db.DB(ctx).Model(&auth.User{}).Where("id = ?", userID).Updates(values)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return errors.New(auth.UserNotFound)
	}
	return nil
}

// doBanUser 封禁用户到指定时间
func doBanUser(ctx context.Context, operatorID, userID uint64, until time.Time, reason string) error {
	if !until.After(time.Now()) {
		return errors.New(BanUntilInvalid)
	}
	return updateOtherUser(ctx, operatorID, userID, map[string]interface{}{"banned_until": until, "ban_reason": reason})
}

// doUnbanUser 解除封禁
func doUnbanUser(ctx context.Context, operatorID, userID uint64) error {
	return updateOtherUser(ctx, operatorID, userID, map[string]interface{}{"banned_until": nil, "ban_reason": ""})
}

// doSetUserDisabled 停用或启用用户
func doSetUserDisabled(ctx context.Context, operatorID, userID uint64, disabled bool) error {
	return updateOtherUser(ctx, operatorID, userID, map[string]interface{}{"is_disabled": disabled})
}

type AuditLogItem struct {
	ID            uint64          `json:"id"`
	ActorID       uint64          `json:"actor_id"`
	ActorUsername string          `json:"actor_username"` // 操作人用户名, 未登录或用户已不存在时为空
	Scope         string          `json:"scope"`
	Action        string          `json:"action"`
	TargetType    string          `json:"target_type"`
	TargetID      uint64          `json:"target_id"`
	Detail        json.RawMessage `json:"detail" swaggertype:"object"`
	IP            string          `json:"ip"`
	UserAgent     string          `json:"user_agent"`
	CreatedAt     time.Time       `json:"created_at"`
}

// QueryAuditLogs 获取操作日志列表
func QueryAuditLogs(ctx context.Context, req *ListAuditLogsRequest) (int64, []AuditLogItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&audit.Log{})
	if req.ActorID != nil {
		query = query.Where("audit_logs.actor_id = ?", *req.ActorID)
	}
	if req.Scope != "" {
		query = query.Where("audit_logs.scope = ?", req.Scope)
	}
	if req.Action != "" {
		query = query.Where("audit_logs.action = ?", req.Action)
	}
	if req.TargetType != "" {
		query = query.Where("audit_logs.target_type = ?", req.TargetType)
	}
	if req.TargetID != nil {
		query = query.Where("audit_logs.target_id = ?", *req.TargetID)
	}
	if req.StartTime != nil {
		query = query.Where("audit_logs.created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("audit_logs.created_at < ?", *req.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	logs := []AuditLogItem{}
	if err := query.
		Select("audit_logs.*, COALESCE(users.username, '') AS actor_username").
		Joins("LEFT JOIN users ON users.id = audit_logs.actor_id").
		Order("audit_logs.id DESC").Offset(offset).Limit(req.Size).
		Scan(&logs).Error; err != nil {
		return 0, nil, err
	}

	return total, logs, nil
}

type WalletTransactionItem struct {
	ID               uint64    `json:"id"`
	UserID           uint64    `json:"user_id"`
	Username         string    `json:"username"` // 用户已不存在时为空
	Currency         string    `json:"currency"`
	Amount           int64     `json:"amount"`
	FrozenAmount     int64     `json:"frozen_amount"`
	BalanceAfter     int64     `json:"balance_after"`
	FrozenAfter      int64     `json:"frozen_after"`
	BizType          string    `json:"biz_type"`
	BizID            string    `json:"biz_id"`
	IdempotencyKey   string    `json:"idempotency_key"`
	OperatorID       uint64    `json:"operator_id"`
	OperatorUsername string    `json:"operator_username"` // 系统操作或用户已不存在时为空
	Remark           string    `json:"remark"`
	CreatedAt        time.Time `json:"created_at"`
}

// QueryWalletTransactions 获取全部用户的余额流水
func QueryWalletTransactions(ctx context.Context, req *ListWalletTransactionsRequest) (int64, []WalletTransactionItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&wallet.Transaction{})
	if req.UserID != nil {
		query = query.Where("wallet_transactions.user_id = ?", *req.UserID)
	}
	if req.OperatorID != nil {
		query = query.Where("wallet_transactions.operator_id = ?", *req.OperatorID)
	}
	if req.Currency != "" {
		query = query.Where("wallet_transactions.currency = ?", req.Currency)
	}
	if req.BizType != "" {
		query = query.Where("wallet_transactions.biz_type = ?", req.BizType)
	}
	if req.BizID != "" {
		query = query.Where("wallet_transactions.biz_id = ?", req.BizID)
	}
	if req.StartTime != nil {
		query = query.Where("wallet_transactions.created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("wallet_transactions.created_at < ?", *req.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	items := []WalletTransactionItem{}
	if err := query.
		Select("wallet_transactions.*, COALESCE(users.username, '') AS username, COALESCE(operators.username, '') AS operator_username").
		Joins("LEFT JOIN users ON users.id = wallet_transactions.user_id").
		Joins("LEFT JOIN users AS operators ON operators.id = wallet_transactions.operator_id").
		Order("wallet_transactions.id DESC").Offset(offset).Limit(req.Size).
		Scan(&items).Error; err != nil {
		return 0, nil, err
	}

	return total, items, nil
}

// queryUserBalances 获取指定用户的各货币余额(含已停用货币)
func queryUserBalances(ctx context.Context, userID uint64) ([]wallet.BalanceItem, error) {
	var user auth.User
	if err := user.Exact(db.DB(ctx), userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(auth.UserNotFound)
		}
		return nil, err
	}
	return wallet.QueryBalances(ctx, userID, true)
}

// doAdjustUserBalance 管理员给其他用户调账, 不允许给自己调账
//
// 幂等键由前端在每次打开调账表单时生成, 防止重复点击导致重复入账。
func doAdjustUserBalance(ctx context.Context, operatorID, userID uint64, req *AdjustUserBalanceRequest) (*wallet.Transaction, bool, error) {
	if operatorID == userID {
		return nil, false, errors.New(CannotOperateSelf)
	}
	remark := strings.TrimSpace(req.Remark)
	if n := utf8.RuneCountInString(remark); n < 1 || n > 255 || strings.ContainsRune(remark, 0) {
		return nil, false, errors.New(RemarkInvalid)
	}

	var (
		txn      *wallet.Transaction
		replayed bool
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var user auth.User
		if err := user.Exact(tx, userID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(auth.UserNotFound)
			}
			return err
		}

		var err error
		txn, replayed, err = wallet.Change(tx, &wallet.ChangeRequest{
			UserID:         userID,
			Currency:       req.Currency,
			Amount:         req.Amount,
			BizType:        wallet.BizAdminAdjust,
			IdempotencyKey: fmt.Sprintf("%s:%d:%s", wallet.BizAdminAdjust, operatorID, req.IdempotencyKey),
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
