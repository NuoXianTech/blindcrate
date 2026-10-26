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

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
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
	session.Clear()
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, LogoutResponse{ErrorMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, LogoutResponse{})
}

type BasicUserInfo struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
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
				AvatarURL: user.AvatarURL,
				IsAdmin:   user.IsAdmin,
			},
		},
	)
}
