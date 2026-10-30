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

package pack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ExportPackRequest 导出配置包
type ExportPackRequest struct {
	Name     string   `json:"name" binding:"required,max=64"` // 配置包的名称
	Author   string   `json:"author" binding:"max=64"`
	Sections []string `json:"sections" binding:"required,min=1,max=32"` // 导出的配置种类, 见 GET /admin/packs/sections
}

// ListPackSections
// @Tags admin
// @Summary 获取配置种类
// @Description 按依赖顺序排列, 导入时也按这个顺序写入; renamable 为是否可以改名导入
// @Produce json
// @Success 200 {object} Response{data=[]SectionView}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Router /api/v1/admin/packs/sections [get]
func ListPackSections(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Data: Sections()})
}

// ExportPack
// @Tags admin
// @Summary 导出配置包
// @Description zip 文件, 包含 manifest.json、每种配置一个 JSON 文件, 以及 images/ 下引用到的本站图片;
// @Description 已删除的配置不导出, 掉落池中已删除物品的条目也不导出; 物品按 ID 导出, 设置只导出游戏分类且不含密钥字段
// @Accept json
// @Produce application/zip
// @Param request body ExportPackRequest true "request body"
// @Success 200 {file} file
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/packs/export [post]
func ExportPack(c *gin.Context) {
	ctx := c.Request.Context()
	var req ExportPackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	data, counts, err := doExportPack(ctx, &req)
	if err != nil {
		var packErr *PackError
		if errors.As(err, &packErr) {
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[ExportPack] export failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionPackExport, audit.TargetPack, 0, map[string]interface{}{
		"name":   strings.TrimSpace(req.Name),
		"counts": counts,
		"size":   len(data),
	})
	logger.InfoF(ctx, "[ExportPack] %d exported pack %v", operator.ID, counts)
	filename := fmt.Sprintf("blindcrate-pack-%s.zip", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/zip", data)
}

// readPackForm 读取上传的配置包和处理方式, 不符合要求时写入响应并返回 false
func readPackForm(c *gin.Context, action string) ([]byte, []PackAction, bool) {
	ctx := c.Request.Context()
	tooLarge := fmt.Sprintf(PackTooLarge, maxPackSize>>20)
	// 请求头中的大小已经超过上限时直接拒绝, 不读取请求体
	if c.Request.ContentLength > maxPackSize+packFormOverhead {
		c.JSON(http.StatusRequestEntityTooLarge, Response{ErrorMsg: tooLarge})
		return nil, nil, false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPackSize+packFormOverhead)
	if err := c.Request.ParseMultipartForm(packFormOverhead); err != nil {
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			c.JSON(http.StatusRequestEntityTooLarge, Response{ErrorMsg: tooLarge})
			return nil, nil, false
		}
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: PackRequired})
		return nil, nil, false
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: PackRequired})
		return nil, nil, false
	}
	if file.Size > maxPackSize {
		c.JSON(http.StatusRequestEntityTooLarge, Response{ErrorMsg: tooLarge})
		return nil, nil, false
	}
	var actions []PackAction
	if raw := c.PostForm("actions"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &actions); err != nil {
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: PackActionInvalid})
			return nil, nil, false
		}
	}
	f, err := file.Open()
	if err != nil {
		logger.ErrorF(ctx, "[%s] open file failed: %v", action, err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return nil, nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxPackSize+1))
	if err != nil {
		logger.ErrorF(ctx, "[%s] read file failed: %v", action, err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return nil, nil, false
	}
	return data, actions, true
}

// PreviewPack
// @Tags admin
// @Summary 预览配置包
// @Description multipart 表单, 按依赖顺序列出每一项与已有配置的比较结果: 新增、覆盖、相同或冲突(带原因), 不写入任何内容;
// @Description 在回滚的事务中按处理方式试着导入, 校验规则与导入相同; 货币、掉落池、地图、商品、榜单按代码对应, 品质按等级, 物品按 ID, 设置按设置组
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "配置包"
// @Param actions formData string false "处理方式, PackAction 的 JSON 数组, 没有提交的项按覆盖处理"
// @Success 200 {object} Response{data=PackPreview}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 413 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/packs/preview [post]
func PreviewPack(c *gin.Context) {
	ctx := c.Request.Context()
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	data, actions, ok := readPackForm(c, "PreviewPack")
	if !ok {
		return
	}
	preview, err := doPreviewPack(ctx, operator.ID, data, actions)
	if err != nil {
		var packErr *PackError
		if errors.As(err, &packErr) {
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[PreviewPack] preview failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: preview})
}

// ImportPack
// @Tags admin
// @Summary 导入配置包
// @Description multipart 表单, 在一个事务中按依赖顺序写入, 全部成功或全部回滚; 只新增或覆盖配置, 不会停用或删除已有配置, 也不改动玩家数据, 已有商品的剩余库存保持不变;
// @Description 有未跳过的冲突项时返回 409, data 为每一项的结果; 设置按版本保存, 来源为 import, 之后可以回滚
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "配置包"
// @Param actions formData string false "处理方式, PackAction 的 JSON 数组, 没有提交的项按覆盖处理"
// @Success 200 {object} Response{data=Import}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 409 {object} Response{data=[]PackEntryResult}
// @Failure 413 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/packs/import [post]
func ImportPack(c *gin.Context) {
	ctx := c.Request.Context()
	operator, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	data, actions, ok := readPackForm(c, "ImportPack")
	if !ok {
		return
	}
	record, results, err := doImportPack(ctx, operator.ID, data, actions)
	if err != nil {
		var packErr *PackError
		switch {
		case errors.As(err, &packErr) && results != nil:
			c.JSON(http.StatusConflict, Response{ErrorMsg: err.Error(), Data: results})
		case errors.As(err, &packErr):
			c.JSON(http.StatusBadRequest, Response{ErrorMsg: err.Error()})
		default:
			logger.ErrorF(ctx, "[ImportPack] import failed: %v", err)
			c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		}
		return
	}
	audit.Record(c, operator.ID, audit.ScopeAdmin, audit.ActionPackImport, audit.TargetPack, record.ID, map[string]interface{}{
		"name":    record.Name,
		"author":  record.Author,
		"version": record.Version,
		"digest":  record.Digest,
		"size":    record.Size,
		"created": record.Created,
		"updated": record.Updated,
		"skipped": record.Skipped,
	})
	logger.InfoF(ctx, "[ImportPack] %d imported pack %d %s: %d created, %d updated, %d skipped",
		operator.ID, record.ID, record.Name, record.Created, record.Updated, record.Skipped)
	c.JSON(http.StatusOK, Response{Data: record})
}

// ListPackImportsRequest 导入记录查询参数
type ListPackImportsRequest struct {
	Current int `json:"current" form:"current" binding:"min=1"`
	Size    int `json:"size" form:"size" binding:"min=1,max=100"`
}

// ListPackImportsResponseData 导入记录分页结果
type ListPackImportsResponseData struct {
	Total   int64    `json:"total"`
	Imports []Import `json:"imports"` // 不含每一项的结果
}

// ListPackImports
// @Tags admin
// @Summary 获取导入记录列表
// @Description 按导入时间从晚到早排列, 每一项的结果见导入记录详情
// @Produce json
// @Param request query ListPackImportsRequest true "request query"
// @Success 200 {object} Response{data=ListPackImportsResponseData}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/packs/imports [get]
func ListPackImports(c *gin.Context) {
	ctx := c.Request.Context()
	req := &ListPackImportsRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	total, imports, err := QueryPackImports(ctx, req)
	if err != nil {
		logger.ErrorF(ctx, "[ListPackImports] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: ListPackImportsResponseData{Total: total, Imports: imports}})
}

// GetPackImport
// @Tags admin
// @Summary 获取导入记录详情
// @Description 包括每一项的处理结果
// @Produce json
// @Param id path int true "import id"
// @Success 200 {object} Response{data=Import}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/admin/packs/imports/{id} [get]
func GetPackImport(c *gin.Context) {
	ctx := c.Request.Context()
	importID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: auth.InvalidParams})
		return
	}
	record, err := queryPackImport(ctx, importID)
	if err != nil {
		if err.Error() == ImportNotFound {
			c.JSON(http.StatusNotFound, Response{ErrorMsg: err.Error()})
			return
		}
		logger.ErrorF(ctx, "[GetPackImport] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{ErrorMsg: auth.InternalError})
		return
	}
	c.JSON(http.StatusOK, Response{Data: record})
}
