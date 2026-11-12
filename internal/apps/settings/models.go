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
	RevisionImport   = "import"   // 导入配置包
)

// anyVersion 保存时不检查版本, 用于导入配置包
const anyVersion int64 = -1

// 设置组的 key
//
// 这里只放运行中可以随时修改、改完立即生效的配置;
// 数据库、Redis、会话、监听地址和各类密钥(app.secret_key 等)始终留在 config.yaml。
const (
	KeySite    = "site"
	KeyAuth    = "auth"
	KeyCaptcha = "captcha"
	KeyMail    = "mail"
	KeyEconomy = "economy"
	KeyStats   = "stats"
	KeyUpload  = "upload"
	KeyRanked  = "ranked"
	KeyBalance = "balance"
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
		Defaults: func() *AuthSettings {
			return &AuthSettings{RegisterEnabled: true, EmailLoginEnabled: true, EmailDomainMode: EmailDomainOff}
		},
	})
	Captcha = Register(Group[CaptchaSettings]{
		Key:         KeyCaptcha,
		Label:       "人机验证",
		Description: "Cloudflare Turnstile 人机验证, 可以按场景单独开关; 密钥在 Cloudflare 控制台的 Turnstile 中创建",
		Category:    CategorySystem,
		Sort:        25,
		Defaults:    func() *CaptchaSettings { return &CaptchaSettings{} },
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
	Upload = Register(Group[UploadSettings]{
		Key:         KeyUpload,
		Label:       "图片上传",
		Description: "上传图片的大小、格式和未使用图片的保留时间",
		Category:    CategorySystem,
		Sort:        50,
		Defaults: func() *UploadSettings {
			return &UploadSettings{MaxSizeKB: 5120, AllowPNG: true, AllowJPEG: true, AllowWebP: true, AvatarEnabled: true, RetentionDays: 7}
		},
	})
	Economy = Register(Group[EconomySettings]{
		Key:         KeyEconomy,
		Label:       "经济",
		Description: "货币的产出和回收, 用于控制货币总量、避免贬值",
		Category:    CategoryGame,
		Sort:        10,
		Defaults: func() *EconomySettings {
			// 基础货币默认为种子数据中的筹码 chip
			return &EconomySettings{BaseCurrency: "chip", SellFeeRate: 200, SignupGrant: 5000, PveDailyProfitCap: 5000}
		},
	})
	Balance = Register(Group[BalanceSettings]{
		Key:         KeyBalance,
		Label:       "玩法平衡",
		Description: "地图保存时按仓库的价值分布推导拍卖金额, 再模拟对局求出领先折让和入场费, 检查是否落在安全区间; 修改后需要在地图管理中重新推导全部地图, 进行中的对局不受影响",
		Category:    CategoryGame,
		Sort:        15,
		Defaults: func() *BalanceSettings {
			return &BalanceSettings{
				SpeedFloorEnabled:   true,
				LeadDiscountEnabled: true,
				TargetSeatEV:        -200,
				StartPriceRate:      400,
				CarryOfMean:         20000,
				CarryOfP90:          15000,
				BotCarryRate:        12000,
				SpeedFloorRate:      7500,
				MaxLeadDiscount:     3000,
				PreviewRate:         3000,
				CarryCoverage:       3000,
				BotCarryShare:       6000,
				BotCarryFloor:       7000,
				HypeLotDecay:        3000,
				HumanPlanShare:      5000,
				HumanBudgetPace:     25000,
				MinRounds:           4,
				MaxRounds:           10,
				MinGridSide:         4,
				MaxGridSide:         10,
				MaxTopShare:         6000,
				PriceValueTolerance: 300,
				MinBidCV:            1200,
				MinSpeedWin:         800,
				MaxSpeedWin:         4500,
				MinAppraiseBias:     8000,
				MaxAppraiseBias:     13000,
				MinSeatEV:           -400,
				MinRankedWinRate:    2500,
				MinPveWinRate:       3000,
				HumanSigmaStart:     4200,
				HumanSigmaStep:      600,
				HumanSigmaFloor:     1400,
				HumanJumpChance:     2500,
				HumanStair:          5500,
				HumanCapFactor:      9700,
			}
		},
	})
	Ranked = Register(Group[RankedSettings]{
		Key:         KeyRanked,
		Label:       "排位",
		Description: "段位分的初始分、每局按名次的加减分、挂机禁赛、防串通和可疑名单",
		Category:    CategoryGame,
		Sort:        20,
		Defaults: func() *RankedSettings {
			return &RankedSettings{
				BaseRating:   1000,
				RankDeltas:   []int{30, 10, -10, -30},
				AfkThreshold: 2,
				AfkBans:      []int{300, 1800, 7200},
				SeparateIP:   true,
				AvoidRecent:  3,
				WatchDays:    7,
				WatchMatches: 10,
				WatchLowRate: 5000,
				WatchLowWins: 3,
			}
		},
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

// 邮箱后缀的限制方式
const (
	EmailDomainOff   = "off"   // 不限制
	EmailDomainAllow = "allow" // 白名单: 只接受列表中的后缀
	EmailDomainDeny  = "deny"  // 黑名单: 不接受列表中的后缀
)

// AuthSettings 注册与登录
type AuthSettings struct {
	RegisterEnabled   bool `json:"register_enabled" label:"开放注册" help:"关闭后新用户无法注册, 系统中还没有用户时不受影响" public:"true"`
	EmailLoginEnabled bool `json:"email_login_enabled" label:"允许邮箱登录" help:"开启后已验证邮箱的用户可以用邮箱代替用户名登录" public:"true"`

	RegisterEmailRequired bool   `json:"register_email_required" label:"注册时验证邮箱" help:"开启后注册时需要填写邮箱并通过验证码验证; 需要先启用邮件发送, 否则新用户无法注册; 已注册的用户和系统中的第一个用户不受影响" public:"true"`
	BlockEmailAlias       bool   `json:"block_email_alias" label:"屏蔽别名邮箱" help:"开启后注册和绑定邮箱时不接受带 + 的别名地址(如 name+tag@example.com), 已绑定的邮箱不受影响" public:"true"`
	EmailDomainMode       string `json:"email_domain_mode" label:"邮箱后缀限制" help:"白名单只接受列表中的后缀, 黑名单不接受列表中的后缀; 注册和绑定邮箱时生效, 已绑定的邮箱不受影响" options:"off:不限制,allow:白名单,deny:黑名单" binding:"oneof=off allow deny"`
	EmailDomains          string `json:"email_domains" label:"邮箱后缀列表" help:"每行一个后缀, 如 qq.com; 子域名一并匹配, 填 example.com 时 mail.example.com 也算; 白名单为空时所有邮箱都不能通过" binding:"max=4000" input:"textarea"`
}

// CaptchaSettings 人机验证(Cloudflare Turnstile), 验证见 internal/captcha
type CaptchaSettings struct {
	SiteKey   string `json:"site_key" label:"站点密钥" help:"Turnstile 的 Site Key; 站点密钥和私钥都填写后下面的开关才生效" binding:"max=255" public:"true"`
	SecretKey string `json:"secret_key" label:"私钥" help:"Turnstile 的 Secret Key; 填错或服务器连不上 Cloudflare 时不拦截请求, 只记录错误日志" binding:"max=255" secret:"true"`

	LoginEnabled    bool `json:"login_enabled" label:"登录" help:"开启后登录时需要通过人机验证" public:"true"`
	RegisterEnabled bool `json:"register_enabled" label:"注册" help:"开启后注册和发送注册验证码时需要通过人机验证" public:"true"`
	CheckinEnabled  bool `json:"checkin_enabled" label:"签到" help:"开启后签到时需要通过人机验证" public:"true"`
	ShopEnabled     bool `json:"shop_enabled" label:"商店购买" help:"开启后在商店购买商品时需要通过人机验证" public:"true"`
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
	DayRetention int `json:"day_retention" label:"按天统计保留天数" help:"含今天, 更早的按天统计每天定时清理, 按周统计保留到包含最早一天的那一周, 累计统计不受影响; 个人数据页展示最近 7 天, 因此不能少于 7 天" binding:"min=7,max=3650"`
}

// UploadSettings 图片上传: 头像、物品图片、地图封面和商品图片
type UploadSettings struct {
	MaxSizeKB     int  `json:"max_size_kb" label:"图片大小上限" help:"原图的大小上限, 单位 KB; 保存时会按用途缩小尺寸并重新编码" binding:"min=64,max=20480" public:"true"`
	AllowPNG      bool `json:"allow_png" label:"允许 PNG" public:"true"`
	AllowJPEG     bool `json:"allow_jpeg" label:"允许 JPEG" public:"true"`
	AllowWebP     bool `json:"allow_webp" label:"允许 WebP" help:"WebP 图片保存时转为 PNG 或 JPEG" public:"true"`
	AvatarEnabled bool `json:"avatar_enabled" label:"允许玩家上传头像" help:"关闭后玩家只能填写头像链接, 管理员不受影响" public:"true"`
	RetentionDays int  `json:"retention_days" label:"未使用图片保留天数" help:"上传后超过这么多天仍没有被头像、物品、地图、商品或设置使用的图片每天定时删除" binding:"min=1,max=3650"`
}

// EconomySettings 经济: 基础货币、出售手续费、新手资金和人机模式的收益上限
type EconomySettings struct {
	BaseCurrency      string `json:"base_currency" label:"基础货币" help:"新手资金和已删除物品的回收价使用的货币, 也是新建物品和地图时默认选中的货币" binding:"required,max=32" ref:"currency" public:"true"`
	SellFeeRate       int    `json:"sell_fee_rate" label:"出售手续费率" help:"卖出物品时按卖出金额扣除的比例, 最高 20%" binding:"min=0,max=2000" unit:"bp" public:"true"`
	SignupGrant       int    `json:"signup_grant" label:"新手资金" help:"新用户注册时发放的基础货币, 0 表示不发放" binding:"min=0,max=1000000000" public:"true"`
	PveDailyProfitCap int    `json:"pve_daily_profit_cap" label:"人机每日收益上限" help:"玩家当天在人机模式的净收益达到上限后, 当天剩下的人机对局变为练习局, 不扣入场费也不发藏品; 按各地图的出价货币分别计算; 0 表示不限制, 地图可以单独设置更低的上限" binding:"min=0,max=1000000000" public:"true"`
}

// BalanceSettings 玩法平衡: 推导拍卖金额的系数、对局验算的闸门和模拟的真人, 以及几条规则的开关
//
// 金额都是仓库期望价值(或 P90)的倍数, 换了藏品或掉落也不用重新调; 推导结果保存在地图上, 开局时随地图快照, 修改后重新推导地图才生效。
// 默认值来自玩法改进的验算, 比例和系数都是基点。
type BalanceSettings struct {
	SpeedFloorEnabled   bool `json:"speed_floor_enabled" label:"速胜下限" help:"开启后速胜还要求第一名的出价不低于速胜下限; 只靠倍率挡不住速胜, 第一名常常天然高出第二名一截"`
	LeadDiscountEnabled bool `json:"lead_discount_enabled" label:"领先折让" help:"开启后成交价 = 第二名的出价 + 领先部分 × (1 - 折让), 折让按地图模拟求出, 让成交价平均落在仓库价值扣掉出售手续费; 关闭时按最高价成交"`

	TargetSeatEV    int `json:"target_seat_ev" label:"入场费: 真人盈亏目标" help:"入场费按地图模拟求出, 让 4 个真人时每个真人席位平均每局的盈亏等于期望价值的这个比例, 负数为亏损, 即系统回收的部分; 入场费不影响出价, 只平移盈亏, 流拍时退还; 排位补人机和人机模式共用这个入场费, 有人机时真人更容易低价拍下, 盈亏通常高于目标; 不能低于真人盈亏下限" binding:"min=-2000,max=0,gtefield=MinSeatEV" unit:"bp"`
	StartPriceRate  int `json:"start_price_rate" label:"起拍价比例" help:"起拍价占期望价值的比例; 过高时大家都从起拍价附近出价, 出价拉不开" binding:"min=1,max=2000" unit:"bp"`
	CarryOfMean     int `json:"carry_of_mean" label:"带入上限 ÷ 期望价值" help:"带入资金上限不低于期望价值的这么多倍, 否则高价值的仓库出价被封顶, 成交价系统性偏低; 低于 2 倍时验算多半不过" binding:"min=10000,max=100000" unit:"bp"`
	CarryOfP90      int `json:"carry_of_p90" label:"带入上限 ÷ P90" help:"带入资金上限也不低于价值 P90 的这么多倍, 两项取大" binding:"min=0,max=100000" unit:"bp"`
	BotCarryRate    int `json:"bot_carry_rate" label:"人机带入资金比例" help:"人机的带入资金占期望价值的比例, 不超过带入资金上限; 人机的钱不来自真人钱包, 带得越多越容易高价抢走大仓库, 真人拍不到, 系统也替它买单" binding:"min=1000,max=100000" unit:"bp"`
	SpeedFloorRate  int `json:"speed_floor_rate" label:"速胜下限比例" help:"速胜下限占期望价值的比例; 过低挡不住速胜, 过高基本只能打到最后一轮" binding:"min=0,max=30000" unit:"bp"`
	MaxLeadDiscount int `json:"max_lead_discount" label:"领先折让上限" help:"折让越大成交价越低, 全折让(二价)时赢家白捡、系统净发放" binding:"min=0,max=9000" unit:"bp"`
	CarryCoverage   int `json:"carry_coverage" label:"拍场: 带入覆盖率" help:"一局多个拍品的地图, 带入资金上限 = 每局拍品数的期望 × 单个拍品的期望价值 × 覆盖率; 地图可以单独设置; 越低越要挑着拍, 太低时后面的拍品没人出得起" binding:"min=1000,max=20000" unit:"bp"`
	BotCarryShare   int `json:"bot_carry_share" label:"拍场: 人机带入比例" help:"一局多个拍品的地图, 人机带入资金的上限占带入资金上限的比例" binding:"min=1000,max=10000" unit:"bp"`
	BotCarryFloor   int `json:"bot_carry_floor" label:"拍场: 人机带入下限" help:"每个人机的带入资金在人机带入资金 × [下限, 100%] 内随机, 真人不能从人机花掉的钱推出它还剩多少" binding:"min=1000,max=10000" unit:"bp"`
	HypeLotDecay    int `json:"hype_lot_decay" label:"拍场: 热度回落" help:"每个拍品开场时热度向 50 回落的比例, 避免一个热门拍品把后面全部点燃" binding:"min=0,max=10000" unit:"bp"`
	PreviewRate     int `json:"preview_rate" label:"首轮扫描比例" help:"第 1 轮每件藏品按这个概率扫出轮廓(只有占的格子, 没有品质和名称), 至少扫出一件; 其余的直到开箱都不公开, 件数也不公开, 只能靠道具看到; 0 表示不扫" binding:"min=0,max=10000" unit:"bp"`

	MinRounds           int `json:"min_rounds" label:"轮数下限" help:"轮数太少时出价来不及拉开, 速胜和追价都没有余地" binding:"min=1,max=10,ltefield=MaxRounds"`
	MaxRounds           int `json:"max_rounds" label:"轮数上限" binding:"min=1,max=10"`
	MinGridSide         int `json:"min_grid_side" label:"仓库边长下限" help:"3×3 的仓库多半摆不下, 一件藏品就占了大半价值" binding:"min=1,max=16,ltefield=MaxGridSide"`
	MaxGridSide         int `json:"max_grid_side" label:"仓库边长上限" help:"更大的仓库没有验算过" binding:"min=1,max=16"`
	MaxTopShare         int `json:"max_top_share" label:"最贵一件的占比上限" help:"超过时一件藏品决定了整个仓库, 速胜和盈亏都被它绑架" binding:"min=0,max=10000" unit:"bp"`
	PriceValueTolerance int `json:"price_value_tolerance" label:"成交价与价值之比的容差" help:"4 个真人时成交价 ÷ 仓库价值与目标(1 - 出售手续费率)的差距" binding:"min=0,max=10000" unit:"bp"`
	MinBidCV            int `json:"min_bid_cv" label:"出价离散度下限" help:"低于它说明大家的出价收敛到同一个价, 暗拍失去意义" binding:"min=0,max=10000" unit:"bp"`
	MinSpeedWin         int `json:"min_speed_win" label:"速胜率下限" help:"低于它基本只能打到最后一轮" binding:"min=0,max=10000,ltefield=MaxSpeedWin" unit:"bp"`
	MaxSpeedWin         int `json:"max_speed_win" label:"速胜率上限" help:"高于它说明速胜倍率和下限没有起作用" binding:"min=0,max=10000" unit:"bp"`
	MinAppraiseBias     int `json:"min_appraise_bias" label:"人机估值偏差下限" help:"人机估值 ÷ 仓库价值, 低于它说明人机系统性低估" binding:"min=0,max=100000,ltefield=MaxAppraiseBias" unit:"bp"`
	MaxAppraiseBias     int `json:"max_appraise_bias" label:"人机估值偏差上限" help:"高于它说明人机系统性高估, 会买贵" binding:"min=0,max=100000" unit:"bp"`
	MinSeatEV           int `json:"min_seat_ev" label:"真人盈亏下限" help:"每个真人席位平均每局的盈亏不低于期望价值的这个比例, 负数为亏损; 人机模式只能练习时不看" binding:"min=-10000,max=10000" unit:"bp"`
	MinRankedWinRate    int `json:"min_ranked_win_rate" label:"排位补人机时真人的胜率下限" help:"2 个真人和 2 个人机时, 每个真人席位拍下仓库的比例(多拍品时为拍场第 1 名的比例)不低于它, 0 表示不检查; 平分为 25%" binding:"min=0,max=10000" unit:"bp"`
	MinPveWinRate       int `json:"min_pve_win_rate" label:"人机模式真人的胜率下限" help:"1 个真人和 3 个人机时, 真人拍下仓库的比例(多拍品时为拍场第 1 名的比例)不低于它, 0 表示不检查; 平分为 25%" binding:"min=0,max=10000" unit:"bp"`

	HumanSigmaStart int `json:"human_sigma_start" label:"模拟真人: 首轮估值误差" help:"验算时模拟的真人知道仓库价值但看不准, 第 1 轮的估值误差占价值的比例, 之后每轮减少, 不低于误差下限" binding:"min=0,max=10000" unit:"bp"`
	HumanSigmaStep  int `json:"human_sigma_step" label:"模拟真人: 每轮减少的误差" help:"按 5 轮的地图计, 轮数不同时按比例缩放, 轮数越多每轮减得越少" binding:"min=0,max=10000" unit:"bp"`
	HumanSigmaFloor int `json:"human_sigma_floor" label:"模拟真人: 误差下限" binding:"min=0,max=10000" unit:"bp"`
	HumanJumpChance int `json:"human_jump_chance" label:"模拟真人: 首轮抢拍概率" help:"第 1 轮按估值的 95% 出价的概率, 否则按起拍价试探" binding:"min=0,max=10000" unit:"bp"`
	HumanStair      int `json:"human_stair" label:"模拟真人: 第 2 轮出价系数" help:"第 2 轮按估值的这个比例出价, 之后每轮加 10%, 不超过 96%" binding:"min=0,max=10000" unit:"bp"`
	HumanCapFactor  int `json:"human_cap_factor" label:"模拟真人: 出价上限系数" help:"出价不超过估值的这个比例" binding:"min=1,max=20000" unit:"bp"`
	HumanBudgetPace int `json:"human_budget_pace" label:"模拟真人: 预算系数" help:"规划预算的模拟真人这个拍品最多花剩余资金 × 估值 ÷ (估值 + 后面拍品的期望) × 预算系数; 4 个人分拍品, 每人平均只拿到 1/4, 系数大于 1 才不会把出价压到价值以下" binding:"min=1000,max=40000" unit:"bp"`
	HumanPlanShare  int `json:"human_plan_share" label:"模拟真人: 规划预算的比例" help:"多拍品的拍场中, 这么多的模拟真人按剩余资金和后面拍品的期望分配预算, 其余只看眼前的拍品, 剩多少钱都敢出" binding:"min=0,max=10000" unit:"bp"`
}

// RankedSettings 排位: 段位分的初始分和加减分, 挂机禁赛, 防串通和可疑名单
type RankedSettings struct {
	BaseRating   int   `json:"base_rating" label:"基础分" help:"第一次打排位时的段位分, 新赛季的初始分也向它靠拢, 见赛季的段位分继承比例" binding:"min=0,max=100000"`
	RankDeltas   []int `json:"rank_deltas" label:"按名次加减分" help:"依次为第 1 名、第 2 名……的加减分, 负数为扣分, 扣到 0 为止; 没有配置的名次不加不减; 流拍不计分" binding:"min=1,max=8,dive,min=-1000,max=1000"`
	AfkThreshold int   `json:"afk_threshold" label:"挂机禁赛次数" help:"一局排位中所有轮次都没有提交出价(包括不加价)记为一次挂机, 24 小时内达到这个次数时禁止排位一段时间; 0 表示不禁赛" binding:"min=0,max=100"`
	AfkBans      []int `json:"afk_bans" label:"挂机禁赛时长" help:"依次为达到次数时、再挂机一次时……的禁赛时长, 超出的次数使用最后一个时长" binding:"min=1,max=10,dive,min=1,max=604800" unit:"second"`
	SeparateIP   bool  `json:"separate_ip" label:"同一 IP 不同局" help:"开启后同一 IP 的账号不会排进同一局, 同一网络下(如同一个宿舍)的玩家也无法同局; 同一设备(浏览器)的账号始终不会排进同一局"`
	AvoidRecent  int   `json:"avoid_recent" label:"避开最近同局" help:"最近这么多局排位中一起打过的人优先不再排到一起, 最早排队的人等到人机补位时间后放宽; 0 表示不避开" binding:"min=0,max=20"`
	WatchDays    int   `json:"watch_days" label:"可疑名单统计天数" help:"可疑名单统计最近这么多天的排位" binding:"min=1,max=90"`
	WatchMatches int   `json:"watch_matches" label:"可疑同局次数" help:"统计天数内两个账号一起打的排位达到这个局数时进入可疑名单" binding:"min=2,max=10000"`
	WatchLowRate int   `json:"watch_low_rate" label:"低价成交比例" help:"成交价低于仓库价值的这个比例时算作低价成交" binding:"min=0,max=10000" unit:"bp"`
	WatchLowWins int   `json:"watch_low_wins" label:"可疑低价成交次数" help:"统计天数内两个账号同局时其中一人低价成交达到这个次数时进入可疑名单" binding:"min=1,max=10000"`
}
