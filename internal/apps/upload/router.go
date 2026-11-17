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

package upload

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/gin-gonic/gin"
)

// Response 统一的 upload 响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// CreateUpload
// @Tags upload
// @Summary 上传图片
// @Description multipart 表单, purpose 为用途: avatar 头像(登录用户), item 物品图片、venue 地图封面、shop 商品图片(管理员);
// @Description 按内容识别格式, 按用途缩小尺寸并重新编码后保存, 返回的地址保存到头像、物品等的图片地址中; 大小上限和允许的格式见公开设置 upload
// @Accept multipart/form-data
// @Produce json
// @Param purpose formData string true "用途" Enums(avatar, item, venue, shop)
// @Param file formData file true "图片"
// @Success 200 {object} Response{data=View}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 413 {object} Response
// @Failure 429 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/uploads [post]
func CreateUpload(c *gin.Context) {
	ctx := c.Request.Context()
	user, ok := auth.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{ErrorMsg: auth.UnAuthorized})
		return
	}
	maxSize, err := MaxSize(ctx)
	if err != nil {
		reply.Fail(c, err, "[CreateUpload] get settings failed")
		return
	}
	tooLarge := fmt.Sprintf(FileTooLarge, maxSize>>10)
	// 请求头中的大小已经超过上限时直接拒绝, 不读取请求体(客户端等待 100 Continue 时不会再上传)
	if c.Request.ContentLength > maxSize+multipartOverhead {
		c.JSON(http.StatusRequestEntityTooLarge, Response{ErrorMsg: tooLarge})
		return
	}
	// 读取表单前限制请求体大小, 超大的请求不会写入临时文件
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize+multipartOverhead)
	if err := c.Request.ParseMultipartForm(maxSize + multipartOverhead); err != nil {
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			c.JSON(http.StatusRequestEntityTooLarge, Response{ErrorMsg: tooLarge})
			return
		}
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: FileRequired})
		return
	}
	purpose := c.PostForm("purpose")
	if err := CheckPurpose(ctx, purpose, user.IsAdmin); err != nil {
		reply.Fail(c, err, "[CreateUpload] check purpose failed")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{ErrorMsg: FileRequired})
		return
	}
	if file.Size > maxSize {
		c.JSON(http.StatusRequestEntityTooLarge, Response{ErrorMsg: tooLarge})
		return
	}
	f, err := file.Open()
	if err != nil {
		reply.Fail(c, err, "[CreateUpload] open file failed")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxSize+1))
	if err != nil {
		reply.Fail(c, err, "[CreateUpload] read file failed")
		return
	}
	view, err := Save(ctx, &SaveRequest{UserID: user.ID, IsAdmin: user.IsAdmin, Purpose: purpose, Data: data})
	if err != nil {
		reply.Fail(c, err, "[CreateUpload] save failed")
		return
	}
	c.JSON(http.StatusOK, Response{Data: view})
}
