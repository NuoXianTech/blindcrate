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

package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhtranslations "github.com/go-playground/validator/v10/translations/zh"
	"gorm.io/gorm/clause"
)

// Field 设置项的描述, 后台据此自动生成表单
type Field struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Help   string `json:"help"`
	Type   string `json:"type"`  // string / bool / int
	Input  string `json:"input"` // text / textarea / password / switch / number
	Secret bool   `json:"secret"`
	Public bool   `json:"public"`
	index  int
}

type group struct {
	key         string
	label       string
	description string
	fields      []Field
	defaults    func() any // 返回指向默认值的指针
}

// GroupView 管理后台看到的设置组, 包含表单描述和当前值
type GroupView struct {
	Key           string                 `json:"key"`
	Label         string                 `json:"label"`
	Description   string                 `json:"description"`
	Fields        []Field                `json:"fields"`
	Value         map[string]interface{} `json:"value"`          // 密钥字段固定为空字符串
	SecretsSet    map[string]bool        `json:"secrets_set"`    // 密钥字段是否已设置
	SecretsBroken []string               `json:"secrets_broken"` // 无法解密的密钥字段, 需要重新填写或清空才能保存
	UpdatedBy     uint64                 `json:"updated_by"`
	UpdatedAt     *time.Time             `json:"updated_at"` // 未保存过为空
}

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

// newGroup 根据结构体 tag 生成设置组定义
func newGroup[T any](key, label, description string, defaults func() *T) *group {
	g := &group{key: key, label: label, description: description, defaults: func() any { return defaults() }}
	typ := reflect.TypeFor[T]()
	for i := range typ.NumField() {
		sf := typ.Field(i)
		f := Field{
			Key:    strings.Split(sf.Tag.Get("json"), ",")[0],
			Label:  sf.Tag.Get("label"),
			Help:   sf.Tag.Get("help"),
			Input:  sf.Tag.Get("input"),
			Secret: sf.Tag.Get("secret") == "true",
			Public: sf.Tag.Get("public") == "true",
			index:  i,
		}
		switch sf.Type.Kind() {
		case reflect.String:
			f.Type = "string"
			if f.Secret {
				f.Input = "password"
			} else if f.Input == "" {
				f.Input = "text"
			}
		case reflect.Bool:
			f.Type, f.Input = "bool", "switch"
		case reflect.Int:
			f.Type, f.Input = "int", "number"
		default:
			panic(fmt.Sprintf("settings: %s.%s: unsupported type %s", key, sf.Name, sf.Type))
		}
		g.fields = append(g.fields, f)
	}
	return g
}

func findGroup(key string) *group {
	for _, g := range groups {
		if g.key == key {
			return g
		}
	}
	return nil
}

func (g *group) field(key string) *Field {
	for i := range g.fields {
		if g.fields[i].Key == key {
			return &g.fields[i]
		}
	}
	return nil
}

// get 读取一个设置组, 未保存过时返回默认值
func get[T any](ctx context.Context, key string) (*T, error) {
	var row Setting
	if err := db.DB(ctx).Where("key = ?", key).Limit(1).Find(&row).Error; err != nil {
		return nil, err
	}
	value, _ := findGroup(key).decode(ctx, &row)
	return value.(*T), nil
}

// QueryGroups 获取全部设置组
func QueryGroups(ctx context.Context) ([]GroupView, error) {
	rows, err := queryRows(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]GroupView, 0, len(groups))
	for _, g := range groups {
		value, broken := g.decode(ctx, rows[g.key])
		views = append(views, g.view(rows[g.key], value, broken))
	}
	return views, nil
}

// QueryPublic 获取全部公开字段, 按设置组分组
func QueryPublic(ctx context.Context) (map[string]map[string]interface{}, error) {
	rows, err := queryRows(ctx)
	if err != nil {
		return nil, err
	}
	result := map[string]map[string]interface{}{}
	for _, g := range groups {
		value, _ := g.decode(ctx, rows[g.key])
		if values := g.values(value, true); len(values) > 0 {
			result[g.key] = values
		}
	}
	return result, nil
}

// queryRows 读取全部设置组, 未保存过的设置组为空记录
func queryRows(ctx context.Context) (map[string]*Setting, error) {
	var rows []Setting
	if err := db.DB(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := map[string]*Setting{}
	for _, g := range groups {
		result[g.key] = &Setting{}
	}
	for i := range rows {
		result[rows[i].Key] = &rows[i]
	}
	return result, nil
}

// Update 保存一个设置组, 返回保存后的设置组和本次修改的字段
//
// 合并规则见 group.merge, 没有任何字段变化时不写库; 多人同时修改时以后保存的为准。
// 返回的错误都可以直接展示: 数据库错误记录日志后返回 SaveFailed, 其余为 GroupNotFound 或校验失败。
func Update(ctx context.Context, operatorID uint64, key string, raw json.RawMessage, clearSecrets []string) (*GroupView, map[string]interface{}, error) {
	g := findGroup(key)
	if g == nil {
		return nil, nil, errors.New(GroupNotFound)
	}

	var row Setting
	if err := db.DB(ctx).Where("key = ?", key).Limit(1).Find(&row).Error; err != nil {
		logger.ErrorF(ctx, "[Settings] query %s failed: %s", key, err.Error())
		return nil, nil, errors.New(SaveFailed)
	}
	cur, broken := g.decode(ctx, &row)
	next, err := g.merge(cur, broken, raw, clearSecrets)
	if err != nil {
		return nil, nil, err
	}
	changes := g.diff(cur, broken, next)
	if len(changes) == 0 {
		view := g.view(&row, cur, broken)
		return &view, changes, nil
	}

	stored, err := g.encode(next)
	if err != nil {
		if err.Error() == SecretKeyMissing {
			return nil, nil, err
		}
		logger.ErrorF(ctx, "[Settings] encode %s failed: %s", key, err.Error())
		return nil, nil, errors.New(SaveFailed)
	}
	row.Key, row.Value, row.UpdatedBy, row.UpdatedAt = key, stored, operatorID, time.Now()
	if err = db.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_by", "updated_at"}),
	}).Create(&row).Error; err != nil {
		logger.ErrorF(ctx, "[Settings] save %s failed: %s", key, err.Error())
		return nil, nil, errors.New(SaveFailed)
	}
	view := g.view(&row, next, nil)
	return &view, changes, nil
}

func (g *group) view(row *Setting, value any, broken []string) GroupView {
	view := GroupView{
		Key:           g.key,
		Label:         g.label,
		Description:   g.description,
		Fields:        g.fields,
		Value:         g.values(value, false),
		SecretsSet:    g.secretsSet(value),
		SecretsBroken: broken,
		UpdatedBy:     row.UpdatedBy,
	}
	if row.Key != "" {
		view.UpdatedAt = &row.UpdatedAt
	}
	return view
}

// merge 把提交的 JSON 合并到当前值上并校验
//
// 没有提交的字段保持当前值; 密钥字段提交空字符串表示不修改, 需要清空时放进 clearSecrets;
// broken 为当前值中无法解密的密钥字段, 这些字段必须重新填写或清空, 否则保存会把原密文覆盖为空。
func (g *group) merge(cur any, broken []string, raw json.RawMessage, clearSecrets []string) (any, error) {
	next := clone(cur)
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(next); err != nil {
		return nil, errors.New(ValueInvalid)
	}
	for _, key := range clearSecrets {
		if f := g.field(key); f == nil || !f.Secret {
			return nil, errors.New(ValueInvalid)
		}
	}

	cv, nv := reflect.ValueOf(cur).Elem(), reflect.ValueOf(next).Elem()
	for _, f := range g.fields {
		fv := nv.Field(f.index)
		switch {
		case !f.Secret:
		case slices.Contains(clearSecrets, f.Key):
			fv.SetString("")
		case fv.String() != "":
		case slices.Contains(broken, f.Key):
			return nil, fmt.Errorf(SecretBroken, f.Label)
		default:
			fv.SetString(cv.Field(f.index).String())
		}
	}

	if err := validate.Struct(next); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			return nil, errors.New(errs[0].Translate(trans))
		}
		return nil, err
	}
	return next, nil
}

// diff 本次修改了哪些字段, 用于操作日志, 密钥字段不记录内容
//
// 无法解密的密钥字段经过 merge 后一定是重新填写或清空, 都算修改, 需要写库覆盖原密文。
func (g *group) diff(cur any, broken []string, next any) map[string]interface{} {
	changes := map[string]interface{}{}
	cv, nv := reflect.ValueOf(cur).Elem(), reflect.ValueOf(next).Elem()
	for _, f := range g.fields {
		oldValue, newValue := cv.Field(f.index).Interface(), nv.Field(f.index).Interface()
		if oldValue == newValue && !slices.Contains(broken, f.Key) {
			continue
		}
		if f.Secret {
			if newValue == "" {
				changes[f.Key] = "<cleared>"
			} else {
				changes[f.Key] = "<updated>"
			}
			continue
		}
		changes[f.Key] = map[string]interface{}{"old": oldValue, "new": newValue}
	}
	return changes
}

// encode 序列化为存库的 JSON, 密钥字段加密
func (g *group) encode(v any) (string, error) {
	stored := clone(v)
	rv := reflect.ValueOf(stored).Elem()
	for _, f := range g.fields {
		fv := rv.Field(f.index)
		if !f.Secret || fv.String() == "" {
			continue
		}
		ciphertext, err := encryptSecret(fv.String(), g.key+"."+f.Key)
		if err != nil {
			return "", err
		}
		fv.SetString(ciphertext)
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// decode 把库里的 JSON 解析到默认值之上并解密密钥字段, 返回设置值和无法解密的密钥字段
//
// 未保存过时为默认值; 库里没有的字段(新加的字段)保持默认值, 多余的字段(已删除的字段)忽略;
// 无法解密的密钥字段(如 app.secret_key 被修改)按空值使用, 但会阻止保存, 避免把原密文覆盖掉。
func (g *group) decode(ctx context.Context, row *Setting) (any, []string) {
	v := g.defaults()
	if row.Key == "" {
		return v, nil
	}
	if err := json.Unmarshal([]byte(row.Value), v); err != nil {
		logger.ErrorF(ctx, "[Settings] decode %s failed: %s", g.key, err.Error())
	}
	var broken []string
	rv := reflect.ValueOf(v).Elem()
	for _, f := range g.fields {
		fv := rv.Field(f.index)
		if !f.Secret || fv.String() == "" {
			continue
		}
		plaintext, err := decryptSecret(fv.String(), g.key+"."+f.Key)
		if err != nil {
			logger.ErrorF(ctx, "[Settings] decrypt %s.%s failed: %s", g.key, f.Key, err.Error())
			broken = append(broken, f.Key)
		}
		fv.SetString(plaintext)
	}
	return v, broken
}

// values 转为 字段名 -> 值, 密钥字段返回空字符串; onlyPublic 为 true 时只返回公开字段
func (g *group) values(v any, onlyPublic bool) map[string]interface{} {
	result := map[string]interface{}{}
	rv := reflect.ValueOf(v).Elem()
	for _, f := range g.fields {
		switch {
		case onlyPublic && !f.Public:
		case f.Secret:
			result[f.Key] = ""
		default:
			result[f.Key] = rv.Field(f.index).Interface()
		}
	}
	return result
}

// secretsSet 各密钥字段是否已设置
func (g *group) secretsSet(v any) map[string]bool {
	result := map[string]bool{}
	rv := reflect.ValueOf(v).Elem()
	for _, f := range g.fields {
		if f.Secret {
			result[f.Key] = rv.Field(f.index).String() != ""
		}
	}
	return result
}

// clone 浅拷贝设置值, 设置结构体只有基础类型字段, 浅拷贝即可
func clone(v any) any {
	dst := reflect.New(reflect.TypeOf(v).Elem())
	dst.Elem().Set(reflect.ValueOf(v).Elem())
	return dst.Interface()
}
