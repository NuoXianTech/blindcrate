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

package storage

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// local 本地磁盘存储, 文件放在 dir 下, 目录按 key 自动创建
type local struct {
	dir       string
	publicURL string
}

// Put 先写临时文件再改名, 读到一半的请求不会看到写了一半的文件
func (s *local) Put(_ context.Context, key string, r io.Reader, _ string) error {
	if err := CheckKey(key); err != nil {
		return err
	}
	path := filepath.Join(s.dir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".upload-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func (s *local) Open(_ context.Context, key string) (io.ReadCloser, error) {
	if err := CheckKey(key); err != nil {
		return nil, err
	}
	return os.Open(filepath.Join(s.dir, filepath.FromSlash(key)))
}

func (s *local) Delete(_ context.Context, key string) error {
	if err := CheckKey(key); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(s.dir, filepath.FromSlash(key))); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (s *local) URL(key string) string {
	return s.publicURL + "/" + key
}

// LocalHandler 提供本地存储中文件的访问, 路由参数 key 为文件的 key, 见 Local
//
// 文件内容按 key 不变, 可以长期缓存; 不存在的文件返回 404 且不缓存, 不列出目录。
func LocalHandler(dir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// key 只能是本站生成的格式, 拼出的路径不会越出目录
		key := strings.TrimPrefix(c.Param("key"), "/")
		if !keyPattern.MatchString(key) {
			c.Status(http.StatusNotFound)
			return
		}
		path := filepath.Join(dir, filepath.FromSlash(key))
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.File(path)
	}
}
