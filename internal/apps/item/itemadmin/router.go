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

package itemadmin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/effect"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListItemsRequest 物品列表查询参数
type ListItemsRequest struct {
	Current   int    `json:"current" form:"current" binding:"min=1"`
	Size      int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"`          // 按代码或名称搜索, 不区分大小写
	Quality   *int16 `json:"quality" form:"quality" binding:"omitempty,min=1"` // 品质等级
	Category  string `json:"category" form:"category" binding:"max=32"`        // 类别代码
	IsEnabled *bool  `json:"is_enabled" form:"is_enabled"`
}

// ListItemsResponseData 物品列表分页结果
type ListItemsResponseData struct {
	Total int64          `json:"total"`
	Items []ItemListItem `json:"items"`
}

// ListItems
// @Tags admin
// @Summary 获取物品列表
// @Description 包含已下架的物品, 不包含已删除的物品
// @Produce json
// @Param request query ListItemsRequest true "request query"
// @Success 200 {object} Response{data=ListItemsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/items [get]
func ListItems(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListItemsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, items, err := QueryItems(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListItems] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListItemsResponseData{Total: total, Items: items}})
}

// ItemRequest 新建或修改物品请求体
type ItemRequest struct {
	Code         string         `json:"code" binding:"required,max=64"` // 英文别名, 小写字母、数字和下划线, 可以重复
	Name         string         `json:"name" binding:"required,max=64"`
	Description  string         `json:"description" binding:"max=255"`
	Quality      int16          `json:"quality" binding:"required,min=1"`   // 品质等级, 必须存在
	Category     string         `json:"category" binding:"required,max=32"` // 类别代码, 必须已注册
	ImageURL     string         `json:"image_url" binding:"omitempty,max=1024,image_url"`
	SellCurrency string         `json:"sell_currency" binding:"required,max=32"`
	SellPrice    int64          `json:"sell_price" binding:"min=0,max=1000000000000"` // 0 表示不可卖出
	Width        int16          `json:"width" binding:"required,min=1,max=10"`        // 在仓库中占的格数
	Height       int16          `json:"height" binding:"required,min=1,max=10"`
	Effect       string         `json:"effect,omitempty" binding:"max=32"`            // 使用效果代码, 空表示不能使用; 藏品不能配置
	EffectParams map[string]any `json:"effect_params,omitempty" swaggertype:"object"` // 效果参数, 结构见 GET /admin/items/effects
	IsEnabled    bool           `json:"is_enabled"`                                   // 下架后不会掉落, 也不能再发放, 已有的保留
}

// CreateItem
// @Tags admin
// @Summary 新建物品
// @Accept json
// @Produce json
// @Param request body ItemRequest true "request body"
// @Success 200 {object} Response{data=item.Item}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/items [post]
func CreateItem(c *gin.Context) {
	ctx := c.Request.Context()
	var req ItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateItem(ctx, &req)
	if err != nil {
		var invalid *effect.InvalidError
		if errors.As(err, &invalid) {
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
			return
		}
		switch err.Error() {
		case item.ItemCodeInvalid, item.ItemNameInvalid, item.QualityNotFound, item.CategoryNotFound, wallet.CurrencyNotFound:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case item.ItemNameTaken:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[CreateItem] create failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionItemCreate, audit.TargetItem, created.ID, itemFields(created))
	logger.InfoF(ctx, "[CreateItem] %d created item %d %s", operator.ID, created.ID, created.Name)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateItem
// @Tags admin
// @Summary 修改物品
// @Description 需提交全部字段; 修改名称不影响已有流水中的名称
// @Accept json
// @Produce json
// @Param id path int true "item id"
// @Param request body ItemRequest true "request body"
// @Success 200 {object} Response{data=item.Item}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/items/{id} [put]
func UpdateItem(c *gin.Context) {
	ctx := c.Request.Context()
	var req ItemRequest
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateItem(ctx, itemID, &req)
	if err != nil {
		var invalid *effect.InvalidError
		if errors.As(err, &invalid) {
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
			return
		}
		switch err.Error() {
		case item.ItemCodeInvalid, item.ItemNameInvalid, item.QualityNotFound, item.CategoryNotFound, wallet.CurrencyNotFound:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case item.ItemNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case item.ItemNameTaken:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[UpdateItem] update failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionItemUpdate, audit.TargetItem, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateItem] %d updated item %d %s", operator.ID, updated.ID, updated.Name)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteItemResponseData 删除物品的结果
type DeleteItemResponseData struct {
	HolderCount int64 `json:"holder_count"` // 删除时持有该物品的用户数
}

// DeleteItem
// @Tags admin
// @Summary 删除物品
// @Description 已持有的玩家背包保留该物品并变为占位物品, 物品 ID 不会被复用
// @Produce json
// @Param id path int true "item id"
// @Success 200 {object} Response{data=DeleteItemResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/items/{id} [delete]
func DeleteItem(c *gin.Context) {
	ctx := c.Request.Context()
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, holders, err := doDeleteItem(ctx, itemID)
	if err != nil {
		if err.Error() == item.ItemNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteItem] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionItemDelete, audit.TargetItem, deleted.ID, map[string]interface{}{
		"name":         deleted.Name,
		"holder_count": holders,
	})
	logger.InfoF(ctx, "[DeleteItem] %d deleted item %d %s, %d holders", operator.ID, deleted.ID, deleted.Name, holders)
	c.JSON(http.StatusOK, Response{Data: DeleteItemResponseData{HolderCount: holders}})
}

// ListItemCategoriesResponseData 物品类别列表
type ListItemCategoriesResponseData struct {
	Categories []item.Category `json:"categories"`
}

// ListItemCategories
// @Tags admin
// @Summary 获取物品类别
// @Description 由代码注册的全部物品类别, 按注册顺序
// @Produce json
// @Success 200 {object} Response{data=ListItemCategoriesResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Router /api/v1/admin/items/categories [get]
func ListItemCategories(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Data: ListItemCategoriesResponseData{Categories: item.Categories()}})
}

// ListItemEffectsResponseData 物品的使用效果列表
type ListItemEffectsResponseData struct {
	Effects []effect.TypeView `json:"effects"`
}

// ListItemEffects
// @Tags admin
// @Summary 获取物品的使用效果
// @Description 由代码注册的全部效果, 按注册顺序; fields 为参数的表单描述, 与条件类型相同
// @Produce json
// @Success 200 {object} Response{data=ListItemEffectsResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Router /api/v1/admin/items/effects [get]
func ListItemEffects(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Data: ListItemEffectsResponseData{Effects: effect.Types()}})
}

// ListQualitiesResponseData 品质列表
type ListQualitiesResponseData struct {
	Qualities []QualityListItem `json:"qualities"`
}

// ListQualities
// @Tags admin
// @Summary 获取品质列表
// @Description 按排序值排列, 带上使用每个品质的物品数
// @Produce json
// @Success 200 {object} Response{data=ListQualitiesResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/qualities [get]
func ListQualities(c *gin.Context) {
	ctx := c.Request.Context()
	qualities, err := queryQualities(ctx)
	if err != nil {
		logger.ErrorF(ctx, "[ListQualities] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListQualitiesResponseData{Qualities: qualities}})
}

// QualityRequest 修改品质请求体, 等级和代码创建后不能修改
type QualityRequest struct {
	Name      string `json:"name" binding:"required,max=32"`
	Color     string `json:"color" binding:"required,max=7"`                // #rrggbb
	Effect    string `json:"effect" binding:"max=16"`                       // 特效, 空表示纯色, rainbow 为流动渐变
	ScanMs    int32  `json:"scan_ms" binding:"min=100,max=10000"`           // 开箱时扫描这一品质藏品的时长, 毫秒; 品质越高可以扫得越久
	SortOrder int32  `json:"sort_order" binding:"min=-1000000,max=1000000"` // 筛选和图例中从小到大排列
}

// CreateQualityRequest 新建品质请求体
type CreateQualityRequest struct {
	Level int16  `json:"level" binding:"required,min=1,max=999"` // 等级, 越大越稀有
	Code  string `json:"code" binding:"required,max=32"`         // 自然键, 小写字母、数字和下划线
	QualityRequest
}

// qualityErrorStatus 品质配置错误对应的状态码, 不是配置错误时返回 0
func qualityErrorStatus(err error) int {
	switch err.Error() {
	case item.QualityCodeInvalid, item.QualityNameInvalid, item.QualityColorInvalid, item.QualityEffectInvalid:
		return http.StatusBadRequest
	case item.QualityNotFound:
		return http.StatusNotFound
	case item.QualityTaken, item.QualityInUse:
		return http.StatusConflict
	}
	return 0
}

// CreateQuality
// @Tags admin
// @Summary 新建品质
// @Accept json
// @Produce json
// @Param request body CreateQualityRequest true "request body"
// @Success 200 {object} Response{data=item.Quality}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/qualities [post]
func CreateQuality(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreateQualityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateQuality(ctx, &req)
	if err != nil {
		if status := qualityErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[CreateQuality] create failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionQualityCreate, audit.TargetQuality, uint64(created.Level), qualityFields(created))
	logger.InfoF(ctx, "[CreateQuality] %d created quality %d %s", operator.ID, created.Level, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateQuality
// @Tags admin
// @Summary 修改品质
// @Description 需提交全部字段, 等级和代码不能修改; 新的扫描时长只影响之后开局的对局
// @Accept json
// @Produce json
// @Param level path int true "quality level"
// @Param request body QualityRequest true "request body"
// @Success 200 {object} Response{data=item.Quality}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/qualities/{level} [put]
func UpdateQuality(c *gin.Context) {
	ctx := c.Request.Context()
	var req QualityRequest
	level, err := strconv.ParseInt(c.Param("level"), 10, 16)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateQuality(ctx, int16(level), &req)
	if err != nil {
		if status := qualityErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[UpdateQuality] update failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionQualityUpdate, audit.TargetQuality, uint64(updated.Level), changes)
		logger.InfoF(ctx, "[UpdateQuality] %d updated quality %d %s", operator.ID, updated.Level, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteQuality
// @Tags admin
// @Summary 删除品质
// @Description 还有未删除的物品使用该品质时不能删除; 已开局对局中的藏品保留原等级
// @Produce json
// @Param level path int true "quality level"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/qualities/{level} [delete]
func DeleteQuality(c *gin.Context) {
	ctx := c.Request.Context()
	level, err := strconv.ParseInt(c.Param("level"), 10, 16)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, err := doDeleteQuality(ctx, int16(level))
	if err != nil {
		if status := qualityErrorStatus(err); status != 0 {
			c.JSON(status, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteQuality] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionQualityDelete, audit.TargetQuality, uint64(deleted.Level), qualityFields(deleted))
	logger.InfoF(ctx, "[DeleteQuality] %d deleted quality %d %s", operator.ID, deleted.Level, deleted.Code)
	c.JSON(http.StatusOK, Response{})
}
