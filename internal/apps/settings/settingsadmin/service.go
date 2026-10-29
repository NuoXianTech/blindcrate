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

package settingsadmin

import (
	"context"
	"errors"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/mail"
)

// SettingRevisionItem 设置组的一个版本, 不含快照内容
type SettingRevisionItem struct {
	Version           int64                  `json:"version"`
	Action            string                 `json:"action"`
	FromVersion       int64                  `json:"from_version"`
	Changes           map[string]interface{} `gorm:"serializer:json" json:"changes"`
	UpdatedBy         uint64                 `json:"updated_by"`
	UpdatedByUsername string                 `json:"updated_by_username"` // 系统操作或用户已不存在时为空
	CreatedAt         time.Time              `json:"created_at"`
}

// QuerySettingRevisions 获取设置组的历史版本, 按版本从新到旧
func QuerySettingRevisions(ctx context.Context, key string, req *ListSettingRevisionsRequest) (int64, []SettingRevisionItem, error) {
	offset := (req.Current - 1) * req.Size
	query := db.DB(ctx).Model(&settings.Revision{}).Where("setting_revisions.key = ?", key)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	revisions := []SettingRevisionItem{}
	if err := query.
		Select("setting_revisions.version, setting_revisions.action, setting_revisions.from_version, setting_revisions.changes, " +
			"setting_revisions.updated_by, setting_revisions.created_at, COALESCE(users.username, '') AS updated_by_username").
		Joins("LEFT JOIN users ON users.id = setting_revisions.updated_by").
		Order("setting_revisions.version DESC").Offset(offset).Limit(req.Size).
		Scan(&revisions).Error; err != nil {
		return 0, nil, err
	}

	return total, revisions, nil
}

// prepareTestMail 校验测试邮件的收件人, 并取出已保存的邮件设置
//
// 只使用已保存的设置, 不接受草稿: 否则改掉 SMTP 服务器后沿用已保存的密码, 就能把密码发到任意服务器。
func prepareTestMail(ctx context.Context, req *TestMailRequest) (string, *settings.MailSettings, error) {
	to, err := mail.NormalizeAddress(req.To)
	if err != nil {
		return "", nil, errors.New(auth.EmailInvalid)
	}
	mailConfig, err := settings.Mail.Get(ctx)
	if err != nil {
		return "", nil, err
	}
	if mailConfig.Host == "" || mailConfig.FromAddress == "" {
		return "", nil, errors.New(MailTestIncomplete)
	}
	return to, mailConfig, nil
}
