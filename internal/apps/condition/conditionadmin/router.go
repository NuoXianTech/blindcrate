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

package conditionadmin

import (
	"net/http"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/gin-gonic/gin"
)

// Response 统一的响应壳, 保持与其他 app 一致
type Response struct {
	ErrorMsg string      `json:"error_msg"`
	Data     interface{} `json:"data"`
}

// ListConditionTypesResponseData 全部条件类型
type ListConditionTypesResponseData struct {
	Types []condition.TypeView `json:"types"`
}

// ListConditionTypes
// @Tags admin
// @Summary 获取全部条件类型
// @Description 包含每种条件的参数表单描述, 用于条件编辑器; input 为 currency、item、scope 的字段需要用选择器填写
// @Produce json
// @Success 200 {object} Response{data=ListConditionTypesResponseData}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Router /api/v1/admin/conditions/types [get]
func ListConditionTypes(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Data: ListConditionTypesResponseData{Types: condition.Types()}})
}
