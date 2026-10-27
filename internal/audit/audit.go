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
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// 操作范围
const (
	ScopeUser  = "user"
	ScopeAdmin = "admin"
)

// 操作对象类型
const (
	TargetUser    = "user"
	TargetSetting = "setting" // target_id 固定为 0, 设置组见 detail.key
	TargetItem    = "item"
)

// 操作类型, 格式为 "模块.动作"
const (
	ActionLogin           = "auth.login"
	ActionLoginFailed     = "auth.login_failed"
	ActionRegister        = "auth.register"
	ActionLogout          = "auth.logout"
	ActionChangePassword  = "auth.change_password"
	ActionVerifyEmail     = "auth.verify_email"
	ActionUpdateProfile   = "auth.update_profile"
	ActionUserUpdate      = "admin.user_update"
	ActionUserBan         = "admin.user_ban"
	ActionUserUnban       = "admin.user_unban"
	ActionUserDisable     = "admin.user_disable"
	ActionUserEnable      = "admin.user_enable"
	ActionWalletAdjust    = "admin.wallet_adjust"
	ActionItemCreate      = "admin.item_create"
	ActionItemUpdate      = "admin.item_update"
	ActionItemDelete      = "admin.item_delete"
	ActionInventoryAdjust = "admin.inventory_adjust"
	ActionSettingsUpdate  = "admin.settings_update"
	ActionMailTest        = "admin.mail_test"
)

type Log struct {
	ID         uint64    `json:"id" gorm:"primaryKey"`
	ActorID    uint64    `json:"actor_id" gorm:"not null;default:0;index:idx_audit_logs_actor_id_created_at,priority:1"`
	Scope      string    `json:"scope" gorm:"size:16;not null;default:''"`
	Action     string    `json:"action" gorm:"size:64;not null;default:'';index:idx_audit_logs_action_created_at,priority:1"`
	TargetType string    `json:"target_type" gorm:"size:32;not null;default:'';index:idx_audit_logs_target,priority:1"`
	TargetID   uint64    `json:"target_id" gorm:"not null;default:0;index:idx_audit_logs_target,priority:2"`
	Detail     string    `json:"detail" gorm:"type:jsonb;not null;default:'{}'"`
	IP         string    `json:"ip" gorm:"size:64;not null;default:''"`
	UserAgent  string    `json:"user_agent" gorm:"size:512;not null;default:''"`
	CreatedAt  time.Time `json:"created_at" gorm:"not null;default:now();autoCreateTime;index;index:idx_audit_logs_actor_id_created_at,priority:2;index:idx_audit_logs_action_created_at,priority:2;index:idx_audit_logs_target,priority:3"`
}

func (l *Log) TableName() string {
	return "audit_logs"
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

func record(ctx context.Context, log *Log, detail map[string]interface{}) {
	log.Detail = "{}"
	if len(detail) > 0 {
		raw, err := json.Marshal(detail)
		if err != nil {
			logger.ErrorF(ctx, "[Audit] marshal detail failed: %s", err.Error())
		} else {
			log.Detail = string(raw)
		}
	}
	if err := db.DB(ctx).Create(log).Error; err != nil {
		logger.ErrorF(ctx, "[Audit] record %s by %d failed: %s", log.Action, log.ActorID, err.Error())
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
