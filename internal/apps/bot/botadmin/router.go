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

package botadmin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/bot"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListNamesRequest 人机昵称列表查询参数
type ListNamesRequest struct {
	Current   int    `json:"current" form:"current" binding:"min=1"`
	Size      int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"` // 按昵称搜索, 不区分大小写
	IsEnabled *bool  `json:"is_enabled" form:"is_enabled"`
}

// ListNamesResponseData 人机昵称列表分页结果
type ListNamesResponseData struct {
	Total int64      `json:"total"`
	Names []bot.Name `json:"names"`
}

// ListBotNames
// @Tags admin
// @Summary 获取人机昵称列表
// @Description 按 ID 倒序排列
// @Produce json
// @Param request query ListNamesRequest true "request query"
// @Success 200 {object} Response{data=ListNamesResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/bot/names [get]
func ListBotNames(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListNamesRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, names, err := QueryNames(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListBotNames] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListNamesResponseData{Total: total, Names: names}})
}

// CreateNamesRequest 批量新建人机昵称请求体
type CreateNamesRequest struct {
	Names     []string `json:"names" binding:"-"` // 1-200 个, 去掉首尾空白后为 1-32 个字符; 已存在的和重复的跳过
	IsEnabled bool     `json:"is_enabled"`
}

// CreateNamesResponseData 批量新建人机昵称的结果
type CreateNamesResponseData struct {
	Created []bot.Name `json:"created"`
	Skipped []string   `json:"skipped"` // 已存在或重复而跳过的昵称
}

// NameRequest 修改人机昵称请求体
type NameRequest struct {
	Name      string `json:"name" binding:"required,max=64"` // 去掉首尾空白后为 1-32 个字符
	IsEnabled bool   `json:"is_enabled"`
}

// nameErrorStatus 昵称错误对应的状态码, 不是昵称错误时返回 0
func nameErrorStatus(err error) int {
	var invalid *bot.InvalidError
	switch {
	case errors.As(err, &invalid):
		return http.StatusBadRequest
	case err.Error() == bot.NameNotFound:
		return http.StatusNotFound
	case err.Error() == bot.NameTaken:
		return http.StatusConflict
	}
	return 0
}

// CreateBotNames
// @Tags admin
// @Summary 批量新建人机昵称
// @Description 已存在的昵称和本次重复的昵称跳过, 在结果的 skipped 中返回
// @Accept json
// @Produce json
// @Param request body CreateNamesRequest true "request body"
// @Success 200 {object} Response{data=CreateNamesResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/bot/names [post]
func CreateBotNames(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreateNamesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	result, err := doCreateNames(ctx, &req)
	if err != nil {
		if status := nameErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateBotNames] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(result.Created) > 0 {
		names := make([]string, len(result.Created))
		for i := range result.Created {
			names[i] = result.Created[i].Name
		}
		audit.Record(c, operator.ID, audit.ScopeAdmin, bot.ActionNameCreate, bot.TargetName, 0, map[string]interface{}{
			"names":      names,
			"is_enabled": req.IsEnabled,
		})
		logger.InfoF(ctx, "[CreateBotNames] %d created %d bot names", operator.ID, len(names))
	}
	c.JSON(http.StatusOK, Response{Data: result})
}

// UpdateBotName
// @Tags admin
// @Summary 修改人机昵称
// @Description 已开局的对局中保存的是昵称本身, 不受影响
// @Accept json
// @Produce json
// @Param id path int true "name id"
// @Param request body NameRequest true "request body"
// @Success 200 {object} Response{data=bot.Name}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/bot/names/{id} [put]
func UpdateBotName(c *gin.Context) {
	ctx := c.Request.Context()
	var req NameRequest
	nameID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateName(ctx, nameID, &req)
	if err != nil {
		if status := nameErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateBotName] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, bot.ActionNameUpdate, bot.TargetName, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateBotName] %d updated bot name %d %s", operator.ID, updated.ID, updated.Name)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteBotName
// @Tags admin
// @Summary 删除人机昵称
// @Description 物理删除, 已开局的对局中保存的是昵称本身, 不受影响
// @Produce json
// @Param id path int true "name id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/bot/names/{id} [delete]
func DeleteBotName(c *gin.Context) {
	ctx := c.Request.Context()
	nameID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeleteName(ctx, nameID)
	if err != nil {
		if err.Error() == bot.NameNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteBotName] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, bot.ActionNameDelete, bot.TargetName, deleted.ID, nameFields(deleted))
	logger.InfoF(ctx, "[DeleteBotName] %d deleted bot name %d %s", operator.ID, deleted.ID, deleted.Name)
	c.JSON(http.StatusOK, Response{})
}

// ListPersonasRequest 人机风格列表查询参数
type ListPersonasRequest struct {
	Current int    `json:"current" form:"current" binding:"min=1"`
	Size    int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword string `json:"keyword" form:"keyword" binding:"max=64"` // 按代码或名称搜索, 不区分大小写
}

// ListPersonasResponseData 人机风格列表分页结果
type ListPersonasResponseData struct {
	Total    int64         `json:"total"`
	Personas []bot.Persona `json:"personas"`
}

// ListBotPersonas
// @Tags admin
// @Summary 获取人机风格列表
// @Description 按 ID 排列, 不含已删除的风格
// @Produce json
// @Param request query ListPersonasRequest true "request query"
// @Success 200 {object} Response{data=ListPersonasResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/bot/personas [get]
func ListBotPersonas(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListPersonasRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, personas, err := QueryPersonas(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListBotPersonas] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListPersonasResponseData{Total: total, Personas: personas}})
}

// PersonaRequest 新建或修改人机风格请求体
//
// 策略和参数不在这里校验, 由 bot.Validate 按策略的参数结构体校验, 提示更具体。
type PersonaRequest struct {
	Code        string         `json:"code" binding:"required,max=32"` // 小写字母、数字和下划线, 创建后不能修改
	Name        string         `json:"name" binding:"required,max=64"`
	Description string         `json:"description" binding:"max=255"`
	Strategy    string         `json:"strategy" binding:"required,max=32"` // 决策策略的代码, 见 GET /admin/bot/strategies
	Params      map[string]any `json:"params" binding:"-"`                 // 策略参数, 缺少的参数取默认值
}

// InUseError 风格正在被地图使用, 不能删除
type InUseError struct {
	Venue string // 引用风格的一张地图的代码
}

func (e *InUseError) Error() string { return fmt.Sprintf(bot.PersonaInUse, e.Venue) }

// personaErrorStatus 风格配置错误对应的状态码, 不是配置错误时返回 0
func personaErrorStatus(err error) int {
	var (
		invalid *bot.InvalidError
		inUse   *InUseError
	)
	switch {
	case errors.As(err, &invalid), err.Error() == bot.CodeImmutable:
		return http.StatusBadRequest
	case err.Error() == bot.PersonaNotFound:
		return http.StatusNotFound
	case err.Error() == bot.CodeTaken, errors.As(err, &inUse):
		return http.StatusConflict
	}
	return 0
}

// CreateBotPersona
// @Tags admin
// @Summary 新建人机风格
// @Accept json
// @Produce json
// @Param request body PersonaRequest true "request body"
// @Success 200 {object} Response{data=bot.Persona}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/bot/personas [post]
func CreateBotPersona(c *gin.Context) {
	ctx := c.Request.Context()
	var req PersonaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreatePersona(ctx, &req)
	if err != nil {
		if status := personaErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateBotPersona] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, bot.ActionPersonaCreate, bot.TargetPersona, created.ID, personaFields(created))
	logger.InfoF(ctx, "[CreateBotPersona] %d created bot persona %d %s", operator.ID, created.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateBotPersona
// @Tags admin
// @Summary 修改人机风格
// @Description 需提交全部字段, 代码不能修改; 进行中的对局使用开局时的快照, 不受影响
// @Accept json
// @Produce json
// @Param id path int true "persona id"
// @Param request body PersonaRequest true "request body"
// @Success 200 {object} Response{data=bot.Persona}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/bot/personas/{id} [put]
func UpdateBotPersona(c *gin.Context) {
	ctx := c.Request.Context()
	var req PersonaRequest
	personaID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdatePersona(ctx, personaID, &req)
	if err != nil {
		if status := personaErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateBotPersona] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, bot.ActionPersonaUpdate, bot.TargetPersona, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateBotPersona] %d updated bot persona %d %s", operator.ID, updated.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteBotPersona
// @Tags admin
// @Summary 删除人机风格
// @Description 软删除, 代码不能再被新风格使用; 被地图引用时不能删除
// @Produce json
// @Param id path int true "persona id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/bot/personas/{id} [delete]
func DeleteBotPersona(c *gin.Context) {
	ctx := c.Request.Context()
	personaID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeletePersona(ctx, personaID)
	if err != nil {
		if status := personaErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteBotPersona] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, bot.ActionPersonaDelete, bot.TargetPersona, deleted.ID, personaFields(deleted))
	logger.InfoF(ctx, "[DeleteBotPersona] %d deleted bot persona %d %s", operator.ID, deleted.ID, deleted.Code)
	c.JSON(http.StatusOK, Response{})
}

// ListBotStrategies
// @Tags admin
// @Summary 获取人机决策策略
// @Description 由玩法模块注册, 按注册顺序; fields 为参数的表单描述, defaults 为新建风格时的默认参数
// @Produce json
// @Success 200 {object} Response{data=[]bot.StrategyView}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Router /api/v1/admin/bot/strategies [get]
func ListBotStrategies(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Data: bot.Strategies()})
}
