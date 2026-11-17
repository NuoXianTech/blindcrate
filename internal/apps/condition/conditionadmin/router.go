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
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/reply"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/gin-gonic/gin"
)

// ListConditionTypesResponseData 全部条件类型
type ListConditionTypesResponseData struct {
	Types []condition.TypeView `json:"types"`
}

// ListConditionTypes
// @Tags admin
// @Summary 获取全部条件类型
// @Description 包含每种条件的参数表单描述, 用于条件编辑器; 下拉框和输入建议的可选值(货币、物品、统计范围等)已经填好
// @Produce json
// @Success 200 {object} reply.Response{data=ListConditionTypesResponseData}
// @Failure 401 {object} reply.Response
// @Failure 403 {object} reply.Response
// @Failure 500 {object} reply.Response
// @Router /api/v1/admin/conditions/types [get]
func ListConditionTypes(c *gin.Context) {
	ctx := c.Request.Context()
	types, err := condition.Types(db.DB(ctx))
	if err != nil {
		reply.Fail(c, err, "[ListConditionTypes] query failed")
		return
	}
	reply.OK(c, ListConditionTypesResponseData{Types: types})
}
