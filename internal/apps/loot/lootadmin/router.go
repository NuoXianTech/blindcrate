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

package lootadmin

import (
	"net/http"
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListLootPoolsRequest 掉落池列表查询参数
type ListLootPoolsRequest struct {
	Current   int    `json:"current" form:"current" binding:"min=1"`
	Size      int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword   string `json:"keyword" form:"keyword" binding:"max=64"` // 按代码或名称搜索, 不区分大小写
	IsEnabled *bool  `json:"is_enabled" form:"is_enabled"`
}

// ListLootPoolsResponseData 掉落池列表分页结果
type ListLootPoolsResponseData struct {
	Total int64              `json:"total"`
	Pools []LootPoolListItem `json:"pools"`
}

// ListLootPools
// @Tags admin
// @Summary 获取掉落池列表
// @Produce json
// @Param request query ListLootPoolsRequest true "request query"
// @Success 200 {object} Response{data=ListLootPoolsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/loot-pools [get]
func ListLootPools(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListLootPoolsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, pools, err := QueryLootPools(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListLootPools] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListLootPoolsResponseData{Total: total, Pools: pools}})
}

// LootPoolRequest 新建或修改掉落池请求体
type LootPoolRequest struct {
	Code        string `json:"code" binding:"required,max=64"` // 玩法中引用的代码, 小写字母、数字和下划线
	Name        string `json:"name" binding:"required,max=64"`
	Description string `json:"description" binding:"max=255"`
	IsEnabled   bool   `json:"is_enabled"` // 停用后不能再抽取
}

// CreateLootPool
// @Tags admin
// @Summary 新建掉落池
// @Description 新建后没有条目, 需要再替换条目
// @Accept json
// @Produce json
// @Param request body LootPoolRequest true "request body"
// @Success 200 {object} Response{data=loot.Pool}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/loot-pools [post]
func CreateLootPool(c *gin.Context) {
	ctx := c.Request.Context()
	var req LootPoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	created, err := doCreateLootPool(ctx, &req)
	if err != nil {
		switch err.Error() {
		case loot.PoolCodeInvalid, loot.PoolNameInvalid:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case loot.PoolCodeTaken:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[CreateLootPool] create failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLootPoolCreate, audit.TargetLootPool, created.ID, lootPoolFields(created))
	logger.InfoF(ctx, "[CreateLootPool] %d created loot pool %d %s", operator.ID, created.ID, created.Code)
	c.JSON(http.StatusOK, Response{Data: created})
}

// UpdateLootPool
// @Tags admin
// @Summary 修改掉落池
// @Description 需提交全部字段; 修改代码后, 仍使用旧代码的玩法会找不到掉落池
// @Accept json
// @Produce json
// @Param id path int true "loot pool id"
// @Param request body LootPoolRequest true "request body"
// @Success 200 {object} Response{data=loot.Pool}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/loot-pools/{id} [put]
func UpdateLootPool(c *gin.Context) {
	ctx := c.Request.Context()
	var req LootPoolRequest
	poolID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	updated, changes, err := doUpdateLootPool(ctx, poolID, &req)
	if err != nil {
		switch err.Error() {
		case loot.PoolCodeInvalid, loot.PoolNameInvalid:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case loot.PoolNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		case loot.PoolCodeTaken:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[UpdateLootPool] update failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLootPoolUpdate, audit.TargetLootPool, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateLootPool] %d updated loot pool %d %s", operator.ID, updated.ID, updated.Code)
	}
	c.JSON(http.StatusOK, Response{Data: updated})
}

// DeleteLootPoolResponseData 删除掉落池的结果
type DeleteLootPoolResponseData struct {
	EntryCount int64 `json:"entry_count"` // 一同删除的条目数
}

// DeleteLootPool
// @Tags admin
// @Summary 删除掉落池
// @Description 连同条目一起删除, 仍使用该代码的玩法会找不到掉落池
// @Produce json
// @Param id path int true "loot pool id"
// @Success 200 {object} Response{data=DeleteLootPoolResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/loot-pools/{id} [delete]
func DeleteLootPool(c *gin.Context) {
	ctx := c.Request.Context()
	poolID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	deleted, entries, err := doDeleteLootPool(ctx, poolID)
	if err != nil {
		if err.Error() == loot.PoolNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[DeleteLootPool] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLootPoolDelete, audit.TargetLootPool, deleted.ID, map[string]interface{}{
		"code":        deleted.Code,
		"name":        deleted.Name,
		"entry_count": entries,
	})
	logger.InfoF(ctx, "[DeleteLootPool] %d deleted loot pool %d %s, %d entries", operator.ID, deleted.ID, deleted.Code, entries)
	c.JSON(http.StatusOK, Response{Data: DeleteLootPoolResponseData{EntryCount: entries}})
}

// LootPoolEntriesResponseData 掉落池及其全部条目
type LootPoolEntriesResponseData struct {
	Pool    loot.Pool          `json:"pool"`
	Entries []loot.EntryDetail `json:"entries"` // 按条目 ID 排序, 包含不参与抽取的条目
}

// ListLootPoolEntries
// @Tags admin
// @Summary 获取掉落池的全部条目
// @Description 包含已下架、已删除的物品和已停用的货币, 这些条目不参与抽取, 概率按其余条目计算
// @Produce json
// @Param id path int true "loot pool id"
// @Success 200 {object} Response{data=LootPoolEntriesResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/loot-pools/{id}/entries [get]
func ListLootPoolEntries(c *gin.Context) {
	ctx := c.Request.Context()
	poolID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	pool, entries, err := queryLootPoolEntries(ctx, poolID)
	if err != nil {
		if err.Error() == loot.PoolNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[ListLootPoolEntries] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: LootPoolEntriesResponseData{Pool: *pool, Entries: entries}})
}

// LootPoolEntryRequest 掉落池的一个条目
type LootPoolEntryRequest struct {
	ItemID      uint64 `json:"item_id"`                                                 // 掉落的物品, 与 currency 二选一
	Currency    string `json:"currency" binding:"max=32"`                               // 掉落的货币
	Weight      int32  `json:"weight" binding:"required,min=1,max=1000000"`             // 权重
	MinQuantity int64  `json:"min_quantity" binding:"required,min=1,max=1000000000000"` // 每次掉落的数量范围, 货币为金额, 物品最多 1000000
	MaxQuantity int64  `json:"max_quantity" binding:"required,min=1,max=1000000000000"`
}

// ReplaceLootPoolEntriesRequest 替换掉落池条目请求体
type ReplaceLootPoolEntriesRequest struct {
	Entries []LootPoolEntryRequest `json:"entries" binding:"required,max=200,dive"` // 提交空数组清空条目
}

// ReplaceLootPoolEntries
// @Tags admin
// @Summary 替换掉落池的全部条目
// @Description 用提交的条目整体替换; 物品需未删除, 货币需存在, 已下架的物品和已停用的货币也可以加入, 恢复后才参与抽取
// @Accept json
// @Produce json
// @Param id path int true "loot pool id"
// @Param request body ReplaceLootPoolEntriesRequest true "request body"
// @Success 200 {object} Response{data=LootPoolEntriesResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/loot-pools/{id}/entries [put]
func ReplaceLootPoolEntries(c *gin.Context) {
	ctx := c.Request.Context()
	var req ReplaceLootPoolEntriesRequest
	poolID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	pool, entries, changes, err := doReplaceLootPoolEntries(ctx, poolID, &req)
	if err != nil {
		switch err.Error() {
		case loot.EntryTargetInvalid, loot.EntryRangeInvalid, loot.EntryTooMany, inventory.QuantityInvalid, item.ItemNotFound, wallet.CurrencyNotFound:
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		case loot.PoolNotFound:
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[ReplaceLootPoolEntries] replace failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionLootPoolEntriesUpdate, audit.TargetLootPool, pool.ID, changes)
	logger.InfoF(ctx, "[ReplaceLootPoolEntries] %d replaced loot pool %d %s with %d entries", operator.ID, pool.ID, pool.Code, len(entries))
	c.JSON(http.StatusOK, Response{Data: LootPoolEntriesResponseData{Pool: *pool, Entries: entries}})
}
