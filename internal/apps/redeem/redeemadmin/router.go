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

// Package redeemadmin 兑换码的后台接口: 生成和管理兑换码批次, 查看兑换码和兑换记录
package redeemadmin

import (
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/redeem"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// ListBatchesRequest 批次列表查询参数
type ListBatchesRequest struct {
	Current int    `json:"current" form:"current" binding:"min=1"`
	Size    int    `json:"size" form:"size" binding:"min=1,max=100"`
	Keyword string `json:"keyword" form:"keyword" binding:"max=64"`                                // 按名称搜索, 不区分大小写; 也可以填完整的兑换码, 查它属于哪个批次
	Status  string `json:"status" form:"status" binding:"omitempty,oneof=active expired disabled"` // active 生效中, expired 已到期, disabled 已停用; 为空时不筛选
}

// ListBatchesResponseData 批次列表分页结果
type ListBatchesResponseData struct {
	Total   int64       `json:"total"`
	Batches []BatchItem `json:"batches"`
}

// ListRedeemBatches
// @Tags admin
// @Summary 获取兑换码批次列表
// @Description 从新到旧, 不含已删除的批次, 带上生成人的用户名和已兑换的次数
// @Produce json
// @Param request query ListBatchesRequest true "request query"
// @Success 200 {object} reply.Response{data=ListBatchesResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/redeem/batches [get]
func ListRedeemBatches(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListBatchesRequest{}
	if !reply.Query(c, req) {
		return
	}
	total, batches, err := QueryBatches(ctx, req)
	if err != nil {
		reply.Fail(c, err, "[ListRedeemBatches] query failed")
		return
	}
	reply.OK(c, ListBatchesResponseData{Total: total, Batches: batches})
}

// BatchRequest 修改批次的请求体, 也是生成兑换码时批次的设置
//
// 奖励、使用条件和次数不在这里校验, 由 redeem.Validate 校验, 提示更具体。
type BatchRequest struct {
	Name         string                `json:"name" binding:"required,max=64"` // 兑换成功后展示给玩家, 也是流水的备注
	Reward       reward.Bundle         `json:"reward" binding:"-"`             // 兑换一次得到的奖励, 数量为正数, 不能为空
	Requirements []condition.Condition `json:"requirements" binding:"-"`       // 使用条件, 需要全部满足
	MaxUses      int32                 `json:"max_uses"`                       // 每个兑换码可以被多少个用户兑换, 1 为单次使用, 0 表示不限
	UserLimit    int32                 `json:"user_limit"`                     // 每个用户最多兑换这个批次中的几个兑换码, 0 表示不限
	ExpiresAt    *time.Time            `json:"expires_at"`                     // RFC3339, 不包含, 精确到分钟, 为空时不过期
	IsEnabled    bool                  `json:"is_enabled"`                     // 停用后不能兑换
}

// CreateBatchRequest 生成兑换码的请求体
type CreateBatchRequest struct {
	BatchRequest
	Count          int    `json:"count"`                                                // 随机生成的个数, 1-10000; 指定了兑换码时忽略
	Code           string `json:"code" binding:"max=64"`                                // 指定兑换码, 4-32 个字母、数字或汉字, 不区分大小写; 不为空时只生成这一个
	IdempotencyKey string `json:"idempotency_key" binding:"required,max=64,printascii"` // 每次打开生成框生成一个, 重复提交只生成一次
}

// CreateRedeemBatch
// @Tags admin
// @Summary 生成兑换码
// @Description 生成一个批次的兑换码, 共用名称、奖励、使用条件、可使用次数和到期时间: 指定了兑换码时只生成这一个, 否则随机生成 count 个 12 位的兑换码。兑换码与已有的重复时返回 409; 幂等键相同的重复提交返回已有的批次, 内容与已有的批次不同时返回 409
// @Accept json
// @Produce json
// @Param request body CreateBatchRequest true "request body"
// @Success 200 {object} reply.Response{data=redeem.Batch}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 409 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/redeem/batches [post]
func CreateRedeemBatch(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreateBatchRequest
	if !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	created, replayed, err := doCreateBatch(ctx, operator.ID, &req)
	if err != nil {
		reply.Fail(c, err, "[CreateRedeemBatch] create failed")
		return
	}
	if replayed {
		reply.OK(c, created)
		return
	}
	// 兑换码本身不记进操作日志和日志, 只记个数
	detail := batchFields(created)
	detail["code_count"] = created.CodeCount
	detail["custom"] = strings.TrimSpace(req.Code) != ""
	audit.Record(c, operator.ID, audit.ScopeAdmin, redeem.ActionCreate, redeem.TargetBatch, created.ID, detail)
	logger.InfoF(ctx, "[CreateRedeemBatch] %d created redeem batch %d with %d codes", operator.ID, created.ID, created.CodeCount)
	reply.OK(c, created)
}

// UpdateRedeemBatch
// @Tags admin
// @Summary 修改兑换码批次
// @Description 需提交全部字段, 兑换码本身不能修改; 修改后只影响之后的兑换, 已兑换的不受影响。到期时间可以改到过去, 批次立即到期
// @Accept json
// @Produce json
// @Param id path int true "batch id"
// @Param request body BatchRequest true "request body"
// @Success 200 {object} reply.Response{data=redeem.Batch}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/redeem/batches/{id} [put]
func UpdateRedeemBatch(c *gin.Context) {
	ctx := c.Request.Context()
	var req BatchRequest
	batchID, ok := reply.ID(c, "id")
	if !ok || !reply.JSON(c, &req) {
		return
	}
	operator := auth.CurrentUser(c)
	updated, changes, err := doUpdateBatch(ctx, batchID, &req)
	if err != nil {
		reply.Fail(c, err, "[UpdateRedeemBatch] update failed")
		return
	}
	if len(changes) > 0 {
		audit.Record(c, operator.ID, audit.ScopeAdmin, redeem.ActionUpdate, redeem.TargetBatch, updated.ID, changes)
		logger.InfoF(ctx, "[UpdateRedeemBatch] %d updated redeem batch %d", operator.ID, updated.ID)
	}
	reply.OK(c, updated)
}

// DeleteRedeemBatch
// @Tags admin
// @Summary 删除兑换码批次
// @Description 软删除, 批次中的兑换码不能再兑换, 也不能再被新的批次使用; 已兑换的奖励不收回
// @Produce json
// @Param id path int true "batch id"
// @Success 200 {object} reply.Response
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/redeem/batches/{id} [delete]
func DeleteRedeemBatch(c *gin.Context) {
	ctx := c.Request.Context()
	batchID, ok := reply.ID(c, "id")
	if !ok {
		return
	}
	operator := auth.CurrentUser(c)
	deleted, err := doDeleteBatch(ctx, batchID)
	if err != nil {
		reply.Fail(c, err, "[DeleteRedeemBatch] delete failed")
		return
	}
	detail := batchFields(deleted)
	detail["code_count"] = deleted.CodeCount
	audit.Record(c, operator.ID, audit.ScopeAdmin, redeem.ActionDelete, redeem.TargetBatch, deleted.ID, detail)
	logger.InfoF(ctx, "[DeleteRedeemBatch] %d deleted redeem batch %d", operator.ID, deleted.ID)
	reply.OK(c, nil)
}

// ListCodesResponseData 批次的全部兑换码
type ListCodesResponseData struct {
	Codes []redeem.Code `json:"codes"`
}

// ListRedeemBatchCodes
// @Tags admin
// @Summary 获取批次的兑换码
// @Description 一次返回批次的全部兑换码和各自已兑换的次数, 按生成顺序排列, 最多 10000 个
// @Produce json
// @Param id path int true "batch id"
// @Success 200 {object} reply.Response{data=ListCodesResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 404 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/redeem/batches/{id}/codes [get]
func ListRedeemBatchCodes(c *gin.Context) {
	ctx := c.Request.Context()
	batchID, ok := reply.ID(c, "id")
	if !ok {
		return
	}
	codes, err := QueryCodes(ctx, batchID)
	if err != nil {
		reply.Fail(c, err, "[ListRedeemBatchCodes] query failed")
		return
	}
	reply.OK(c, ListCodesResponseData{Codes: codes})
}

// ListRecordsRequest 兑换记录查询参数
type ListRecordsRequest struct {
	Current int     `json:"current" form:"current" binding:"min=1"`
	Size    int     `json:"size" form:"size" binding:"min=1,max=100"`
	BatchID *uint64 `json:"batch_id" form:"batch_id"`
	UserID  *uint64 `json:"user_id" form:"user_id"`
	Code    string  `json:"code" form:"code" binding:"max=64"` // 完整的兑换码, 不区分大小写
}

// ListRecordsResponseData 兑换记录分页结果
type ListRecordsResponseData struct {
	Total   int64        `json:"total"`
	Records []RecordItem `json:"records"`
}

// ListRedeemRecords
// @Tags admin
// @Summary 查询兑换记录
// @Description 从新到旧, 包括已删除的批次的记录, 带上用户名、批次名称和兑换码
// @Produce json
// @Param request query ListRecordsRequest true "request query"
// @Success 200 {object} reply.Response{data=ListRecordsResponseData}
// @Failure 400 {object} reply.Response
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/redeem/records [get]
func ListRedeemRecords(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListRecordsRequest{}
	if !reply.Query(c, req) {
		return
	}
	total, records, err := QueryRecords(ctx, req)
	if err != nil {
		reply.Fail(c, err, "[ListRedeemRecords] query failed")
		return
	}
	reply.OK(c, ListRecordsResponseData{Total: total, Records: records})
}
