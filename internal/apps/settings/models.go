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

// 分类, 后台按分类分页展示设置组
const (
	CategorySystem = "system" // 系统: 站点、注册登录、邮件、数据统计等
	CategoryGame   = "game"   // 游戏: 经济、对局等玩法参数
)

// 版本的来源
const (
	RevisionUpdate   = "update"   // 保存
	RevisionRollback = "rollback" // 回滚到某个旧版本
	RevisionReset    = "reset"    // 恢复默认值
	RevisionBaseline = "baseline" // 启用版本记录前已保存的内容, 第一次保存时补记为版本 0
)

// 设置组的 key
//
// 这里只放运行中可以随时修改、改完立即生效的配置;
// 数据库、Redis、会话、监听地址和各类密钥(app.secret_key 等)始终留在 config.yaml。
const (
	KeySite    = "site"
	KeyAuth    = "auth"
	KeyMail    = "mail"
	KeyEconomy = "economy"
	KeyStats   = "stats"
	KeyRanked  = "ranked"
)

// Setting 一个设置组存一行, 内容为该组的 JSON, 未保存过的设置组没有记录, 使用代码中的默认值
type Setting struct {
	Key       string    `gorm:"primaryKey;size:64" json:"key"`
	Value     string    `gorm:"type:jsonb;not null;default:'{}'" json:"value"` // 密钥字段为密文
	Version   int64     `gorm:"not null;default:0" json:"version"`             // 每次保存加 1, 保存时校验, 防止覆盖别人的修改
	UpdatedBy uint64    `gorm:"not null;default:0" json:"updated_by"`
	CreatedAt time.Time `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();autoUpdateTime" json:"updated_at"`
}

// TableName 自定义表名
func (Setting) TableName() string { return "settings" }

// Revision 设置组的一个版本, 每次保存写入一条, 用于查看历史和回滚
type Revision struct {
	ID          uint64                 `gorm:"primaryKey" json:"id"`
	Key         string                 `gorm:"size:64;not null;uniqueIndex:uniq_setting_revisions_key_version,priority:1" json:"key"`
	Version     int64                  `gorm:"not null;uniqueIndex:uniq_setting_revisions_key_version,priority:2" json:"version"`
	Value       string                 `gorm:"type:jsonb;not null;default:'{}'" json:"-"`                       // 保存后的完整内容, 密钥字段为密文
	Changes     map[string]interface{} `gorm:"type:jsonb;serializer:json;not null;default:'{}'" json:"changes"` // 相对上一个版本改动的字段, 格式同操作日志
	Action      string                 `gorm:"size:16;not null" json:"action"`
	FromVersion int64                  `gorm:"not null;default:0" json:"from_version"` // 回滚时为回滚到的版本, 其他为 0
	UpdatedBy   uint64                 `gorm:"not null;default:0" json:"updated_by"`
	CreatedAt   time.Time              `gorm:"not null;default:now();autoCreateTime" json:"created_at"`
}

// TableName 自定义表名
func (Revision) TableName() string { return "setting_revisions" }

// 全部设置组, 新增设置组时在这里注册
//
// 字段 tag 说明:
//   - json:    字段名, 改名会丢失已保存的值
//   - label:   后台表单中的字段名称, 也用于校验失败的提示
//   - help:    字段说明
//   - binding: 校验规则, 即 go-playground/validator 的规则, 与请求参数的写法一致
//   - input:   字符串的输入方式, 默认 text, 可选 textarea
//   - options: 字符串的可选值, 如 "a:选项 A,b:选项 B", 后台显示为下拉框, 保存时校验
//   - unit:    整数的单位, 可选 bp(基点, 后台按百分比输入)、second、millisecond
//   - secret:  "true" 表示密钥, 加密存储, 读取时不返回明文, 只支持字符串
//   - public:  "true" 表示通过公开接口返回给前端
//
// 字段类型支持 string、bool、int / int32 / int64 和它们的切片(整数列表)。
var (
	Site = Register(Group[SiteSettings]{
		Key:         KeySite,
		Label:       "站点",
		Description: "站点名称、简介和公告等展示信息",
		Category:    CategorySystem,
		Sort:        10,
		Defaults:    func() *SiteSettings { return &SiteSettings{Name: "BlindCrate"} },
	})
	Auth = Register(Group[AuthSettings]{
		Key:         KeyAuth,
		Label:       "注册与登录",
		Description: "控制新用户注册和登录方式",
		Category:    CategorySystem,
		Sort:        20,
		Defaults:    func() *AuthSettings { return &AuthSettings{RegisterEnabled: true, EmailLoginEnabled: true} },
	})
	Mail = Register(Group[MailSettings]{
		Key:         KeyMail,
		Label:       "邮件",
		Description: "SMTP 发信配置, 用于发送邮箱验证码等",
		Category:    CategorySystem,
		Sort:        30,
		Defaults:    func() *MailSettings { return &MailSettings{Port: 465} },
	})
	Stats = Register(Group[StatsSettings]{
		Key:         KeyStats,
		Label:       "数据统计",
		Description: "玩家统计数据的保存时间",
		Category:    CategorySystem,
		Sort:        40,
		Defaults:    func() *StatsSettings { return &StatsSettings{DayRetention: 60} },
	})
	Economy = Register(Group[EconomySettings]{
		Key:         KeyEconomy,
		Label:       "经济",
		Description: "货币的产出和回收, 用于控制货币总量、避免贬值",
		Category:    CategoryGame,
		Sort:        10,
		Defaults: func() *EconomySettings {
			// 基础货币默认为种子数据中的筹码 chip
			return &EconomySettings{BaseCurrency: "chip", SellFeeRate: 500, SignupGrant: 5000, PveDailyProfitCap: 5000}
		},
	})
	Ranked = Register(Group[RankedSettings]{
		Key:         KeyRanked,
		Label:       "排位",
		Description: "段位分的初始分和每局按名次的加减分",
		Category:    CategoryGame,
		Sort:        20,
		Defaults:    func() *RankedSettings { return &RankedSettings{BaseRating: 1000, RankDeltas: []int{30, 10, -10, -30}} },
	})
)

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

// StatsSettings 数据统计
type StatsSettings struct {
	DayRetention int `json:"day_retention" label:"按天统计保留天数" help:"含今天, 更早的按天统计每天定时清理, 累计统计不受影响; 个人数据页展示最近 7 天, 因此不能少于 7 天" binding:"min=7,max=3650"`
}

// EconomySettings 经济: 基础货币、出售手续费、新手资金和人机模式的收益上限
type EconomySettings struct {
	BaseCurrency      string `json:"base_currency" label:"基础货币" help:"新手资金和已删除物品的回收价使用的货币, 也是新建物品和地图时默认选中的货币" binding:"required,max=32" ref:"currency" public:"true"`
	SellFeeRate       int    `json:"sell_fee_rate" label:"出售手续费率" help:"卖出物品时按卖出金额扣除的比例, 最高 20%" binding:"min=0,max=2000" unit:"bp" public:"true"`
	SignupGrant       int    `json:"signup_grant" label:"新手资金" help:"新用户注册时发放的基础货币, 0 表示不发放" binding:"min=0,max=1000000000" public:"true"`
	PveDailyProfitCap int    `json:"pve_daily_profit_cap" label:"人机每日收益上限" help:"玩家当天在人机模式的净收益达到上限后, 当天剩下的人机对局变为练习局, 不扣入场费也不发藏品; 按各地图的出价货币分别计算; 0 表示不限制, 地图可以单独设置更低的上限" binding:"min=0,max=1000000000" public:"true"`
}

// RankedSettings 排位: 段位分的初始分和加减分
type RankedSettings struct {
	BaseRating int   `json:"base_rating" label:"基础分" help:"第一次打排位时的段位分, 新赛季的初始分也向它靠拢, 见赛季的段位分继承比例" binding:"min=0,max=100000"`
	RankDeltas []int `json:"rank_deltas" label:"按名次加减分" help:"依次为第 1 名、第 2 名……的加减分, 负数为扣分, 扣到 0 为止; 没有配置的名次不加不减; 流拍不计分" binding:"min=1,max=8,dive,min=-1000,max=1000"`
}
