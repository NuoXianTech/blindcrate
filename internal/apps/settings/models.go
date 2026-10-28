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

import "time"

// 设置组的 key
//
// 这里只放运行中可以随时修改、改完立即生效的配置;
// 数据库、Redis、会话、监听地址和各类密钥(app.secret_key 等)始终留在 config.yaml。
const (
	KeySite    = "site"
	KeyAuth    = "auth"
	KeyMail    = "mail"
	KeyEconomy = "economy"
)

// Setting 一个设置组存一行, 内容为该组的 JSON, 未保存过的设置组没有记录, 使用代码中的默认值
type Setting struct {
	Key       string    `gorm:"primaryKey;size:64" json:"key"`
	Value     string    `gorm:"type:jsonb;not null;default:'{}'" json:"value"` // 密钥字段为密文
	UpdatedBy uint64    `gorm:"not null;default:0" json:"updated_by"`
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Setting) TableName() string { return "system_settings" }

// 设置组定义, 新增字段只需在结构体中加一行, 后台表单自动生成
//
// 字段 tag 说明:
//   - json:    字段名, 改名会丢失已保存的值
//   - label:   后台表单中的字段名称, 也用于校验失败的提示
//   - help:    字段说明
//   - binding: 校验规则, 即 go-playground/validator 的规则, 与请求参数的写法一致
//   - input:   字符串的输入方式, 默认 text, 可选 textarea
//   - secret:  "true" 表示密钥, 加密存储, 读取时不返回明文, 只支持字符串
//   - public:  "true" 表示通过公开接口返回给前端
var groups = []*group{
	newGroup(KeySite, "站点", "站点名称、简介和公告等展示信息", func() *SiteSettings {
		return &SiteSettings{Name: "BlindCrate"}
	}),
	newGroup(KeyAuth, "注册与登录", "控制新用户注册和登录方式", func() *AuthSettings {
		return &AuthSettings{RegisterEnabled: true, EmailLoginEnabled: true}
	}),
	newGroup(KeyMail, "邮件", "SMTP 发信配置, 用于发送邮箱验证码等", func() *MailSettings {
		return &MailSettings{Port: 465}
	}),
	newGroup(KeyEconomy, "经济", "货币的产出和回收, 用于控制筹码总量、避免贬值", func() *EconomySettings {
		return &EconomySettings{SellFeeRate: 500, SignupGrant: 5000}
	}),
}

// SiteSettings 站点展示信息
type SiteSettings struct {
	Name         string `json:"name" label:"站点名称" binding:"required,max=64" public:"true"`
	Description  string `json:"description" label:"站点简介" binding:"max=255" input:"textarea" public:"true"`
	LogoURL      string `json:"logo_url" label:"Logo 地址" binding:"omitempty,max=1024,http_url" public:"true"`
	Announcement string `json:"announcement" label:"站点公告" help:"留空则不展示" binding:"max=2000" input:"textarea" public:"true"`
	ICP          string `json:"icp" label:"备案号" binding:"max=64" public:"true"`
	ContactEmail string `json:"contact_email" label:"联系邮箱" binding:"omitempty,max=255,email" public:"true"`
}

// AuthSettings 注册与登录
type AuthSettings struct {
	RegisterEnabled   bool `json:"register_enabled" label:"开放注册" help:"关闭后新用户无法注册, 系统中还没有用户时不受影响" public:"true"`
	EmailLoginEnabled bool `json:"email_login_enabled" label:"允许邮箱登录" help:"开启后已验证邮箱的用户可以用邮箱代替用户名登录" public:"true"`
}

// MailSettings 邮件(SMTP)配置
type MailSettings struct {
	Enabled     bool   `json:"enabled" label:"启用邮件发送" help:"未启用时开发环境会把邮件内容打印到日志, 其他环境无法发送邮件"`
	Host        string `json:"host" label:"SMTP 服务器" binding:"required_if=Enabled true,max=255"`
	Port        int    `json:"port" label:"SMTP 端口" help:"465 为隐式 TLS, 其他端口(如 587)强制 STARTTLS" binding:"min=1,max=65535"`
	Username    string `json:"username" label:"用户名" help:"留空表示不认证" binding:"max=255"`
	Password    string `json:"password" label:"密码" binding:"max=255" secret:"true"`
	FromName    string `json:"from_name" label:"发件人名称" help:"留空则使用站点名称" binding:"max=64"`
	FromAddress string `json:"from_address" label:"发件人邮箱" binding:"required_if=Enabled true,omitempty,max=255,email"`
}

// EconomySettings 经济: 出售手续费和新手资金
type EconomySettings struct {
	SellFeeRate int `json:"sell_fee_rate" label:"出售手续费率" help:"卖出物品时按卖出金额扣除的比例, 单位为基点(万分之一), 500 即 5%, 最高 2000 即 20%" binding:"min=0,max=2000" public:"true"`
	SignupGrant int `json:"signup_grant" label:"新手资金" help:"新用户注册时发放的筹码, 0 表示不发放" binding:"min=0,max=1000000000" public:"true"`
}
