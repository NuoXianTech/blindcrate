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

package auth

import (
	"net/http"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// BasicUserInfo 当前用户的基本信息
type BasicUserInfo struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"` // 只有验证通过的邮箱才会写入, 未绑定为空
	AvatarURL string `json:"avatar_url"`
	IsAdmin   bool   `json:"is_admin"`
}

// NewBasicUserInfo 用户的基本信息
func NewBasicUserInfo(user *User) BasicUserInfo {
	return BasicUserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
		IsAdmin:   user.IsAdmin,
	}
}

// LoginRequest 登录请求体
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 用户名或已验证的邮箱
	Password string `json:"password" binding:"required"`
}

// Login
// @Tags auth
// @Summary 用户名或邮箱登录
// @Description 支持用户名或已验证的邮箱 + 密码登录, 邮箱登录需在设置中开启
// @Accept json
// @Produce json
// @Param request body LoginRequest true "request body"
// @Success 200 {object} reply.Response{data=BasicUserInfo}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 429 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Param X-Captcha-Token header string false "人机验证令牌, 该场景开启人机验证时必填"
// @Router /api/v1/auth/login [post]
func Login(c *gin.Context) {
	ctx := c.Request.Context()
	var req LoginRequest
	if !reply.JSON(c, &req) {
		return
	}
	user, err := doLogin(ctx, req.Username, req.Password)
	if err != nil {
		// 密码错误和账号被封禁记入操作日志, 用于排查撞库
		if reason := err.Error(); reason == InvalidCredentials || reason == BannedAccount {
			audit.Record(c, 0, audit.ScopeUser, ActionLoginFailed, "", 0, map[string]interface{}{"identifier": req.Username, "reason": reason})
		}
		reply.Fail(c, err, "[Login] login failed")
		return
	}
	session := sessions.Default(c)
	setSessionUser(session, user)
	if err := session.Save(); err != nil {
		reply.Fail(c, err, "[Login] save session failed")
		return
	}
	if err := trackSession(ctx, user.ID, session.ID()); err != nil {
		logger.ErrorF(ctx, "[Login] track session failed: %v", err)
	}
	LogForAudit(ctx, user, c)
	audit.Record(c, user.ID, audit.ScopeUser, ActionLogin, TargetUser, user.ID, nil)
	logger.InfoF(ctx, "[Login] %d %s", user.ID, user.Username)
	reply.OK(c, NewBasicUserInfo(user))
}

// RegisterRequest 注册请求体
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email"` // 开启注册时验证邮箱后必填, 未开启时忽略
	Code     string `json:"code"`  // 发到邮箱的验证码, 见 /api/v1/auth/register/code
}

// SendRegisterCodeRequest 发送注册验证码请求体
type SendRegisterCodeRequest struct {
	Email string `json:"email" binding:"required"`
}

// SendRegisterCode
// @Tags auth
// @Summary 向注册用的邮箱发送验证码
// @Description 只在开启注册时验证邮箱后可用
// @Accept json
// @Produce json
// @Param request body SendRegisterCodeRequest true "request body"
// @Success 200 {object} reply.Response
// @Failure 400 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 429 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Failure 503 {object} reply.Response
// @Param X-Captcha-Token header string false "人机验证令牌, 该场景开启人机验证时必填"
// @Router /api/v1/auth/register/code [post]
func SendRegisterCode(c *gin.Context) {
	ctx := c.Request.Context()
	var req SendRegisterCodeRequest
	if !reply.JSON(c, &req) {
		return
	}
	if err := doSendRegisterCode(ctx, req.Email); err != nil {
		reply.Fail(c, err, "[SendRegisterCode] send failed")
		return
	}
	reply.OK(c, nil)
}

// Register
// @Tags auth
// @Summary 注册新用户
// @Description 注册成功后直接登录; 系统中第一个注册的用户自动成为管理员, 且不受注册开关和邮箱验证限制; 开启注册时验证邮箱后需要邮箱和验证码
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "request body"
// @Success 201 {object} reply.Response{data=BasicUserInfo}
// @Failure 400 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 429 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Param X-Captcha-Token header string false "人机验证令牌, 该场景开启人机验证时必填"
// @Router /api/v1/auth/register [post]
func Register(c *gin.Context) {
	ctx := c.Request.Context()
	var req RegisterRequest
	if !reply.JSON(c, &req) {
		return
	}
	user, err := doRegister(ctx, req.Username, req.Password, req.Email, req.Code)
	if err != nil {
		reply.Fail(c, err, "[Register] register failed")
		return
	}
	session := sessions.Default(c)
	setSessionUser(session, user)
	if err := session.Save(); err != nil {
		reply.Fail(c, err, "[Register] save session failed")
		return
	}
	if err := trackSession(ctx, user.ID, session.ID()); err != nil {
		logger.ErrorF(ctx, "[Register] track session failed: %v", err)
	}
	LogForAudit(ctx, user, c)
	changes := map[string]interface{}{"is_admin": user.IsAdmin}
	if user.Email != "" {
		changes["email"] = user.Email
	}
	audit.Record(c, user.ID, audit.ScopeUser, ActionRegister, TargetUser, user.ID, changes)
	logger.InfoF(ctx, "[Register] %d %s", user.ID, user.Username)
	c.JSON(http.StatusCreated, reply.Response{Data: NewBasicUserInfo(user)})
}

// Logout
// @Tags auth
// @Summary 退出登录
// @Produce json
// @Success 200 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/auth/logout [post]
func Logout(c *gin.Context) {
	ctx := c.Request.Context()
	session := sessions.Default(c)
	userID, sessionID := GetUserIDFromSession(session), session.ID()
	session.Clear()
	if err := session.Save(); err != nil {
		reply.Fail(c, err, "[Logout] save session failed")
		return
	}
	if userID > 0 {
		audit.Record(c, userID, audit.ScopeUser, ActionLogout, TargetUser, userID, nil)
	}
	if userID > 0 && sessionID != "" {
		if err := untrackSession(ctx, userID, sessionID); err != nil {
			logger.ErrorF(ctx, "[Logout] untrack session failed: %v", err)
		}
	}
	reply.OK(c, nil)
}

// ChangePasswordRequest 修改密码请求体
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"` // 未设置过密码时可不传
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword
// @Tags auth
// @Summary 修改当前用户的密码
// @Description 未设置过密码(仅第三方登录)的用户无需提供原密码; 修改后当前设备保持登录, 其他设备下线
// @Accept json
// @Produce json
// @Param request body ChangePasswordRequest true "request body"
// @Success 200 {object} reply.Response
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 429 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/auth/password [put]
func ChangePassword(c *gin.Context) {
	ctx := c.Request.Context()
	var req ChangePasswordRequest
	if !reply.JSON(c, &req) {
		return
	}
	user := CurrentUser(c)
	if err := doChangePassword(ctx, user, req.OldPassword, req.NewPassword); err != nil {
		reply.Fail(c, err, "[ChangePassword] change password failed")
		return
	}
	// 保留当前设备, 其他设备下线
	if err := RevokeUserSessions(ctx, user.ID, sessions.Default(c).ID()); err != nil {
		logger.ErrorF(ctx, "[ChangePassword] revoke sessions failed: %v", err)
	}
	audit.Record(c, user.ID, audit.ScopeUser, ActionChangePassword, TargetUser, user.ID, nil)
	logger.InfoF(ctx, "[ChangePassword] %d %s", user.ID, user.Username)
	reply.OK(c, nil)
}

// SendEmailCodeRequest 发送邮箱验证码请求体
type SendEmailCodeRequest struct {
	Email string `json:"email" binding:"required"`
}

// SendEmailCode
// @Tags auth
// @Summary 向待绑定的邮箱发送验证码
// @Description 邮箱在验证通过前不会写入账号
// @Accept json
// @Produce json
// @Param request body SendEmailCodeRequest true "request body"
// @Success 200 {object} reply.Response
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 429 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Failure 503 {object} reply.Response
// @Router /api/v1/auth/email/code [post]
func SendEmailCode(c *gin.Context) {
	ctx := c.Request.Context()
	var req SendEmailCodeRequest
	if !reply.JSON(c, &req) {
		return
	}
	user := CurrentUser(c)
	if err := doSendEmailCode(ctx, user.ID, req.Email); err != nil {
		reply.Fail(c, err, "[SendEmailCode] send failed")
		return
	}
	reply.OK(c, nil)
}

// VerifyEmailRequest 校验邮箱验证码请求体
type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

// VerifyEmail
// @Tags auth
// @Summary 校验邮箱验证码并绑定邮箱
// @Accept json
// @Produce json
// @Param request body VerifyEmailRequest true "request body"
// @Success 200 {object} reply.Response
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/auth/email/verify [post]
func VerifyEmail(c *gin.Context) {
	ctx := c.Request.Context()
	var req VerifyEmailRequest
	if !reply.JSON(c, &req) {
		return
	}
	user := CurrentUser(c)
	if err := doVerifyEmail(ctx, user.ID, req.Email, req.Code); err != nil {
		reply.Fail(c, err, "[VerifyEmail] verify failed")
		return
	}
	audit.Record(c, user.ID, audit.ScopeUser, ActionVerifyEmail, TargetUser, user.ID, map[string]interface{}{"email": req.Email})
	logger.InfoF(ctx, "[VerifyEmail] %d %s", user.ID, user.Username)
	reply.OK(c, nil)
}

// UserInfo
// @Tags auth
// @Summary 获取当前用户信息
// @Produce json
// @Success 200 {object} reply.Response{data=BasicUserInfo}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/auth/user-info [get]
func UserInfo(c *gin.Context) {
	ctx := c.Request.Context()
	userID := GetUserIDFromContext(c)
	var user User
	tx := db.DB(ctx).Where("id = ?", userID).First(&user)
	if tx.Error != nil {
		reply.Fail(c, tx.Error, "[UserInfo] query user failed")
		return
	}
	reply.OK(c, NewBasicUserInfo(&user))
}

// UpdateUserInfoRequest 修改资料请求体
type UpdateUserInfoRequest struct {
	Nickname  *string `json:"nickname"`   // 不传表示不修改
	AvatarURL *string `json:"avatar_url"` // 不传表示不修改, 传空字符串表示清除头像
}

// UpdateUserInfo
// @Tags auth
// @Summary 修改当前用户的昵称和头像
// @Accept json
// @Produce json
// @Param request body UpdateUserInfoRequest true "request body"
// @Success 200 {object} reply.Response{data=BasicUserInfo}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/auth/user-info [patch]
func UpdateUserInfo(c *gin.Context) {
	ctx := c.Request.Context()
	var req UpdateUserInfoRequest
	if !reply.JSON(c, &req) {
		return
	}
	user := CurrentUser(c)
	updated, err := UpdateUserProfile(ctx, user.ID, req.Nickname, req.AvatarURL)
	if err != nil {
		reply.Fail(c, err, "[UpdateUserInfo] update failed")
		return
	}
	audit.Record(c, updated.ID, audit.ScopeUser, ActionUpdateProfile, TargetUser, updated.ID, ProfileChanges(req.Nickname, req.AvatarURL))
	logger.InfoF(ctx, "[UpdateUserInfo] %d %s", updated.ID, updated.Username)
	reply.OK(c, NewBasicUserInfo(updated))
}
