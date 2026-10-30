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

package mailboxadmin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/mailbox"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// maxMissingShown 收件人不存在时提示中最多列出的用户 ID 数
const maxMissingShown = 10

// doSendMails 校验内容和收件人后发给指定用户, 返回实际写入的封数和校验后的内容; 幂等键相同的重复提交不会重复发送
func doSendMails(ctx context.Context, req *SendMailsRequest) (int64, *mailbox.Letter, error) {
	userIDs := slices.Compact(slices.Sorted(slices.Values(req.UserIDs)))
	if len(userIDs) == 0 || len(userIDs) > mailbox.MaxRecipients {
		return 0, nil, &mailbox.InvalidError{Reason: mailbox.RecipientsInvalid}
	}
	expiresAt, err := mailbox.Expiry(ctx, req.ExpireDays)
	if err != nil {
		return 0, nil, err
	}
	letter := mailbox.Letter{Sender: req.Sender, Title: req.Title, Content: req.Content, Attachments: req.Attachments}
	var sent *mailbox.Sent
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := mailbox.ValidateLetter(tx, &letter); err != nil {
			return err
		}
		var found []uint64
		if err := tx.Model(&auth.User{}).Where("id IN ?", userIDs).Pluck("id", &found).Error; err != nil {
			return err
		}
		if len(found) != len(userIDs) {
			var missing []string
			for _, id := range userIDs {
				if !slices.Contains(found, id) && len(missing) < maxMissingShown {
					missing = append(missing, strconv.FormatUint(id, 10))
				}
			}
			return &mailbox.InvalidError{Reason: fmt.Sprintf(mailbox.UsersNotFound, strings.Join(missing, ", "))}
		}
		mails := make([]mailbox.Mail, len(userIDs))
		for i, id := range userIDs {
			mails[i] = mailbox.Mail{
				UserID:      id,
				Source:      mailbox.SourceAdmin,
				SourceID:    req.IdempotencyKey,
				Sender:      letter.Sender,
				Title:       letter.Title,
				Content:     letter.Content,
				Attachments: letter.Attachments,
				ExpiresAt:   expiresAt,
			}
		}
		sent, err = mailbox.Send(tx, mails)
		return err
	})
	if err != nil {
		return 0, nil, err
	}
	sent.Push(ctx)
	return sent.Count, &letter, nil
}

// MailItem 后台展示的邮件, 带上收件人的用户名和删除时间
type MailItem struct {
	mailbox.Mail
	Username  string     `json:"username"`   // 用户已不存在时为空
	RemovedAt *time.Time `json:"removed_at"` // 玩家删除或超出保留封数被移出信箱的时间, 在信箱中时为空
}

// QueryMails 获取邮件列表, 从新到旧, 包括已删除的
func QueryMails(ctx context.Context, req *ListMailsRequest) (int64, []MailItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Unscoped().Model(&mailbox.Mail{})
	if req.UserID != nil {
		query = query.Where("mails.user_id = ?", *req.UserID)
	}
	if req.Source != "" {
		query = query.Where("mails.source = ?", req.Source)
	}
	if req.Keyword != "" {
		query = query.Where("mails.title ILIKE ?", "%"+db.EscapeLike(req.Keyword)+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	mails := []MailItem{}
	if err := query.
		Select("mails.*, mails.deleted_at AS removed_at, COALESCE(users.username, '') AS username").
		Joins("LEFT JOIN users ON users.id = mails.user_id").
		Order("mails.id DESC").Offset(offset).Limit(req.Size).
		Find(&mails).Error; err != nil {
		return 0, nil, err
	}
	return total, mails, nil
}

// BroadcastItem 后台展示的全服邮件, 带上发送人的用户名、已放进信箱和已领取附件的人数
type BroadcastItem struct {
	mailbox.Broadcast
	CreatedUsername string `json:"created_username"`
	RevokedUsername string `json:"revoked_username"`
	Expanded        int64  `json:"expanded"` // 已放进信箱的人数, 包括之后删除的
	Claimed         int64  `json:"claimed"`  // 已领取附件的人数
}

// QueryBroadcasts 获取全服邮件列表, 从新到旧
func QueryBroadcasts(ctx context.Context, req *ListBroadcastsRequest) (int64, []BroadcastItem, error) {
	offset := (req.Current - 1) * req.Size

	query := db.DB(ctx).Model(&mailbox.Broadcast{})
	if req.Keyword != "" {
		query = query.Where("mail_broadcasts.title ILIKE ?", "%"+db.EscapeLike(req.Keyword)+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	// 统计走 mails 的来源唯一索引
	const mailsOf = "FROM mails WHERE mails.source = @source AND mails.source_id = CAST(mail_broadcasts.id AS VARCHAR)"
	items := []BroadcastItem{}
	if err := query.
		Select("mail_broadcasts.*, COALESCE(creators.username, '') AS created_username, COALESCE(revokers.username, '') AS revoked_username, "+
			"(SELECT COUNT(*) "+mailsOf+") AS expanded, (SELECT COUNT(*) "+mailsOf+" AND mails.claimed_at IS NOT NULL) AS claimed",
			map[string]any{"source": mailbox.SourceBroadcast}).
		Joins("LEFT JOIN users AS creators ON creators.id = mail_broadcasts.created_by").
		Joins("LEFT JOIN users AS revokers ON revokers.id = mail_broadcasts.revoked_by").
		Order("mail_broadcasts.id DESC").Offset(offset).Limit(req.Size).
		Find(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}

// doCreateBroadcast 校验并发送全服邮件; 之后注册的用户是否能收到见 IncludeNewUsers
//
// 幂等键已发送过时返回已有的全服邮件, replayed 为 true。
func doCreateBroadcast(ctx context.Context, operatorID uint64, req *BroadcastRequest) (_ *mailbox.Broadcast, replayed bool, _ error) {
	expiresAt, err := mailbox.Expiry(ctx, req.ExpireDays)
	if err != nil {
		return nil, false, err
	}
	letter := mailbox.Letter{Sender: req.Sender, Title: req.Title, Content: req.Content, Attachments: req.Attachments}
	target := &mailbox.Broadcast{Requirements: req.Requirements, ExpiresAt: expiresAt, IdempotencyKey: req.IdempotencyKey, CreatedBy: operatorID}
	if target.Requirements == nil {
		target.Requirements = []condition.Condition{}
	}
	if !req.IncludeNewUsers {
		now := time.Now()
		target.RegisteredBefore = &now
	}
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := mailbox.ValidateLetter(tx, &letter); err != nil {
			return err
		}
		if err := condition.Validate(tx, target.Requirements); err != nil {
			var invalid *condition.InvalidError
			if errors.As(err, &invalid) {
				return &mailbox.InvalidError{Reason: fmt.Sprintf(mailbox.RequirementInvalid, invalid.Error())}
			}
			return err
		}
		target.Sender, target.Title, target.Content, target.Attachments = letter.Sender, letter.Title, letter.Content, letter.Attachments
		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "idempotency_key"}}, DoNothing: true}).Create(target)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			replayed = true
			*target = mailbox.Broadcast{}
			return tx.Where("idempotency_key = ?", req.IdempotencyKey).First(target).Error
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if !replayed {
		mailbox.NotifyAll(ctx)
	}
	return target, replayed, nil
}

// doRevokeBroadcast 撤回全服邮件: 不再展开, 已放进信箱但附件未领取的邮件移出信箱, 返回撤回后的全服邮件和移出的封数
func doRevokeBroadcast(ctx context.Context, operatorID, broadcastID uint64) (*mailbox.Broadcast, int64, error) {
	var (
		target  mailbox.Broadcast
		removed int64
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", broadcastID).First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(mailbox.BroadcastNotFound)
			}
			return err
		}
		if target.RevokedAt != nil {
			return errors.New(mailbox.BroadcastRevoked)
		}
		now := time.Now()
		target.RevokedAt = &now
		target.RevokedBy = operatorID
		if err := tx.Save(&target).Error; err != nil {
			return err
		}
		result := tx.Where("source = ? AND source_id = ? AND claimed_at IS NULL", mailbox.SourceBroadcast, strconv.FormatUint(target.ID, 10)).
			Delete(&mailbox.Mail{})
		removed = result.RowsAffected
		return result.Error
	})
	if err != nil {
		return nil, 0, err
	}
	if removed > 0 {
		mailbox.NotifyAll(ctx)
	}
	return &target, removed, nil
}
