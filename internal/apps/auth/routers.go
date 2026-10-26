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
	"strconv"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 用户名或已验证的邮箱
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// Login godoc
// @Tags auth
// @Param request body LoginRequest true "request body"
// @Produce json
// @Success 200 {object} LoginResponse
// @Failure 400 {object} LoginResponse
// @Failure 401 {object} LoginResponse
// @Failure 403 {object} LoginResponse
// @Failure 500 {object} LoginResponse
// @Router /api/v1/auth/login [post]
func Login(c *gin.Context) {
	// init req
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LoginResponse{ErrorMsg: InvalidParams})
		return
	}
	// do login
	user, err := doLogin(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if err.Error() == InvalidCredentials {
			c.JSON(http.StatusUnauthorized, LoginResponse{ErrorMsg: InvalidCredentials})
			return
		}
		if err.Error() == BannedAccount {
			c.JSON(http.StatusForbidden, LoginResponse{ErrorMsg: BannedAccount})
			return
		}
		c.JSON(http.StatusInternalServerError, LoginResponse{ErrorMsg: err.Error()})
		return
	}
	// bind to session
	session := sessions.Default(c)
	session.Set(UserIDKey, user.ID)
	session.Set(UserNameKey, user.Username)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, LoginResponse{ErrorMsg: err.Error()})
		return
	}
	if err := trackSession(c.Request.Context(), user.ID, session.ID()); err != nil {
		logger.ErrorF(c.Request.Context(), "[Login] track session failed: %s", err.Error())
	}
	// log
	LogForAudit(c.Request.Context(), user, c)
	logger.InfoF(c.Request.Context(), "[Login] %d %s", user.ID, user.Username)
	// response
	c.JSON(
		http.StatusOK,
		LoginResponse{
			Data: BasicUserInfo{
				ID:        user.ID,
				Username:  user.Username,
				Nickname:  user.Nickname,
				Email:     user.Email,
				AvatarURL: user.AvatarURL,
				IsAdmin:   user.IsAdmin,
			},
		},
	)
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// Register godoc
// @Tags auth
// @Param request body RegisterRequest true "request body"
// @Produce json
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} RegisterResponse
// @Failure 409 {object} RegisterResponse
// @Failure 500 {object} RegisterResponse
// @Router /api/v1/auth/register [post]
func Register(c *gin.Context) {
	// init req
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, RegisterResponse{ErrorMsg: InvalidParams})
		return
	}
	// do register
	user, err := doRegister(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		switch err.Error() {
		case UsernameInvalid, PasswordInvalid:
			c.JSON(http.StatusBadRequest, RegisterResponse{ErrorMsg: err.Error()})
		case UsernameTaken:
			c.JSON(http.StatusConflict, RegisterResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[Register] register failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, RegisterResponse{ErrorMsg: InternalError})
		}
		return
	}
	// bind to session
	session := sessions.Default(c)
	session.Set(UserIDKey, user.ID)
	session.Set(UserNameKey, user.Username)
	if err := session.Save(); err != nil {
		logger.ErrorF(c.Request.Context(), "[Register] save session failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, RegisterResponse{ErrorMsg: InternalError})
		return
	}
	if err := trackSession(c.Request.Context(), user.ID, session.ID()); err != nil {
		logger.ErrorF(c.Request.Context(), "[Register] track session failed: %s", err.Error())
	}
	// log
	LogForAudit(c.Request.Context(), user, c)
	logger.InfoF(c.Request.Context(), "[Register] %d %s", user.ID, user.Username)
	// response
	c.JSON(
		http.StatusCreated,
		RegisterResponse{
			Data: BasicUserInfo{
				ID:        user.ID,
				Username:  user.Username,
				Nickname:  user.Nickname,
				Email:     user.Email,
				AvatarURL: user.AvatarURL,
				IsAdmin:   user.IsAdmin,
			},
		},
	)
}

type LogoutResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// Logout godoc
// @Tags auth
// @Produce json
// @Success 200 {object} LogoutResponse
// @Failure 500 {object} LogoutResponse
// @Router /api/v1/auth/logout [post]
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	userID, sessionID := GetUserIDFromSession(session), session.ID()
	session.Clear()
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, LogoutResponse{ErrorMsg: err.Error()})
		return
	}
	if userID > 0 && sessionID != "" {
		if err := untrackSession(c.Request.Context(), userID, sessionID); err != nil {
			logger.ErrorF(c.Request.Context(), "[Logout] untrack session failed: %s", err.Error())
		}
	}
	c.JSON(http.StatusOK, LogoutResponse{})
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password" binding:"required"`
}

type ChangePasswordResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ChangePassword godoc
// @Tags auth
// @Param request body ChangePasswordRequest true "request body"
// @Produce json
// @Success 200 {object} ChangePasswordResponse
// @Failure 400 {object} ChangePasswordResponse
// @Failure 401 {object} ChangePasswordResponse
// @Failure 500 {object} ChangePasswordResponse
// @Router /api/v1/auth/password [put]
func ChangePassword(c *gin.Context) {
	// init req
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ChangePasswordResponse{ErrorMsg: InvalidParams})
		return
	}
	user, ok := GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ChangePasswordResponse{ErrorMsg: UnAuthorized})
		return
	}
	// do change
	if err := doChangePassword(c.Request.Context(), user, req.OldPassword, req.NewPassword); err != nil {
		switch err.Error() {
		case WrongPassword, PasswordInvalid:
			c.JSON(http.StatusBadRequest, ChangePasswordResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[ChangePassword] change password failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, ChangePasswordResponse{ErrorMsg: InternalError})
		}
		return
	}
	// 保留当前设备, 其他设备下线
	if err := RevokeUserSessions(c.Request.Context(), user.ID, sessions.Default(c).ID()); err != nil {
		logger.ErrorF(c.Request.Context(), "[ChangePassword] revoke sessions failed: %s", err.Error())
	}
	logger.InfoF(c.Request.Context(), "[ChangePassword] %d %s", user.ID, user.Username)
	c.JSON(http.StatusOK, ChangePasswordResponse{})
}

type SendEmailCodeRequest struct {
	Email string `json:"email" binding:"required"`
}

type SendEmailCodeResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// SendEmailCode godoc
// @Tags auth
// @Param request body SendEmailCodeRequest true "request body"
// @Produce json
// @Success 200 {object} SendEmailCodeResponse
// @Failure 400 {object} SendEmailCodeResponse
// @Failure 401 {object} SendEmailCodeResponse
// @Failure 409 {object} SendEmailCodeResponse
// @Failure 429 {object} SendEmailCodeResponse
// @Failure 500 {object} SendEmailCodeResponse
// @Failure 503 {object} SendEmailCodeResponse
// @Router /api/v1/auth/email/code [post]
func SendEmailCode(c *gin.Context) {
	// init req
	var req SendEmailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, SendEmailCodeResponse{ErrorMsg: InvalidParams})
		return
	}
	user, ok := GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, SendEmailCodeResponse{ErrorMsg: UnAuthorized})
		return
	}
	// do send
	if err := doSendEmailCode(c.Request.Context(), user.ID, req.Email); err != nil {
		switch err.Error() {
		case EmailInvalid:
			c.JSON(http.StatusBadRequest, SendEmailCodeResponse{ErrorMsg: err.Error()})
		case EmailTaken:
			c.JSON(http.StatusConflict, SendEmailCodeResponse{ErrorMsg: err.Error()})
		case CodeTooFrequent:
			c.JSON(http.StatusTooManyRequests, SendEmailCodeResponse{ErrorMsg: err.Error()})
		case MailUnavailable:
			c.JSON(http.StatusServiceUnavailable, SendEmailCodeResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[SendEmailCode] send failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, SendEmailCodeResponse{ErrorMsg: InternalError})
		}
		return
	}
	c.JSON(http.StatusOK, SendEmailCodeResponse{})
}

type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

type VerifyEmailResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// VerifyEmail godoc
// @Tags auth
// @Param request body VerifyEmailRequest true "request body"
// @Produce json
// @Success 200 {object} VerifyEmailResponse
// @Failure 400 {object} VerifyEmailResponse
// @Failure 401 {object} VerifyEmailResponse
// @Failure 409 {object} VerifyEmailResponse
// @Failure 500 {object} VerifyEmailResponse
// @Router /api/v1/auth/email/verify [post]
func VerifyEmail(c *gin.Context) {
	// init req
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, VerifyEmailResponse{ErrorMsg: InvalidParams})
		return
	}
	user, ok := GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, VerifyEmailResponse{ErrorMsg: UnAuthorized})
		return
	}
	// do verify
	if err := doVerifyEmail(c.Request.Context(), user.ID, req.Email, req.Code); err != nil {
		switch err.Error() {
		case EmailInvalid, CodeInvalid:
			c.JSON(http.StatusBadRequest, VerifyEmailResponse{ErrorMsg: err.Error()})
		case EmailTaken:
			c.JSON(http.StatusConflict, VerifyEmailResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[VerifyEmail] verify failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, VerifyEmailResponse{ErrorMsg: InternalError})
		}
		return
	}
	logger.InfoF(c.Request.Context(), "[VerifyEmail] %d %s", user.ID, user.Username)
	c.JSON(http.StatusOK, VerifyEmailResponse{})
}

type BanUserRequest struct {
	BannedUntil time.Time `json:"banned_until" binding:"required"` // RFC3339, 必须晚于当前时间; 永久封禁请使用停用
	Reason      string    `json:"reason" binding:"max=255"`
}

type BanUserResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// BanUser godoc
// @Tags auth
// @Param id path int true "user id"
// @Param request body BanUserRequest true "request body"
// @Produce json
// @Success 200 {object} BanUserResponse
// @Failure 400 {object} BanUserResponse
// @Failure 401 {object} BanUserResponse
// @Failure 403 {object} BanUserResponse
// @Failure 404 {object} BanUserResponse
// @Failure 500 {object} BanUserResponse
// @Router /api/v1/auth/users/{id}/ban [put]
func BanUser(c *gin.Context) {
	// init req
	var req BanUserRequest
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, BanUserResponse{ErrorMsg: InvalidParams})
		return
	}
	operator, ok := GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, BanUserResponse{ErrorMsg: UnAuthorized})
		return
	}
	// do ban
	if err := doBanUser(c.Request.Context(), operator.ID, userID, req.BannedUntil, req.Reason); err != nil {
		switch err.Error() {
		case CannotOperateSelf, BanUntilInvalid:
			c.JSON(http.StatusBadRequest, BanUserResponse{ErrorMsg: err.Error()})
		case UserNotFound:
			c.JSON(http.StatusNotFound, BanUserResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[BanUser] ban failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, BanUserResponse{ErrorMsg: InternalError})
		}
		return
	}
	// 所有设备下线
	if err := RevokeUserSessions(c.Request.Context(), userID, ""); err != nil {
		logger.ErrorF(c.Request.Context(), "[BanUser] revoke sessions failed: %s", err.Error())
	}
	logger.InfoF(c.Request.Context(), "[BanUser] %d banned %d until %s", operator.ID, userID, req.BannedUntil.Format(time.RFC3339))
	c.JSON(http.StatusOK, BanUserResponse{})
}

type UnbanUserResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// UnbanUser godoc
// @Tags auth
// @Param id path int true "user id"
// @Produce json
// @Success 200 {object} UnbanUserResponse
// @Failure 400 {object} UnbanUserResponse
// @Failure 401 {object} UnbanUserResponse
// @Failure 403 {object} UnbanUserResponse
// @Failure 404 {object} UnbanUserResponse
// @Failure 500 {object} UnbanUserResponse
// @Router /api/v1/auth/users/{id}/ban [delete]
func UnbanUser(c *gin.Context) {
	// init req
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, UnbanUserResponse{ErrorMsg: InvalidParams})
		return
	}
	operator, ok := GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, UnbanUserResponse{ErrorMsg: UnAuthorized})
		return
	}
	// do unban
	if err := doUnbanUser(c.Request.Context(), operator.ID, userID); err != nil {
		switch err.Error() {
		case CannotOperateSelf:
			c.JSON(http.StatusBadRequest, UnbanUserResponse{ErrorMsg: err.Error()})
		case UserNotFound:
			c.JSON(http.StatusNotFound, UnbanUserResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[UnbanUser] unban failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, UnbanUserResponse{ErrorMsg: InternalError})
		}
		return
	}
	logger.InfoF(c.Request.Context(), "[UnbanUser] %d unbanned %d", operator.ID, userID)
	c.JSON(http.StatusOK, UnbanUserResponse{})
}

type SetUserDisabledRequest struct {
	Disabled *bool `json:"disabled" binding:"required"`
}

type SetUserDisabledResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// SetUserDisabled godoc
// @Tags auth
// @Param id path int true "user id"
// @Param request body SetUserDisabledRequest true "request body"
// @Produce json
// @Success 200 {object} SetUserDisabledResponse
// @Failure 400 {object} SetUserDisabledResponse
// @Failure 401 {object} SetUserDisabledResponse
// @Failure 403 {object} SetUserDisabledResponse
// @Failure 404 {object} SetUserDisabledResponse
// @Failure 500 {object} SetUserDisabledResponse
// @Router /api/v1/auth/users/{id}/disabled [put]
func SetUserDisabled(c *gin.Context) {
	// init req
	var req SetUserDisabledRequest
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, SetUserDisabledResponse{ErrorMsg: InvalidParams})
		return
	}
	operator, ok := GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, SetUserDisabledResponse{ErrorMsg: UnAuthorized})
		return
	}
	// do set
	if err := doSetUserDisabled(c.Request.Context(), operator.ID, userID, *req.Disabled); err != nil {
		switch err.Error() {
		case CannotOperateSelf:
			c.JSON(http.StatusBadRequest, SetUserDisabledResponse{ErrorMsg: err.Error()})
		case UserNotFound:
			c.JSON(http.StatusNotFound, SetUserDisabledResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[SetUserDisabled] set failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, SetUserDisabledResponse{ErrorMsg: InternalError})
		}
		return
	}
	// 停用时所有设备下线
	if *req.Disabled {
		if err := RevokeUserSessions(c.Request.Context(), userID, ""); err != nil {
			logger.ErrorF(c.Request.Context(), "[SetUserDisabled] revoke sessions failed: %s", err.Error())
		}
	}
	logger.InfoF(c.Request.Context(), "[SetUserDisabled] %d set %d disabled=%t", operator.ID, userID, *req.Disabled)
	c.JSON(http.StatusOK, SetUserDisabledResponse{})
}

type BasicUserInfo struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"` // 只有验证通过的邮箱才会写入, 未绑定为空
	AvatarURL string `json:"avatar_url"`
	IsAdmin   bool   `json:"is_admin"`
}

type UserInfoResponse struct {
	ErrorMsg string        `json:"error_msg"`
	Data     BasicUserInfo `json:"data"`
}

// UserInfo godoc
// @Tags auth
// @Produce json
// @Success 200 {object} UserInfoResponse
// @Failure 500 {object} UserInfoResponse
// @Router /api/v1/auth/user-info [get]
func UserInfo(c *gin.Context) {
	// init
	userID := GetUserIDFromContext(c)
	// query db
	var user User
	tx := db.DB(c.Request.Context()).Where("id = ?", userID).First(&user)
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, UserInfoResponse{ErrorMsg: tx.Error.Error()})
		return
	}
	c.JSON(
		http.StatusOK,
		UserInfoResponse{
			ErrorMsg: "",
			Data: BasicUserInfo{
				ID:        user.ID,
				Username:  user.Username,
				Nickname:  user.Nickname,
				Email:     user.Email,
				AvatarURL: user.AvatarURL,
				IsAdmin:   user.IsAdmin,
			},
		},
	)
}

type UpdateUserInfoRequest struct {
	Nickname  *string `json:"nickname"`   // 不传表示不修改
	AvatarURL *string `json:"avatar_url"` // 不传表示不修改, 传空字符串表示清除头像
}

type UpdateUserInfoResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// UpdateUserInfo godoc
// @Tags auth
// @Param request body UpdateUserInfoRequest true "request body"
// @Produce json
// @Success 200 {object} UpdateUserInfoResponse
// @Failure 400 {object} UpdateUserInfoResponse
// @Failure 401 {object} UpdateUserInfoResponse
// @Failure 500 {object} UpdateUserInfoResponse
// @Router /api/v1/auth/user-info [patch]
func UpdateUserInfo(c *gin.Context) {
	// init req
	var req UpdateUserInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, UpdateUserInfoResponse{ErrorMsg: InvalidParams})
		return
	}
	user, ok := GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, UpdateUserInfoResponse{ErrorMsg: UnAuthorized})
		return
	}
	// do update
	updated, err := doUpdateUserInfo(c.Request.Context(), user.ID, nil, req.Nickname, req.AvatarURL)
	if err != nil {
		switch err.Error() {
		case NicknameInvalid, AvatarURLInvalid:
			c.JSON(http.StatusBadRequest, UpdateUserInfoResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[UpdateUserInfo] update failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, UpdateUserInfoResponse{ErrorMsg: InternalError})
		}
		return
	}
	logger.InfoF(c.Request.Context(), "[UpdateUserInfo] %d %s", updated.ID, updated.Username)
	c.JSON(
		http.StatusOK,
		UpdateUserInfoResponse{
			Data: BasicUserInfo{
				ID:        updated.ID,
				Username:  updated.Username,
				Nickname:  updated.Nickname,
				Email:     updated.Email,
				AvatarURL: updated.AvatarURL,
				IsAdmin:   updated.IsAdmin,
			},
		},
	)
}

type AdminUpdateUserRequest struct {
	Username  *string `json:"username"`   // 不传表示不修改
	Nickname  *string `json:"nickname"`   // 不传表示不修改
	AvatarURL *string `json:"avatar_url"` // 不传表示不修改, 传空字符串表示清除头像
}

type AdminUpdateUserResponse struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// AdminUpdateUser godoc
// @Tags auth
// @Param id path int true "user id"
// @Param request body AdminUpdateUserRequest true "request body"
// @Produce json
// @Success 200 {object} AdminUpdateUserResponse
// @Failure 400 {object} AdminUpdateUserResponse
// @Failure 401 {object} AdminUpdateUserResponse
// @Failure 403 {object} AdminUpdateUserResponse
// @Failure 404 {object} AdminUpdateUserResponse
// @Failure 409 {object} AdminUpdateUserResponse
// @Failure 500 {object} AdminUpdateUserResponse
// @Router /api/v1/auth/users/{id} [patch]
func AdminUpdateUser(c *gin.Context) {
	// init req
	var req AdminUpdateUserRequest
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, AdminUpdateUserResponse{ErrorMsg: InvalidParams})
		return
	}
	operator, ok := GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, AdminUpdateUserResponse{ErrorMsg: UnAuthorized})
		return
	}
	// do update
	updated, err := doUpdateUserInfo(c.Request.Context(), userID, req.Username, req.Nickname, req.AvatarURL)
	if err != nil {
		switch err.Error() {
		case UsernameInvalid, NicknameInvalid, AvatarURLInvalid:
			c.JSON(http.StatusBadRequest, AdminUpdateUserResponse{ErrorMsg: err.Error()})
		case UserNotFound:
			c.JSON(http.StatusNotFound, AdminUpdateUserResponse{ErrorMsg: err.Error()})
		case UsernameTaken:
			c.JSON(http.StatusConflict, AdminUpdateUserResponse{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(c.Request.Context(), "[AdminUpdateUser] update failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, AdminUpdateUserResponse{ErrorMsg: InternalError})
		}
		return
	}
	logger.InfoF(c.Request.Context(), "[AdminUpdateUser] %d updated %d %s", operator.ID, updated.ID, updated.Username)
	c.JSON(
		http.StatusOK,
		AdminUpdateUserResponse{
			Data: BasicUserInfo{
				ID:        updated.ID,
				Username:  updated.Username,
				Nickname:  updated.Nickname,
				Email:     updated.Email,
				AvatarURL: updated.AvatarURL,
				IsAdmin:   updated.IsAdmin,
			},
		},
	)
}
