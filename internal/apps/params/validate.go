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

package params

import (
	"errors"
	"reflect"

	"github.com/NuoXianTech/blindcrate/internal/apps/fault"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhtranslations "github.com/go-playground/validator/v10/translations/zh"
)

var (
	validate = validator.New(validator.WithRequiredStructEnabled())
	trans, _ = ut.New(zh.New()).GetTranslator("zh")
)

func init() {
	// 规则写法与 gin 的请求参数校验一致, 校验失败的提示使用字段的 label
	validate.SetTagName("binding")
	validate.RegisterTagNameFunc(func(sf reflect.StructField) string {
		return sf.Tag.Get("label")
	})
	if err := zhtranslations.RegisterDefaultTranslations(validate, trans); err != nil {
		panic(err)
	}

	// 中文翻译中没有 http_url
	if err := validate.RegisterTranslation("http_url", trans, func(t ut.Translator) error {
		return t.Add("http_url", "{0}必须是 http(s) 链接", false)
	}, func(t ut.Translator, fe validator.FieldError) string {
		msg, _ := t.T("http_url", fe.Field())
		return msg
	}); err != nil {
		panic(err)
	}
}

// ValidateStruct 按字段的 binding 规则校验结构体, 校验失败时返回第一个错误的中文提示, 提示中使用字段的 label
//
// 保存在 JSONB 中的配置(设置组、条件参数、地图规则等)都用它校验, 规则写法一致。
func ValidateStruct(v any) error {
	if err := validate.Struct(v); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			return fault.Invalid("%s", errs[0].Translate(trans))
		}
		return err
	}
	return nil
}
