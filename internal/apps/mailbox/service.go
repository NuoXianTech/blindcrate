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

package mailbox

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/ledger"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/ws"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	ledger.RegisterBizType(ledger.KindWallet, BizClaim, "邮件附件")
	ledger.RegisterBizType(ledger.KindInventory, BizClaim, "邮件附件")
	audit.RegisterAction(ActionSend, "发送邮件")
	audit.RegisterAction(ActionBroadcast, "发送全服邮件")
	audit.RegisterAction(ActionRevoke, "撤回全服邮件")
	RegisterSource(SourceAdmin, "管理员发送")
	RegisterSource(SourceBroadcast, "全服邮件")
}

// Settings 游戏邮件的设置
var Settings = settings.Register(settings.Group[MailboxSettings]{
	Key:         "mailbox",
	Label:       "游戏邮件",
	Description: "游戏内邮件的有效期和保留封数",
	Category:    settings.CategoryGame,
	Sort:        30,
	Defaults:    func() *MailboxSettings { return &MailboxSettings{ExpireDays: 30, MaxKept: 100, RetentionDays: 30} },
})

// MailboxSettings 游戏邮件
type MailboxSettings struct {
	ExpireDays    int `json:"expire_days" label:"默认有效期" help:"发送时没有指定有效期的邮件在这么多天后过期, 过期后从信箱中消失, 附件不能再领取; 单位天" binding:"min=1,max=365"`
	MaxKept       int `json:"max_kept" label:"每人保留封数" help:"信箱中的邮件超过这个数量时, 每天定时移除最早的邮件, 附件未领取且未过期的邮件不移除" binding:"min=10,max=1000"`
	RetentionDays int `json:"retention_days" label:"过期邮件保留天数" help:"过期后再保留这么多天供后台查询, 之后每天定时删除" binding:"min=1,max=3650"`
}

// Source 一种邮件来源
type Source struct {
	Code  string `json:"code"`
	Label string `json:"label"` // 后台展示的名称, 如 "全服邮件"
}

var (
	// sources 已登记的邮件来源, 按登记顺序, 只在启动前写入
	sources []Source
	// sourcePattern 邮件来源的格式, 与 mails.source 的长度一致
	sourcePattern = regexp.MustCompile(`^[a-z0-9_]{1,16}$`)
)

// RegisterSource 登记一种邮件来源, 需要在启动前调用; 格式不正确或重复时 panic
func RegisterSource(code, label string) {
	if !sourcePattern.MatchString(code) || label == "" {
		panic(fmt.Sprintf("mailbox: invalid source %q", code))
	}
	if slices.ContainsFunc(sources, func(s Source) bool { return s.Code == code }) {
		panic(fmt.Sprintf("mailbox: source %s already registered", code))
	}
	sources = append(sources, Source{Code: code, Label: label})
}

// Sources 全部邮件来源, 按登记顺序
func Sources() []Source {
	return slices.Clone(sources)
}

// InvalidError 邮件内容不合法, Error 返回的提示可以直接展示给管理员
type InvalidError struct {
	Reason string
}

func (e *InvalidError) Error() string { return e.Reason }

// ValidateLetter 发送前校验邮件内容: 去掉首尾空白, 发件人为空时为 "系统", 合并附件中的同类项
//
// 不合法时返回 *InvalidError; 其他错误为查库失败等内部错误。
func ValidateLetter(tx *gorm.DB, l *Letter) error {
	l.Sender = strings.TrimSpace(l.Sender)
	if l.Sender == "" {
		l.Sender = defaultSender
	}
	if utf8.RuneCountInString(l.Sender) > maxSender {
		return &InvalidError{Reason: SenderInvalid}
	}
	l.Title = strings.TrimSpace(l.Title)
	if n := utf8.RuneCountInString(l.Title); n < 1 || n > maxTitle {
		return &InvalidError{Reason: TitleInvalid}
	}
	l.Content = strings.TrimSpace(l.Content)
	if utf8.RuneCountInString(l.Content) > maxContent {
		return &InvalidError{Reason: ContentInvalid}
	}
	attachments, err := reward.Check(tx, l.Attachments)
	if err != nil {
		var invalid *reward.InvalidError
		if errors.As(err, &invalid) {
			return &InvalidError{Reason: fmt.Sprintf(AttachmentInvalid, invalid.Reason)}
		}
		return err
	}
	l.Attachments = attachments
	return nil
}

// Expiry 有效期为 days 天的邮件的过期时间, days 为 0 时使用设置中的默认有效期
func Expiry(ctx context.Context, days int) (time.Time, error) {
	if days < 0 || days > MaxExpireDays {
		return time.Time{}, &InvalidError{Reason: ExpireInvalid}
	}
	if days == 0 {
		cfg, err := Settings.Get(ctx)
		if err != nil {
			return time.Time{}, err
		}
		days = cfg.ExpireDays
	}
	return time.Now().Add(time.Duration(days) * 24 * time.Hour), nil
}

// Sent 写入的邮件, 事务提交后调用 Push 通知收件人
type Sent struct {
	Count   int64 // 实际写入的封数, 来源单号已有邮件的不计
	userIDs []uint64
}

// Push 通知收件人重新获取未读数, 需要在事务提交后调用; 推送失败只记录日志
func (s *Sent) Push(ctx context.Context) {
	if s == nil || s.Count == 0 {
		return
	}
	if err := ws.SendToUsers(ctx, s.userIDs, MessageNew, nil); err != nil {
		logger.ErrorF(ctx, "[Mailbox] push new mail to %d users failed: %v", len(s.userIDs), err)
	}
}

// NotifyAll 通知全部在线用户重新获取未读数, 用于发送和撤回全服邮件; 推送失败只记录日志
func NotifyAll(ctx context.Context) {
	if err := ws.Broadcast(ctx, MessageNew, nil); err != nil {
		logger.ErrorF(ctx, "[Mailbox] broadcast new mail failed: %v", err)
	}
}

// Send 写入邮件, 同一来源单号对同一用户已有邮件(包括已删除的)时跳过; tx 可以是调用方的事务
//
// 内容由调用方保证合法, 通常用 ValidateLetter 校验一次后发给多人; 发件人为空时为 "系统", 过期时间为空时按默认有效期。
// 来源需要用 RegisterSource 登记, 来源单号不能为空。mails 不会被修改。
func Send(tx *gorm.DB, mails []Mail) (*Sent, error) {
	if len(mails) == 0 {
		return &Sent{}, nil
	}
	var defaultExpiry time.Time
	rows := make([]Mail, len(mails))
	users := map[uint64]struct{}{}
	for i, m := range mails {
		if !slices.ContainsFunc(sources, func(s Source) bool { return s.Code == m.Source }) {
			return nil, fmt.Errorf("mailbox: unknown source %q", m.Source)
		}
		if m.SourceID == "" || len(m.SourceID) > maxSourceIDLen || m.UserID == 0 {
			return nil, fmt.Errorf("mailbox: invalid source id %q for user %d", m.SourceID, m.UserID)
		}
		if m.Sender == "" {
			m.Sender = defaultSender
		}
		if m.ExpiresAt.IsZero() {
			if defaultExpiry.IsZero() {
				var err error
				if defaultExpiry, err = Expiry(tx.Statement.Context, 0); err != nil {
					return nil, err
				}
			}
			m.ExpiresAt = defaultExpiry
		}
		m.ID, m.ReadAt, m.ClaimedAt, m.DeletedAt = 0, nil, nil, gorm.DeletedAt{}
		rows[i] = m
		users[m.UserID] = struct{}{}
	}
	// 跳过的行不会返回 ID, rows 中的 ID 不可靠, 只取写入的行数
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source"}, {Name: "source_id"}, {Name: "user_id"}},
		DoNothing: true,
	}).Omit("id").CreateInBatches(&rows, 500)
	if result.Error != nil {
		return nil, result.Error
	}
	return &Sent{Count: result.RowsAffected, userIDs: slices.Sorted(maps.Keys(users))}, nil
}

// Expand 把用户可以收到、还没有展开过的全服邮件写成用户自己的邮件
//
// 在玩家打开信箱或获取未读数时调用; 不满足收件条件的全服邮件这次跳过, 之后满足时仍会展开。
// 全服邮件加共享锁, 与撤回互斥: 撤回先提交时这里读到的是已撤回的状态, 这里先提交时撤回能移出刚展开的邮件。
func Expand(ctx context.Context, user *auth.User) error {
	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		var broadcasts []Broadcast
		if err := tx.Clauses(clause.Locking{Strength: "SHARE", Table: clause.Table{Name: Broadcast{}.TableName()}}).
			Where("revoked_at IS NULL AND expires_at > ? AND (registered_before IS NULL OR registered_before > ?)", now, user.CreatedAt).
			Where("NOT EXISTS (SELECT 1 FROM mails WHERE mails.user_id = ? AND mails.source = ? AND mails.source_id = CAST(mail_broadcasts.id AS VARCHAR))", user.ID, SourceBroadcast).
			Order("id").Find(&broadcasts).Error; err != nil {
			return err
		}
		mails := make([]Mail, 0, len(broadcasts))
		for _, b := range broadcasts {
			unmet, err := condition.FirstUnmet(tx, user.ID, b.Requirements)
			if err != nil {
				return err
			}
			if unmet != nil {
				continue
			}
			mails = append(mails, Mail{
				UserID:      user.ID,
				Source:      SourceBroadcast,
				SourceID:    strconv.FormatUint(b.ID, 10),
				Sender:      b.Sender,
				Title:       b.Title,
				Content:     b.Content,
				Attachments: b.Attachments,
				ExpiresAt:   b.ExpiresAt,
			})
		}
		_, err := Send(tx, mails)
		return err
	})
}

// visible 用户信箱中的邮件: 未删除且未过期
func visible(tx *gorm.DB, userID uint64, now time.Time) *gorm.DB {
	return tx.Model(&Mail{}).Where("user_id = ? AND expires_at > ?", userID, now)
}

// Counts 信箱中未读和附件未领取的封数
type Counts struct {
	Unread    int64 `json:"unread"`
	Claimable int64 `json:"claimable"`
}

// QueryCounts 先展开全服邮件, 再统计未读和附件未领取的封数
func QueryCounts(ctx context.Context, user *auth.User) (*Counts, error) {
	if err := Expand(ctx, user); err != nil {
		return nil, err
	}
	var counts Counts
	if err := visible(db.DB(ctx), user.ID, time.Now()).
		Select("COUNT(*) FILTER (WHERE read_at IS NULL) AS unread, COUNT(*) FILTER (WHERE claimed_at IS NULL AND attachments <> '{}'::jsonb) AS claimable").
		Scan(&counts).Error; err != nil {
		return nil, err
	}
	return &counts, nil
}

// MailView 玩家看到的邮件, 附件带名称
type MailView struct {
	ID          uint64        `json:"id"`
	Source      string        `json:"source"`
	Sender      string        `json:"sender"`
	Title       string        `json:"title"`
	Content     string        `json:"content"`
	Attachments []reward.Line `json:"attachments"`
	ReadAt      *time.Time    `json:"read_at"`
	ClaimedAt   *time.Time    `json:"claimed_at"`
	ExpiresAt   time.Time     `json:"expires_at"`
	CreatedAt   time.Time     `json:"created_at"`
}

// QueryMails 先展开全服邮件, 再分页获取信箱中的邮件, 从新到旧
func QueryMails(ctx context.Context, user *auth.User, offset, limit int) (int64, []MailView, error) {
	if err := Expand(ctx, user); err != nil {
		return 0, nil, err
	}
	tx := db.DB(ctx)
	query := visible(tx, user.ID, time.Now())
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var mails []Mail
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&mails).Error; err != nil {
		return 0, nil, err
	}
	bundles := make([]reward.Bundle, len(mails))
	for i, m := range mails {
		bundles[i] = m.Attachments
	}
	lines, err := reward.Lines(tx, bundles...)
	if err != nil {
		return 0, nil, err
	}
	views := make([]MailView, len(mails))
	for i, m := range mails {
		views[i] = MailView{
			ID:          m.ID,
			Source:      m.Source,
			Sender:      m.Sender,
			Title:       m.Title,
			Content:     m.Content,
			Attachments: lines[i],
			ReadAt:      m.ReadAt,
			ClaimedAt:   m.ClaimedAt,
			ExpiresAt:   m.ExpiresAt,
			CreatedAt:   m.CreatedAt,
		}
	}
	return total, views, nil
}

// lockMail 锁定用户信箱中的一封邮件, 包括已过期的
func lockMail(tx *gorm.DB, userID, mailID uint64) (*Mail, error) {
	var m Mail
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", mailID, userID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(MailNotFound)
		}
		return nil, err
	}
	return &m, nil
}

// Read 标记为已读, 已读的邮件不变
func Read(ctx context.Context, userID, mailID uint64) error {
	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		m, err := lockMail(tx, userID, mailID)
		if err != nil {
			return err
		}
		if m.ReadAt != nil {
			return nil
		}
		return tx.Model(m).UpdateColumn("read_at", time.Now()).Error
	})
}

// ReadAll 把信箱中的未读邮件全部标记为已读, 返回标记的封数
func ReadAll(ctx context.Context, userID uint64) (int64, error) {
	now := time.Now()
	result := visible(db.DB(ctx), userID, now).Where("read_at IS NULL").UpdateColumn("read_at", now)
	return result.RowsAffected, result.Error
}

// Delete 从信箱中删除, 附件未领取且未过期的邮件不能删除
func Delete(ctx context.Context, userID, mailID uint64) error {
	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		m, err := lockMail(tx, userID, mailID)
		if err != nil {
			return err
		}
		if m.Claimable(time.Now()) {
			return errors.New(UnclaimedAttachments)
		}
		return tx.Delete(m).Error
	})
}

// ClaimResult 领取附件的结果
type ClaimResult struct {
	Claimed  int           `json:"claimed"`  // 领取了附件的封数
	Skipped  int           `json:"skipped"`  // 附件暂时无法领取而跳过的封数, 只有一键领取会跳过
	Received []reward.Line `json:"received"` // 实际得到的货币和物品, 已下架的物品按回收价折成了货币
}

// Claim 领取一封邮件的附件, 同时标记为已读
//
// 附件中已下架或已删除的物品按回收价折成货币; 货币已停用时返回 AttachmentUnavailable, 可以之后再领取。
func Claim(ctx context.Context, userID, mailID uint64) (*ClaimResult, error) {
	received, err := claim(ctx, userID, mailID)
	if err != nil {
		return nil, err
	}
	lines, err := reward.Lines(db.DB(ctx), received)
	if err != nil {
		return nil, err
	}
	return &ClaimResult{Claimed: 1, Received: lines[0]}, nil
}

// ClaimAll 领取信箱中全部可以领取的附件, 从旧到新, 一次最多 maxClaimAll 封
//
// 每封邮件单独一个事务, 附件暂时无法领取的跳过; 遇到内部错误时停止, 已领取的不回滚。
func ClaimAll(ctx context.Context, userID uint64) (*ClaimResult, error) {
	var ids []uint64
	if err := visible(db.DB(ctx), userID, time.Now()).
		Where("claimed_at IS NULL AND attachments <> '{}'::jsonb").
		Order("id").Limit(maxClaimAll).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	result := &ClaimResult{}
	var (
		total    reward.Bundle
		received []reward.Bundle
	)
	for _, id := range ids {
		bundle, err := claim(ctx, userID, id)
		if err != nil {
			switch err.Error() {
			case AttachmentUnavailable, AttachmentTooLarge:
				result.Skipped++
				continue
			case MailNotFound, MailExpired, AlreadyClaimed:
				// 查询之后被删除、过期或在别处领取了
				continue
			}
			return nil, err
		}
		result.Claimed++
		total.Add(bundle)
		received = append(received, bundle)
	}
	// 合计超出单笔上限时不合并, 逐封列出; 附件已经发放, 不能因为展示失败返回错误
	if normalized, err := total.Normalize(); err == nil {
		received = []reward.Bundle{normalized}
	}
	lines, err := reward.Lines(db.DB(ctx), received...)
	if err != nil {
		return nil, err
	}
	result.Received = []reward.Line{}
	for _, l := range lines {
		result.Received = append(result.Received, l...)
	}
	return result, nil
}

// claim 在一个事务中领取一封邮件的附件, 返回实际发放的内容
func claim(ctx context.Context, userID, mailID uint64) (reward.Bundle, error) {
	var (
		result   *reward.Result
		replayed bool
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		m, err := lockMail(tx, userID, mailID)
		if err != nil {
			return err
		}
		now := time.Now()
		switch {
		case m.Attachments.IsEmpty():
			return errors.New(NoAttachments)
		case m.ClaimedAt != nil:
			return errors.New(AlreadyClaimed)
		case !now.Before(m.ExpiresAt):
			return errors.New(MailExpired)
		}
		bundle, err := reward.Deliverable(tx, m.Attachments, nil)
		if err != nil {
			return err
		}
		bizID := strconv.FormatUint(m.ID, 10)
		if result, replayed, err = reward.Apply(tx, &reward.ApplyRequest{
			UserID:         userID,
			Bundle:         bundle,
			BizType:        BizClaim,
			BizID:          bizID,
			IdempotencyKey: "mail:" + bizID,
			Remark:         m.Title,
		}); err != nil {
			switch err.Error() {
			case item.ItemNotFound, item.ItemDisabled, wallet.CurrencyNotFound, wallet.CurrencyDisabled:
				return errors.New(AttachmentUnavailable)
			case wallet.AmountInvalid, inventory.QuantityInvalid:
				return errors.New(AttachmentTooLarge)
			}
			return err
		}
		return tx.Model(m).UpdateColumns(map[string]any{
			"claimed_at": now,
			"read_at":    gorm.Expr("COALESCE(read_at, ?)", now),
		}).Error
	})
	if err != nil {
		return reward.Bundle{}, err
	}
	if !replayed {
		result.Push(ctx)
	}
	return result.Bundle, nil
}
