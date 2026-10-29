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

package settingsadmin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/mail"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListSettingsRequest 获取设置组请求参数
type ListSettingsRequest struct {
	Category string `json:"category" form:"category" binding:"omitempty,oneof=system game"` // 为空时返回全部
}

// ListSettingsResponseData 设置组和分类
type ListSettingsResponseData struct {
	Categories []settings.Category  `json:"categories"`
	Groups     []settings.GroupView `json:"groups"`
}

// ListSettings
// @Tags admin
// @Summary 获取设置组
// @Description 包含表单描述和当前值, 密钥字段不返回明文; 可以按分类筛选
// @Produce json
// @Param request query ListSettingsRequest true "request query"
// @Success 200 {object} Response{data=ListSettingsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings [get]
func ListSettings(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListSettingsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	groups, err := settings.QueryGroups(ctx, req.Category)
	if err != nil {
		logger.ErrorF(ctx, "[ListSettings] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListSettingsResponseData{Categories: settings.Categories, Groups: groups}})
}

// GetSetting
// @Tags admin
// @Summary 获取一个设置组
// @Produce json
// @Param key path string true "setting group key"
// @Success 200 {object} Response{data=settings.GroupView}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings/{key} [get]
func GetSetting(c *gin.Context) {
	ctx := c.Request.Context()
	group, err := settings.QueryGroup(ctx, c.Param("key"))
	if err != nil {
		if err.Error() == settings.GroupNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[GetSetting] query %s failed: %v", c.Param("key"), err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: group})
}

// UpdateSettingRequest 保存设置组请求体
type UpdateSettingRequest struct {
	Version      *int64          `json:"version" binding:"required,min=0"`              // 读取时的版本, 已被别人修改时返回 409
	Value        json.RawMessage `json:"value" binding:"required" swaggertype:"object"` // 设置组的字段, 只需提交改动的字段, 没有提交的保持不变, 密钥字段留空表示不修改
	ClearSecrets []string        `json:"clear_secrets" binding:"max=32"`                // 需要清空的密钥字段
}

// UpdateSetting
// @Tags admin
// @Summary 保存一个设置组
// @Description 保存后立即生效, 并生成一个新版本
// @Accept json
// @Produce json
// @Param key path string true "setting group key"
// @Param request body UpdateSettingRequest true "request body"
// @Success 200 {object} Response{data=settings.GroupView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings/{key} [put]
func UpdateSetting(c *gin.Context) {
	ctx := c.Request.Context()
	var req UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	group, changes, err := settings.Update(ctx, operator.ID, c.Param("key"), *req.Version, req.Value, req.ClearSecrets)
	if err != nil {
		c.JSON(settingErrorStatus(err), Response{ErrorMsg: err.Error()})
		return
	}
	recordSettingChange(c, operator.ID, audit.ActionSettingsUpdate, group, changes, nil)
	c.JSON(http.StatusOK, Response{Data: group})
}

// ListSettingRevisionsRequest 获取设置组版本请求参数
type ListSettingRevisionsRequest struct {
	Current int `json:"current" form:"current" binding:"min=1"`
	Size    int `json:"size" form:"size" binding:"min=1,max=100"`
}

// ListSettingRevisionsResponseData 设置组版本分页结果
type ListSettingRevisionsResponseData struct {
	Total     int64                 `json:"total"`
	Revisions []SettingRevisionItem `json:"revisions"`
}

// ListSettingRevisions
// @Tags admin
// @Summary 获取设置组的历史版本
// @Description 按版本从新到旧排列, 不含快照内容
// @Produce json
// @Param key path string true "setting group key"
// @Param request query ListSettingRevisionsRequest true "request query"
// @Success 200 {object} Response{data=ListSettingRevisionsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings/{key}/revisions [get]
func ListSettingRevisions(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListSettingRevisionsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	key := c.Param("key")
	if !settings.HasGroup(key) {
		c.JSON(http.StatusNotFound, Response{ErrorMsg: settings.GroupNotFound})
		return
	}
	total, revisions, err := QuerySettingRevisions(ctx, key, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListSettingRevisions] query %s failed: %v", key, err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListSettingRevisionsResponseData{Total: total, Revisions: revisions}})
}

// GetSettingRevision
// @Tags admin
// @Summary 获取设置组某个版本的快照
// @Description 密钥字段不返回明文, 只返回是否已设置
// @Produce json
// @Param key path string true "setting group key"
// @Param version path int true "version"
// @Success 200 {object} Response{data=settings.RevisionView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings/{key}/revisions/{version} [get]
func GetSettingRevision(c *gin.Context) {
	ctx := c.Request.Context()
	version, err := strconv.ParseInt(c.Param("version"), 10, 64)
	if err != nil || version < 0 {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	revision, err := settings.QueryRevision(ctx, c.Param("key"), version)
	if err != nil {
		switch err.Error() {
		case settings.GroupNotFound, settings.RevisionNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[GetSettingRevision] query %s@%d failed: %v", c.Param("key"), version, err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	c.JSON(http.StatusOK, Response{Data: revision})
}

// RollbackSettingRequest 回滚设置组请求体
type RollbackSettingRequest struct {
	Version *int64 `json:"version" binding:"required,min=0"` // 当前版本, 已被别人修改时返回 409
	Target  *int64 `json:"target" binding:"required,min=0"`  // 回滚到的版本
}

// RollbackSetting
// @Tags admin
// @Summary 回滚设置组
// @Description 用旧版本的快照生成一个新版本, 立即生效; 快照中的密钥无法解密时不能回滚
// @Accept json
// @Produce json
// @Param key path string true "setting group key"
// @Param request body RollbackSettingRequest true "request body"
// @Success 200 {object} Response{data=settings.GroupView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings/{key}/rollback [post]
func RollbackSetting(c *gin.Context) {
	ctx := c.Request.Context()
	var req RollbackSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	group, changes, err := settings.Rollback(ctx, operator.ID, c.Param("key"), *req.Version, *req.Target)
	if err != nil {
		c.JSON(settingErrorStatus(err), Response{ErrorMsg: err.Error()})
		return
	}
	recordSettingChange(c, operator.ID, audit.ActionSettingsRollback, group, changes, map[string]interface{}{"target": *req.Target})
	c.JSON(http.StatusOK, Response{Data: group})
}

// ResetSettingRequest 恢复默认值请求体
type ResetSettingRequest struct {
	Version *int64 `json:"version" binding:"required,min=0"` // 当前版本, 已被别人修改时返回 409
}

// ResetSetting
// @Tags admin
// @Summary 恢复设置组的默认值
// @Description 恢复为代码中的默认值并生成一个新版本, 密钥字段会被清空
// @Accept json
// @Produce json
// @Param key path string true "setting group key"
// @Param request body ResetSettingRequest true "request body"
// @Success 200 {object} Response{data=settings.GroupView}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/settings/{key}/reset [post]
func ResetSetting(c *gin.Context) {
	ctx := c.Request.Context()
	var req ResetSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	group, changes, err := settings.Reset(ctx, operator.ID, c.Param("key"), *req.Version)
	if err != nil {
		c.JSON(settingErrorStatus(err), Response{ErrorMsg: err.Error()})
		return
	}
	recordSettingChange(c, operator.ID, audit.ActionSettingsReset, group, changes, nil)
	c.JSON(http.StatusOK, Response{Data: group})
}

// settingErrorStatus 保存设置组的错误对应的状态码, 错误提示都可以直接展示
func settingErrorStatus(err error) int {
	switch err.Error() {
	case settings.GroupNotFound, settings.RevisionNotFound:
		return http.StatusNotFound
	case settings.VersionConflict:
		return http.StatusConflict
	case settings.SaveFailed:
		return http.StatusInternalServerError
	default:
		// 其余均为设置内容校验失败
		return http.StatusBadRequest
	}
}

// recordSettingChange 记录设置组的修改, 没有任何修改时不记录
func recordSettingChange(c *gin.Context, operatorID uint64, action string, group *settings.GroupView, changes, extra map[string]interface{}) {
	if len(changes) == 0 {
		return
	}
	detail := map[string]interface{}{
		"key":     group.Key,
		"version": group.Version,
		"changes": changes,
	}
	for k, v := range extra {
		detail[k] = v
	}
	audit.Record(c, operatorID, audit.ScopeAdmin, action, audit.TargetSetting, 0, detail)
	logger.InfoF(c.Request.Context(), "[%s] %d changed %s to version %d", action, operatorID, group.Key, group.Version)
}

// TestMailRequest 测试邮件请求体
type TestMailRequest struct {
	To string `json:"to" binding:"required"` // 收件人
}

// TestMail
// @Tags admin
// @Summary 发送测试邮件
// @Description 使用已保存的邮件设置发送, 未启用邮件发送时也可以测试; 发送失败时返回 SMTP 的错误信息
// @Accept json
// @Produce json
// @Param request body TestMailRequest true "request body"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Failure 502 {object} Response
// @Router /api/v1/admin/settings/mail/test [post]
func TestMail(c *gin.Context) {
	ctx := c.Request.Context()
	var req TestMailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	to, mailConfig, err := prepareTestMail(ctx, &req)
	if err != nil {
		switch err.Error() {
		case auth.EmailInvalid, MailTestIncomplete:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[TestMail] query settings failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	// 发送失败时把 SMTP 返回的错误交给管理员排查
	sendErr := mail.Deliver(ctx, mailConfig, to, "测试邮件", "这是一封测试邮件, 收到说明邮件设置可用。")
	detail := map[string]interface{}{"to": to, "host": mailConfig.Host, "port": mailConfig.Port, "success": sendErr == nil}
	if sendErr != nil {
		detail["error"] = sendErr.Error()
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionMailTest, audit.TargetSetting, 0, detail)
	if sendErr != nil {
		logger.WarnF(ctx, "[TestMail] %d send to %s failed: %v", operator.ID, to, sendErr)
		c.JSON(http.StatusBadGateway, Response{ErrorMsg: fmt.Sprintf("%s: %s", MailTestFailed, sendErr.Error())})
		return
	}
	c.JSON(http.StatusOK, Response{})
}
