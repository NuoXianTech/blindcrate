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
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

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

// record 序列化 detail 后写库
func record(ctx context.Context, log *Log, detail map[string]interface{}) {
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
