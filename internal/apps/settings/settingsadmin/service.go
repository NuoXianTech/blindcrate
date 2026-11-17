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

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/fault"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/mail"
)

// prepareTestMail 校验测试邮件的收件人, 并取出已保存的邮件设置
//
// 只使用已保存的设置, 不接受草稿: 否则改掉 SMTP 服务器后沿用已保存的密码, 就能把密码发到任意服务器。
func prepareTestMail(ctx context.Context, req *TestMailRequest) (string, *settings.MailSettings, error) {
	to, err := mail.NormalizeAddress(req.To)
	if err != nil {
		return "", nil, fault.Invalid(auth.EmailInvalid)
	}
	mailConfig, err := settings.Mail.Get(ctx)
	if err != nil {
		return "", nil, err
	}
	if mailConfig.Host == "" || mailConfig.FromAddress == "" {
		return "", nil, fault.Invalid(MailTestIncomplete)
	}
	return to, mailConfig, nil
}
