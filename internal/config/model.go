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

package config

import "time"

type configModel struct {
	App      appConfig      `mapstructure:"app"`
	Database databaseConfig `mapstructure:"database"`
	Redis    redisConfig    `mapstructure:"redis"`
	Log      logConfig      `mapstructure:"log"`
	Schedule scheduleConfig `mapstructure:"schedule"`
	Worker   workerConfig   `mapstructure:"worker"`
	WS       wsConfig       `mapstructure:"ws"`
	Storage  storageConfig  `mapstructure:"storage"`
	Otel     otelConfig     `mapstructure:"otel"`
}

// appConfig 应用基本配置
type appConfig struct {
	AppName           string   `mapstructure:"app_name"`
	Env               string   `mapstructure:"env"`
	Addr              string   `mapstructure:"addr"`
	TrustedProxies    []string `mapstructure:"trusted_proxies"` // 可信的反向代理 IP 或网段, 只采信它们转发的 X-Forwarded-For, 为空时客户端 IP 取连接的对端地址
	SessionCookieName string   `mapstructure:"session_cookie_name"`
	SessionSecret     string   `mapstructure:"session_secret"`
	SessionDomain     string   `mapstructure:"session_domain"`
	SessionAge        int      `mapstructure:"session_age"`
	SessionHttpOnly   bool     `mapstructure:"session_http_only"`
	SessionSecure     bool     `mapstructure:"session_secure"`
	APIPrefix         string   `mapstructure:"api_prefix"`
	SecretKey         string   `mapstructure:"secret_key"` // 加密存在数据库中的密钥, 如 SMTP 密码
}

// databaseConfig 数据库配置
type databaseConfig struct {
	Enabled                   bool          `mapstructure:"enabled"`
	Host                      string        `mapstructure:"host"`
	Port                      int           `mapstructure:"port"`
	Username                  string        `mapstructure:"username"`
	Password                  string        `mapstructure:"password"`
	Database                  string        `mapstructure:"database"`
	SSLMode                   string        `mapstructure:"ssl_mode"`
	TimeZone                  string        `mapstructure:"time_zone"`
	MaxIdleConn               int           `mapstructure:"max_idle_conn"`
	MaxOpenConn               int           `mapstructure:"max_open_conn"`
	ConnMaxLifetime           int           `mapstructure:"conn_max_lifetime"`
	LogLevel                  string        `mapstructure:"log_level"`
	SlowThreshold             time.Duration `mapstructure:"slow_threshold"`
	IgnoreRecordNotFoundError bool          `mapstructure:"ignore_record_not_found_error"`
}

// redisConfig Redis配置
type redisConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConn  int    `mapstructure:"min_idle_conn"`
	DialTimeout  int    `mapstructure:"dial_timeout"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// logConfig 日志配置
type logConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
	Compress   bool   `mapstructure:"compress"`
}

// scheduleConfig 定时任务调度配置
type scheduleConfig struct {
	Port          int    `mapstructure:"port"`           // 探针端口
	TimeZone      string `mapstructure:"time_zone"`      // cron 表达式按这个时区解析
	MatchSweep    string `mapstructure:"match_sweep"`    // 巡检到期未推进的对局和排队的 cron 表达式
	StatsCleanup  string `mapstructure:"stats_cleanup"`  // 清理过期的按天统计的 cron 表达式, 留空时不清理
	SeasonSettle  string `mapstructure:"season_settle"`  // 给已结束的赛季发放赛季奖励的 cron 表达式, 留空时不发放
	UploadCleanup string `mapstructure:"upload_cleanup"` // 清理没有被引用的上传图片的 cron 表达式, 留空时不清理

	// Crons 按任务类型覆盖登记时的默认 cron 表达式, 如 mailbox:cleanup, 为空字符串时不定时投递, 见 task.Register
	Crons map[string]string `mapstructure:"crons"`
}

// workerConfig 任务处理配置
type workerConfig struct {
	Port        int `mapstructure:"port"`        // 探针端口
	Concurrency int `mapstructure:"concurrency"` // 同时处理的任务数
}

// wsConfig WebSocket 配置
type wsConfig struct {
	OriginPatterns []string `mapstructure:"origin_patterns"` // 允许发起连接的页面来源, 与 API 同域名的页面总是允许
}

// storageConfig 上传文件的存储, 不配置时存在本地磁盘
type storageConfig struct {
	Driver    string `mapstructure:"driver"`     // 存储方式, 目前只有 local: 本地磁盘
	LocalDir  string `mapstructure:"local_dir"`  // local 存储的目录
	PublicURL string `mapstructure:"public_url"` // 文件的访问地址前缀, 如 /uploads 或 https://cdn.example.com/uploads; 为站内路径时由 API 提供访问
}

// otelConfig OpenTelemetry 配置
type otelConfig struct {
	SamplingRate float64 `mapstructure:"sampling_rate"`
}
