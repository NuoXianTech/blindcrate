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
	"regexp"
	"slices"
	"strings"

	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhtranslations "github.com/go-playground/validator/v10/translations/zh"
	"gorm.io/gorm"
)

// Field 设置项的描述, 后台据此自动生成表单
type Field struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Help    string   `json:"help"`
	Type    string   `json:"type"`              // string / bool / int / int_list
	Input   string   `json:"input"`             // text / textarea / password / select / switch / number / list
	Options []Option `json:"options,omitempty"` // 下拉框的可选值
	Ref     string   `json:"ref,omitempty"`     // 引用其他模块的数据, 如 currency; 可选值在读取时由 RegisterRef 注册的函数生成
	Unit    string   `json:"unit,omitempty"`    // 整数的单位: bp / second / millisecond
	Secret  bool     `json:"secret"`
	Public  bool     `json:"public"`
	index   int
}

// Option 下拉框的一个可选值
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Category 设置组的分类
type Category struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Group 注册设置组时的定义
type Group[T any] struct {
	Key         string    // 小写字母、数字和下划线, 改名会丢失已保存的值
	Label       string    // 后台展示的名称
	Description string    // 后台展示的说明
	Category    string    // CategorySystem 或 CategoryGame
	Sort        int       // 同一分类中从小到大排列
	Defaults    func() *T // 默认值, 未保存过的设置组和新加的字段使用默认值
}

// Handle 已注册的设置组, 用于读取当前值
type Handle[T any] struct {
	g *group
}

// group 设置组定义, 由 Register 根据结构体 tag 生成
type group struct {
	key         string
	label       string
	description string
	category    string
	sort        int
	fields      []Field
	defaults    func() any // 返回指向默认值的指针
}

var (
	// Categories 全部分类, 按展示顺序
	Categories = []Category{
		{Key: CategorySystem, Label: "系统"},
		{Key: CategoryGame, Label: "游戏"},
	}

	// groups 已注册的设置组, 按分类和 Sort 排列
	groups     []*group
	units      = []string{"bp", "second", "millisecond"}
	keyPattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

	// refs 引用字段的可选值, 按引用名称, 只在启动前写入
	refs = map[string]RefOptions{}

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

// RefOptions 生成引用字段的可选值, 如全部启用的货币
type RefOptions func(tx *gorm.DB) ([]Option, error)

// RegisterRef 注册一种引用, 带 ref tag 的字段在后台显示为下拉框, 保存时只能选可选值; 需要在启动前调用, 重复注册时 panic
func RegisterRef(name string, options RefOptions) {
	if _, ok := refs[name]; ok {
		panic(fmt.Sprintf("settings: ref %s already registered", name))
	}
	refs[name] = options
}

// refOptions 引用字段的可选值, 引用没有注册时返回错误
func refOptions(tx *gorm.DB, name string) ([]Option, error) {
	options, ok := refs[name]
	if !ok {
		return nil, fmt.Errorf("settings: ref %s not registered", name)
	}
	return options(tx)
}

// Register 注册一个设置组, 在包级变量中调用, key 重复、分类未知或字段类型不支持时 panic
func Register[T any](def Group[T]) *Handle[T] {
	if !keyPattern.MatchString(def.Key) {
		panic(fmt.Sprintf("settings: invalid group key %q", def.Key))
	}
	if findGroup(def.Key) != nil {
		panic(fmt.Sprintf("settings: group %s already registered", def.Key))
	}
	if categoryIndex(def.Category) < 0 {
		panic(fmt.Sprintf("settings: group %s: unknown category %q", def.Key, def.Category))
	}
	g := newGroup(def)
	groups = append(groups, g)
	slices.SortStableFunc(groups, func(a, b *group) int {
		if c := categoryIndex(a.category) - categoryIndex(b.category); c != 0 {
			return c
		}
		return a.sort - b.sort
	})
	return &Handle[T]{g: g}
}

// Key 设置组的 key
func (h *Handle[T]) Key() string {
	return h.g.key
}

// categoryIndex 分类的展示顺序, 未知分类返回 -1
func categoryIndex(category string) int {
	return slices.IndexFunc(Categories, func(c Category) bool { return c.Key == category })
}

// newGroup 根据结构体 tag 生成设置组定义
func newGroup[T any](def Group[T]) *group {
	g := &group{
		key:         def.Key,
		label:       def.Label,
		description: def.Description,
		category:    def.Category,
		sort:        def.Sort,
		defaults:    func() any { return def.Defaults() },
	}
	typ := reflect.TypeFor[T]()
	for i := range typ.NumField() {
		sf := typ.Field(i)
		name := def.Key + "." + sf.Name
		f := Field{
			Key:     strings.Split(sf.Tag.Get("json"), ",")[0],
			Label:   sf.Tag.Get("label"),
			Help:    sf.Tag.Get("help"),
			Input:   sf.Tag.Get("input"),
			Options: parseOptions(name, sf.Tag.Get("options")),
			Ref:     sf.Tag.Get("ref"),
			Unit:    sf.Tag.Get("unit"),
			Secret:  sf.Tag.Get("secret") == "true",
			Public:  sf.Tag.Get("public") == "true",
			index:   i,
		}
		switch kind := sf.Type.Kind(); {
		case kind == reflect.String:
			f.Type = "string"
			switch {
			case f.Secret:
				f.Input = "password"
			case len(f.Options) > 0 || f.Ref != "":
				f.Input = "select"
			case f.Input == "":
				f.Input = "text"
			}
		case kind == reflect.Bool:
			f.Type, f.Input = "bool", "switch"
		case isInt(kind):
			f.Type, f.Input = "int", "number"
		case kind == reflect.Slice && isInt(sf.Type.Elem().Kind()):
			f.Type, f.Input = "int_list", "list"
		default:
			panic(fmt.Sprintf("settings: %s: unsupported type %s", name, sf.Type))
		}
		switch {
		case f.Secret && (f.Type != "string" || len(f.Options) > 0 || f.Ref != ""):
			panic(fmt.Sprintf("settings: %s: only plain strings can be secret", name))
		case (len(f.Options) > 0 || f.Ref != "") && f.Type != "string":
			panic(fmt.Sprintf("settings: %s: only strings can have options", name))
		case len(f.Options) > 0 && f.Ref != "":
			panic(fmt.Sprintf("settings: %s: options and ref are exclusive", name))
		case f.Unit != "" && (!slices.Contains(units, f.Unit) || (f.Type != "int" && f.Type != "int_list")):
			panic(fmt.Sprintf("settings: %s: unsupported unit %q", name, f.Unit))
		}
		g.fields = append(g.fields, f)
	}
	return g
}

// isInt 是否为支持的整数类型
func isInt(kind reflect.Kind) bool {
	return kind == reflect.Int || kind == reflect.Int32 || kind == reflect.Int64
}

// parseOptions 解析 options tag, 格式为 "值:名称,值:名称"
func parseOptions(name, tag string) []Option {
	if tag == "" {
		return nil
	}
	var options []Option
	for _, part := range strings.Split(tag, ",") {
		value, label, ok := strings.Cut(part, ":")
		if !ok || value == "" || label == "" {
			panic(fmt.Sprintf("settings: %s: invalid options %q", name, tag))
		}
		options = append(options, Option{Value: value, Label: label})
	}
	return options
}

// findGroup 按 key 查找设置组, 不存在时返回 nil
func findGroup(key string) *group {
	for _, g := range groups {
		if g.key == key {
			return g
		}
	}
	return nil
}

// field 按字段名查找字段, 不存在时返回 nil
func (g *group) field(key string) *Field {
	for i := range g.fields {
		if g.fields[i].Key == key {
			return &g.fields[i]
		}
	}
	return nil
}

// formFields 后台表单的字段, 引用字段带上当前的可选值
func (g *group) formFields(tx *gorm.DB) ([]Field, error) {
	fields := slices.Clone(g.fields)
	for i := range fields {
		if fields[i].Ref == "" {
			continue
		}
		options, err := refOptions(tx, fields[i].Ref)
		if err != nil {
			return nil, err
		}
		fields[i].Options = options
	}
	return fields, nil
}

// view 生成管理后台看到的设置组, fields 见 formFields
func (g *group) view(fields []Field, row *Setting, value any, broken []string) GroupView {
	view := GroupView{
		Key:           g.key,
		Label:         g.label,
		Description:   g.description,
		Category:      g.category,
		Fields:        fields,
		Value:         g.values(value, false),
		SecretsSet:    g.secretsSet(value),
		SecretsBroken: broken,
		Version:       row.Version,
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
func (g *group) merge(tx *gorm.DB, cur any, broken []string, raw json.RawMessage, clearSecrets []string) (any, error) {
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

	if err := g.validate(tx, next); err != nil {
		return nil, err
	}
	return next, nil
}

// validate 按 binding 规则校验, 并检查下拉框和引用字段的值是否在可选值中
func (g *group) validate(tx *gorm.DB, v any) error {
	if err := ValidateStruct(v); err != nil {
		return err
	}
	rv := reflect.ValueOf(v).Elem()
	for _, f := range g.fields {
		if len(f.Options) == 0 && f.Ref == "" {
			continue
		}
		options := f.Options
		if f.Ref != "" {
			var err error
			if options, err = refOptions(tx, f.Ref); err != nil {
				return err
			}
		}
		value := rv.Field(f.index).String()
		if !slices.ContainsFunc(options, func(o Option) bool { return o.Value == value }) {
			return fmt.Errorf(OptionInvalid, f.Label)
		}
	}
	return nil
}

// ValidateStruct 按字段的 binding 规则校验结构体, 校验失败时返回第一个错误的中文提示, 提示中使用字段的 label
//
// 其他模块的 JSONB 配置(如条件参数)也用它校验, 规则写法与设置组一致。
func ValidateStruct(v any) error {
	if err := validate.Struct(v); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			return errors.New(errs[0].Translate(trans))
		}
		return err
	}
	return nil
}

// diff 本次修改了哪些字段, 用于操作日志和版本记录, 密钥字段不记录内容
//
// 无法解密的密钥字段经过 merge 后一定是重新填写或清空, 都算修改, 需要写库覆盖原密文。
func (g *group) diff(cur any, broken []string, next any) map[string]interface{} {
	changes := map[string]interface{}{}
	cv, nv := reflect.ValueOf(cur).Elem(), reflect.ValueOf(next).Elem()
	for _, f := range g.fields {
		oldValue, newValue := cv.Field(f.index).Interface(), nv.Field(f.index).Interface()
		if reflect.DeepEqual(oldValue, newValue) && !slices.Contains(broken, f.Key) {
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

// decode 读取当前值, 返回设置值和无法解密的密钥字段, 未保存过时为默认值
func (g *group) decode(ctx context.Context, row *Setting) (any, []string) {
	if row.Key == "" {
		return g.defaults(), nil
	}
	return g.decodeValue(ctx, row.Value)
}

// decodeValue 把存库的 JSON 解析到默认值之上并解密密钥字段, 也用于解析版本快照
//
// 库里没有的字段(新加的字段)保持默认值, 多余的字段(已删除的字段)忽略;
// 无法解密的密钥字段(如 app.secret_key 被修改)按空值使用, 但会阻止保存, 避免把原密文覆盖掉。
func (g *group) decodeValue(ctx context.Context, raw string) (any, []string) {
	v := g.defaults()
	if err := json.Unmarshal([]byte(raw), v); err != nil {
		logger.ErrorF(ctx, "[Settings] decode %s failed: %v", g.key, err)
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
			logger.ErrorF(ctx, "[Settings] decrypt %s.%s failed: %v", g.key, f.Key, err)
			broken = append(broken, f.Key)
		}
		fv.SetString(plaintext)
	}
	return v, broken
}

// values 转为 字段名 -> 值, 密钥字段返回空字符串, 整数列表为空时返回空数组; onlyPublic 为 true 时只返回公开字段
func (g *group) values(v any, onlyPublic bool) map[string]interface{} {
	result := map[string]interface{}{}
	rv := reflect.ValueOf(v).Elem()
	for _, f := range g.fields {
		fv := rv.Field(f.index)
		switch {
		case onlyPublic && !f.Public:
		case f.Secret:
			result[f.Key] = ""
		case fv.Kind() == reflect.Slice && fv.IsNil():
			result[f.Key] = reflect.MakeSlice(fv.Type(), 0, 0).Interface()
		default:
			result[f.Key] = fv.Interface()
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

// clone 复制设置值, 切片字段也复制一份, 修改副本不会影响原值
func clone(v any) any {
	src := reflect.ValueOf(v).Elem()
	dst := reflect.New(src.Type())
	dst.Elem().Set(src)
	for i := range src.NumField() {
		if fv := src.Field(i); fv.Kind() == reflect.Slice && !fv.IsNil() {
			copied := reflect.MakeSlice(fv.Type(), fv.Len(), fv.Len())
			reflect.Copy(copied, fv)
			dst.Elem().Field(i).Set(copied)
		}
	}
	return dst.Interface()
}
