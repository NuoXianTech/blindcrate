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

// Package storage 上传文件的存储: 按 key 写入、删除文件, 并生成访问地址
//
// key 由调用方生成, 如 item/ab/abcd….png, 只能包含小写字母、数字、下划线、横线和目录分隔符, 最后是扩展名;
// 业务表只保存访问地址 URL(key), 更换存储或域名时批量改写地址前缀。
// 存储方式和访问地址前缀在 config.yaml 的 storage 中配置, 不配置时存在本地磁盘的 ./data/uploads, 由 API 在 /uploads 提供访问。
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// DriverLocal 本地磁盘
const DriverLocal = "local"

// Storage 文件存储
type Storage interface {
	// Put 写入文件, key 已存在时覆盖
	Put(ctx context.Context, key string, r io.Reader, contentType string) error
	// Delete 删除文件, 文件不存在时不报错
	Delete(ctx context.Context, key string) error
	// URL 文件的访问地址
	URL(key string) string
}

// keyPattern 文件的 key, 只有最后一段可以带扩展名, 不会出现 . 和 .. 目录
var keyPattern = regexp.MustCompile(`^[a-z0-9_-]+(/[a-z0-9_-]+)*\.[a-z0-9]+$`)

var (
	once     sync.Once
	instance Storage
	initErr  error
)

// Default 按配置创建的存储, 只创建一次
func Default() (Storage, error) {
	once.Do(func() {
		cfg := config.Config.Storage
		switch cfg.Driver {
		case DriverLocal:
			if cfg.LocalDir == "" {
				initErr = errors.New("storage: storage.local_dir is required")
				return
			}
			instance = &local{dir: cfg.LocalDir, publicURL: publicURL()}
		default:
			initErr = fmt.Errorf("storage: unknown driver %q", cfg.Driver)
		}
	})
	return instance, initErr
}

// CheckKey key 格式是否正确
func CheckKey(key string) error {
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("storage: invalid key %q", key)
	}
	return nil
}

// publicURL 访问地址前缀, 去掉末尾的 /
func publicURL() string {
	return strings.TrimRight(config.Config.Storage.PublicURL, "/")
}

// Local 本地存储且访问前缀为站内路径时, 返回需要由 API 提供访问的目录和路径前缀
func Local() (dir, prefix string, ok bool) {
	cfg := config.Config.Storage
	prefix = publicURL()
	if cfg.Driver != DriverLocal || !strings.HasPrefix(prefix, "/") || strings.HasPrefix(prefix, "//") {
		return "", "", false
	}
	return cfg.LocalDir, prefix, true
}

// ValidURL 图片地址是否可以保存: http(s) 链接, 或访问前缀为站内路径时本站存储的文件地址
func ValidURL(s string) bool {
	if prefix := publicURL(); strings.HasPrefix(prefix, "/") && strings.HasPrefix(s, prefix+"/") {
		return keyPattern.MatchString(strings.TrimPrefix(s, prefix+"/"))
	}
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func init() {
	// 请求参数用 binding:"image_url" 校验图片地址, 见 ValidURL
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("image_url", func(fl validator.FieldLevel) bool { return ValidURL(fl.Field().String()) }); err != nil {
			panic(err)
		}
	}
}
