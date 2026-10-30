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

func init() {
	// 早期模块的操作类型, 新模块在自己的 constants.go 中定义并在 init 中登记
	for _, a := range []Action{
		{Code: ActionLogin, Label: "登录"},
		{Code: ActionLoginFailed, Label: "登录失败"},
		{Code: ActionRegister, Label: "注册"},
		{Code: ActionLogout, Label: "退出登录"},
		{Code: ActionChangePassword, Label: "修改密码"},
		{Code: ActionVerifyEmail, Label: "绑定邮箱"},
		{Code: ActionUpdateProfile, Label: "修改资料"},
		{Code: ActionUserUpdate, Label: "修改用户资料"},
		{Code: ActionUserBan, Label: "封禁用户"},
		{Code: ActionUserUnban, Label: "解除封禁"},
		{Code: ActionUserDisable, Label: "停用用户"},
		{Code: ActionUserEnable, Label: "启用用户"},
		{Code: ActionWalletAdjust, Label: "调整余额"},
		{Code: ActionItemCreate, Label: "新建物品"},
		{Code: ActionItemUpdate, Label: "修改物品"},
		{Code: ActionItemDelete, Label: "删除物品"},
		{Code: ActionInventoryAdjust, Label: "调整背包"},
		{Code: ActionLootPoolCreate, Label: "新建掉落池"},
		{Code: ActionLootPoolUpdate, Label: "修改掉落池"},
		{Code: ActionLootPoolDelete, Label: "删除掉落池"},
		{Code: ActionLootPoolEntriesUpdate, Label: "修改掉落池条目"},
		{Code: ActionVenueCreate, Label: "新建地图"},
		{Code: ActionVenueUpdate, Label: "修改地图"},
		{Code: ActionVenueDelete, Label: "删除地图"},
		{Code: ActionMatchCancel, Label: "取消对局"},
		{Code: ActionLeaderboardCreate, Label: "新建榜单"},
		{Code: ActionLeaderboardUpdate, Label: "修改榜单"},
		{Code: ActionLeaderboardDelete, Label: "删除榜单"},
		{Code: ActionQualityCreate, Label: "新建品质"},
		{Code: ActionQualityUpdate, Label: "修改品质"},
		{Code: ActionQualityDelete, Label: "删除品质"},
		{Code: ActionCurrencyCreate, Label: "新建货币"},
		{Code: ActionCurrencyUpdate, Label: "修改货币"},
		{Code: ActionSeasonCreate, Label: "新建赛季"},
		{Code: ActionSeasonUpdate, Label: "修改赛季"},
		{Code: ActionSeasonDelete, Label: "删除赛季"},
		{Code: ActionSeasonEnd, Label: "提前结束赛季"},
		{Code: ActionRankedTierCreate, Label: "新建段位"},
		{Code: ActionRankedTierUpdate, Label: "修改段位"},
		{Code: ActionRankedTierDelete, Label: "删除段位"},
		{Code: ActionSuspectReview, Label: "处理可疑账号"},
		{Code: ActionShopItemCreate, Label: "新建商品"},
		{Code: ActionShopItemUpdate, Label: "修改商品"},
		{Code: ActionShopItemDelete, Label: "删除商品"},
		{Code: ActionShopOrderRefund, Label: "订单退款"},
		{Code: ActionSettingsUpdate, Label: "修改系统设置"},
		{Code: ActionSettingsRollback, Label: "回滚系统设置"},
		{Code: ActionSettingsReset, Label: "恢复系统设置默认值"},
		{Code: ActionMailTest, Label: "发送测试邮件"},
		{Code: ActionPackExport, Label: "导出配置包"},
		{Code: ActionPackImport, Label: "导入配置包"},
	} {
		RegisterAction(a.Code, a.Label)
	}
}

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
