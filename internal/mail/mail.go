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

package mail

import (
	"context"
	"errors"
	"fmt"
	netmail "net/mail"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	gomail "github.com/wneessen/go-mail"
)

// ErrDisabled 邮件功能未启用且不在开发环境
var ErrDisabled = errors.New("mail is disabled")

// ErrInvalidAddress 邮箱地址格式不正确
var ErrInvalidAddress = errors.New("invalid email address")

const sendTimeout = 10 * time.Second

// NormalizeAddress 校验并规范化邮箱地址
//
// 只接受纯地址(不带显示名), 长度不超过 255, 统一转为小写。
func NormalizeAddress(address string) (string, error) {
	address = strings.TrimSpace(address)
	if len(address) > 255 {
		return "", ErrInvalidAddress
	}
	parsed, err := netmail.ParseAddress(address)
	if err != nil || parsed.Address != address {
		return "", ErrInvalidAddress
	}
	return strings.ToLower(address), nil
}

// Send 按后台的邮件设置发送纯文本邮件
//
// 未启用邮件时: 开发环境把邮件内容打印到日志便于联调, 其他环境返回 ErrDisabled。
func Send(ctx context.Context, to, subject, body string) error {
	mailConfig, err := settings.Mail.Get(ctx)
	if err != nil {
		return err
	}
	if !mailConfig.Enabled {
		if config.Config.App.Env == "development" {
			logger.InfoF(ctx, "[Mail] disabled, dump mail to %s\nSubject: %s\n%s", to, subject, body)
			return nil
		}
		return ErrDisabled
	}
	return Deliver(ctx, mailConfig, to, subject, body)
}

// Deliver 按给定的 SMTP 配置发送纯文本邮件, 不检查是否启用, 用于发送测试邮件
//
// 发件人名称为空时使用站点名称。
func Deliver(ctx context.Context, mailConfig *settings.MailSettings, to, subject, body string) error {
	fromName := mailConfig.FromName
	if fromName == "" {
		site, err := settings.Site.Get(ctx)
		if err != nil {
			return err
		}
		fromName = site.Name
	}
	msg := gomail.NewMsg()
	if err := msg.FromFormat(fromName, mailConfig.FromAddress); err != nil {
		return fmt.Errorf("set from: %w", err)
	}
	if err := msg.To(to); err != nil {
		return fmt.Errorf("set to: %w", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(gomail.TypeTextPlain, body)

	opts := []gomail.Option{
		gomail.WithPort(mailConfig.Port),
		gomail.WithTimeout(sendTimeout),
	}
	if mailConfig.Port == 465 {
		// 465 端口为隐式 TLS
		opts = append(opts, gomail.WithSSLPort(false))
	} else {
		// 其他端口强制 STARTTLS, 不允许明文发送
		opts = append(opts, gomail.WithTLSPortPolicy(gomail.TLSMandatory))
	}
	if mailConfig.Username != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
			gomail.WithUsername(mailConfig.Username),
			gomail.WithPassword(mailConfig.Password),
		)
	}

	client, err := gomail.NewClient(mailConfig.Host, opts...)
	if err != nil {
		return fmt.Errorf("new mail client: %w", err)
	}
	if err = client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}
