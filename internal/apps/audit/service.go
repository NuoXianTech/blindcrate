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

// Package audit 记录用户和管理员的操作日志(audit_logs 表)
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// actionPattern 操作类型的格式, 即 "模块.动作"
var actionPattern = regexp.MustCompile(`^[a-z0-9_]+\.[a-z0-9_]+$`)

// Action 一种操作类型
type Action struct {
	Code  string `json:"code"`
	Label string `json:"label"` // 后台日志中展示的名称, 如 "新建物品"
}

// actions 已登记的操作类型, 按登记顺序, 只在启动前写入
var actions []Action

// RegisterAction 登记一种操作类型, 需要在启动前调用; 格式不正确或重复时 panic
func RegisterAction(code, label string) {
	if len(code) > maxActionLength || !actionPattern.MatchString(code) || label == "" {
		panic(fmt.Sprintf("audit: invalid action %q", code))
	}
	if registered(code) {
		panic(fmt.Sprintf("audit: action %s already registered", code))
	}
	actions = append(actions, Action{Code: code, Label: label})
}

// Actions 全部操作类型, 按登记顺序
func Actions() []Action {
	return slices.Clone(actions)
}

// registered 操作类型是否已登记
func registered(code string) bool {
	return slices.ContainsFunc(actions, func(a Action) bool { return a.Code == code })
}

// Record 写入一条操作日志, 失败只记录错误日志, 不影响业务
//
// detail 为附加信息(会序列化为 JSON), 不要放入密码、验证码等敏感数据。
func Record(c *gin.Context, actorID uint64, scope, action, targetType string, targetID uint64, detail map[string]interface{}) {
	record(c.Request.Context(), &Log{
		ActorID:    actorID,
		Scope:      scope,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		IP:         c.ClientIP(),
		UserAgent:  truncate(c.Request.UserAgent(), 512),
	}, detail)
}

// record 序列化 detail 后写库; 未登记的操作类型照常写入, 只记录警告
func record(ctx context.Context, log *Log, detail map[string]interface{}) {
	if !registered(log.Action) {
		logger.WarnF(ctx, "[Audit] action %s not registered", log.Action)
	}
	log.Detail = "{}"
	if len(detail) > 0 {
		raw, err := json.Marshal(detail)
		if err != nil {
			logger.ErrorF(ctx, "[Audit] marshal detail failed: %v", err)
		} else {
			log.Detail = string(raw)
		}
	}
	if err := db.DB(ctx).Create(log).Error; err != nil {
		logger.ErrorF(ctx, "[Audit] record %s by %d failed: %v", log.Action, log.ActorID, err)
	}
}

// truncate 按字符截断, 避免切断多字节字符
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	s = s[:max]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}
