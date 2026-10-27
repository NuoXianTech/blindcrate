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
	Otel     otelConfig     `mapstructure:"otel"`
}

// appConfig 应用基本配置
type appConfig struct {
	AppName           string `mapstructure:"app_name"`
	Env               string `mapstructure:"env"`
	Addr              string `mapstructure:"addr"`
	SessionCookieName string `mapstructure:"session_cookie_name"`
	SessionSecret     string `mapstructure:"session_secret"`
	SessionDomain     string `mapstructure:"session_domain"`
	SessionAge        int    `mapstructure:"session_age"`
	SessionHttpOnly   bool   `mapstructure:"session_http_only"`
	SessionSecure     bool   `mapstructure:"session_secure"`
	APIPrefix         string `mapstructure:"api_prefix"`
	SecretKey         string `mapstructure:"secret_key"` // 加密存在数据库中的密钥, 如 SMTP 密码
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
	Port     int    `mapstructure:"port"`      // 探针端口
	TimeZone string `mapstructure:"time_zone"` // cron 表达式按这个时区解析
}

// workerConfig 任务处理配置
type workerConfig struct {
	Port        int `mapstructure:"port"`        // 探针端口
	Concurrency int `mapstructure:"concurrency"` // 同时处理的任务数
}

// otelConfig OpenTelemetry 配置
type otelConfig struct {
	SamplingRate float64 `mapstructure:"sampling_rate"`
}
