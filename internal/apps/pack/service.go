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
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/upload"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"github.com/NuoXianTech/blindcrate/internal/storage"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// PackManifest 配置包的说明, 即包中的 manifest.json
type PackManifest struct {
	Format     string    `json:"format"`  // 固定为 blindcrate-pack
	Version    int       `json:"version"` // 格式版本
	Name       string    `json:"name"`
	Author     string    `json:"author"`
	ExportedAt time.Time `json:"exported_at"`
	Sections   []string  `json:"sections"` // 包含的配置种类, 见 RegisterSection
}

// PackEntryResult 配置包中一项配置的比较或导入结果
type PackEntryResult struct {
	Section string `json:"section"`          // 配置种类, 见 RegisterSection
	Key     string `json:"key"`              // 包中的代码, 品质为等级, 物品为 ID, 设置为设置组的 key
	Name    string `json:"name"`             // 包中的名称
	Status  string `json:"status"`           // 见 PackStatus 开头的常量, 跳过的项为与已有配置比较的结果
	Action  string `json:"action"`           // 见 PackAction 开头的常量
	Code    string `json:"code,omitempty"`   // 改名导入时的新代码
	Reason  string `json:"reason,omitempty"` // 无法导入的原因
}

// PackAction 导入时一项配置的处理方式, 没有提交的项按覆盖处理
type PackAction struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Action  string `json:"action"` // 见 PackAction 开头的常量
	Code    string `json:"code"`   // 改名导入时的新代码
}

// PackPreview 配置包的预览, 按依赖顺序列出每一项的比较结果
type PackPreview struct {
	Manifest PackManifest      `json:"manifest"`
	Digest   string            `json:"digest"` // 文件的 sha256
	Size     int64             `json:"size"`
	Entries  []PackEntryResult `json:"entries"`
}

// 包中每种配置的一项, 字段与后台保存时提交的内容相同; 品质、地图、商品和榜单直接使用保存时的请求

// pack 读取后的配置包
type pack struct {
	manifest PackManifest
	digest   string
	size     int64
	contents map[string]Content   // 按配置种类, 只有 manifest 中列出的种类
	images   map[string]*zip.File // 按包中的路径, 如 images/item/ab/abcd.png
}

// PackError 配置包或处理方式不符合要求, 可以直接展示
type PackError struct {
	Reason string
}

func (e *PackError) Error() string { return e.Reason }

func packErrorf(format string, args ...any) error {
	return &PackError{Reason: fmt.Sprintf(format, args...)}
}

// packUpgrades 把旧格式版本的文件升级到下一个版本, 按版本号; 参数为包中各配置种类的 JSON
var packUpgrades = map[int]func(files map[string][]byte) error{}

// packLockKey 导入和预览在这个咨询锁上排队, 同一时间只有一个导入
const packLockKey = "blindcrate:pack:import"

// packSavePoint 写入每一项前的保存点, 失败时回滚到这里继续处理下一项
const packSavePoint = "pack_entry"

// errPackRollback 预览结束后回滚事务
var errPackRollback = errors.New("pack preview rollback")

// checkPackNames 去掉配置包名称和作者首尾的空白并校验
func checkPackNames(name, author string) (string, string, error) {
	name, author = strings.TrimSpace(name), strings.TrimSpace(author)
	if n := utf8.RuneCountInString(name); n < 1 || n > 64 || utf8.RuneCountInString(author) > 64 {
		return "", "", packErrorf(PackNameInvalid)
	}
	return name, author, nil
}

// orderPackSections 校验配置种类并按依赖顺序排列, 去掉重复的种类
func orderPackSections(sections []string) ([]string, error) {
	if len(sections) == 0 {
		return nil, packErrorf(PackSectionsRequired)
	}
	for _, section := range sections {
		if sectionOf(section) == nil {
			return nil, packErrorf(PackSectionInvalid, section)
		}
	}
	ordered := make([]string, 0, len(sections))
	for _, section := range sectionList() {
		if slices.Contains(sections, section.Code) {
			ordered = append(ordered, section.Code)
		}
	}
	return ordered, nil
}

// packImage 图片地址在包中的写法: 本站存储的文件写成 images/<key>, 其他地址原样保留
func packImage(url string) string {
	if key, ok := storage.KeyOf(url); ok {
		return packImageDir + key
	}
	return url
}

// packExporter 导出时收集引用到的上传图片
type packExporter struct {
	ctx    context.Context
	store  storage.Storage
	images map[string][]byte // 按包中的路径
	size   int               // 图片的总大小
	err    error             // 读取图片时的第一个错误
}

// image 本站存储的图片打包进 images/ 并返回包中的路径; 文件已不存在时清空地址, 其他地址原样保留
func (ex *packExporter) image(url string) string {
	path := packImage(url)
	key, ok := strings.CutPrefix(path, packImageDir)
	if !ok || ex.err != nil {
		return path
	}
	if _, ok := ex.images[path]; ok {
		return path
	}
	if ex.store == nil {
		if ex.store, ex.err = storage.Default(); ex.err != nil {
			return path
		}
	}
	rc, err := ex.store.Open(ex.ctx, key)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			logger.WarnF(ex.ctx, "[Pack] image %s not found, exported without it", key)
			return ""
		}
		ex.err = err
		return path
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, maxPackImageSize+1))
	if err != nil {
		ex.err = err
		return path
	}
	if len(data) > maxPackImageSize {
		logger.WarnF(ex.ctx, "[Pack] image %s is too large, exported without it", key)
		return ""
	}
	// 图片超过包的大小上限时提前结束, 不再读取
	if ex.size += len(data); ex.size > maxPackSize {
		ex.err = packErrorf(PackTooLarge, maxPackSize>>20)
		return path
	}
	if len(ex.images) >= maxPackImages {
		ex.err = packErrorf(PackImagesTooMany, maxPackImages)
		return path
	}
	ex.images[path] = data
	return path
}

// doExportPack 导出选中的配置种类, 返回 zip 文件和每种配置的条数
//
// 在只读的可重复读事务中读取, 各种配置之间的引用一致; 已删除的物品、地图和商品不导出, 设置只导出游戏分类且不含密钥字段。
func doExportPack(ctx context.Context, req *ExportPackRequest) ([]byte, map[string]int, error) {
	name, author, err := checkPackNames(req.Name, req.Author)
	if err != nil {
		return nil, nil, err
	}
	sections, err := orderPackSections(req.Sections)
	if err != nil {
		return nil, nil, err
	}

	ex := &packExporter{ctx: ctx, images: map[string][]byte{}}
	files := map[string]any{}
	counts := map[string]int{}
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		for _, section := range sections {
			value, count, err := sectionOf(section).Export(ctx, tx, ex.image)
			if err != nil {
				return err
			}
			if ex.err != nil {
				return ex.err
			}
			if count > maxPackEntries {
				return packErrorf(PackTooMany, sectionLabel(section), maxPackEntries)
			}
			files[section], counts[section] = value, count
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, nil, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	writeJSON := func(name string, v any) error {
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	manifest := PackManifest{
		Format:     packFormat,
		Version:    packVersion,
		Name:       name,
		Author:     author,
		ExportedAt: time.Now().UTC().Truncate(time.Second),
		Sections:   sections,
	}
	if err := writeJSON(packManifest, manifest); err != nil {
		return nil, nil, err
	}
	for _, section := range sections {
		if err := writeJSON(section+".json", files[section]); err != nil {
			return nil, nil, err
		}
	}
	paths := make([]string, 0, len(ex.images))
	for path := range ex.images {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		// 图片已经压缩过, 直接存储
		w, err := zw.CreateHeader(&zip.FileHeader{Name: path, Method: zip.Store, Modified: manifest.ExportedAt})
		if err != nil {
			return nil, nil, err
		}
		if _, err := w.Write(ex.images[path]); err != nil {
			return nil, nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, nil, err
	}
	if buf.Len() > maxPackSize {
		return nil, nil, packErrorf(PackTooLarge, maxPackSize>>20)
	}
	return buf.Bytes(), counts, nil
}

// readPackFile 读取包中的一个文件, 解压后超过 limit 字节时返回 PackFileInvalid
func readPackFile(f *zip.File, limit int64) ([]byte, error) {
	if f.UncompressedSize64 > uint64(limit) {
		return nil, packErrorf(PackFileInvalid, f.Name)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, packErrorf(PackFileInvalid, f.Name)
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, packErrorf(PackFileInvalid, f.Name)
	}
	return data, nil
}

// decodePackJSON 严格解析 JSON, 有未知字段或多余内容时返回错误
func decodePackJSON(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("unexpected data after JSON")
	}
	return nil
}

// readPack 解析配置包, 校验格式版本并把旧版本升级到当前版本; 包不符合要求时返回可以直接展示的错误
func readPack(data []byte) (*pack, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(zr.File) > maxPackFiles {
		return nil, packErrorf(PackInvalid)
	}
	sum := sha256.Sum256(data)
	p := &pack{digest: hex.EncodeToString(sum[:]), size: int64(len(data)), images: map[string]*zip.File{}}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		if key, ok := strings.CutPrefix(f.Name, packImageDir); ok {
			if storage.CheckKey(key) != nil {
				return nil, packErrorf(PackFileInvalid, f.Name)
			}
			p.images[f.Name] = f
			continue
		}
		files[f.Name] = f
	}

	f := files[packManifest]
	if f == nil {
		return nil, packErrorf(PackInvalid)
	}
	raw, err := readPackFile(f, maxPackJSONSize)
	if err != nil {
		return nil, err
	}
	if err := decodePackJSON(raw, &p.manifest); err != nil {
		return nil, packErrorf(PackFileInvalid, packManifest)
	}
	if p.manifest.Format != packFormat {
		return nil, packErrorf(PackInvalid)
	}
	if p.manifest.Version < 1 || p.manifest.Version > packVersion {
		return nil, packErrorf(PackVersionUnsupported, p.manifest.Version)
	}
	if p.manifest.Name, p.manifest.Author, err = checkPackNames(p.manifest.Name, p.manifest.Author); err != nil {
		return nil, err
	}
	if p.manifest.Sections, err = orderPackSections(p.manifest.Sections); err != nil {
		return nil, err
	}

	raws := map[string][]byte{}
	for _, section := range p.manifest.Sections {
		name := section + ".json"
		if files[name] == nil {
			return nil, packErrorf(PackFileInvalid, name)
		}
		if raws[section], err = readPackFile(files[name], maxPackJSONSize); err != nil {
			return nil, err
		}
	}
	for v := p.manifest.Version; v < packVersion; v++ {
		if err := packUpgrades[v](raws); err != nil {
			return nil, err
		}
	}
	p.contents = make(map[string]Content, len(p.manifest.Sections))
	for _, section := range p.manifest.Sections {
		content, err := sectionOf(section).Decode(raws[section])
		if err != nil {
			return nil, packErrorf(PackFileInvalid, section+".json")
		}
		if content.Len() > maxPackEntries {
			return nil, packErrorf(PackTooMany, sectionLabel(section), maxPackEntries)
		}
		p.contents[section] = content
	}
	return p, nil
}

// Run 一次预览或导入
type Run struct {
	ctx        context.Context
	pack       *pack
	operatorID uint64
	dryRun     bool                     // 预览时只检查图片, 不保存
	images     map[string]*packImageRef // 已处理的图片, 按 用途:包中的路径
}

// packImageRef 包中的一张图片导入后的地址
type packImageRef struct {
	url   string
	ready bool // 本站已有这张图片或已经保存过, 不需要再保存
}

// imageData 读取包中的图片
func (r *Run) imageData(url string) ([]byte, error) {
	f := r.pack.images[url]
	if f == nil {
		return nil, packErrorf(PackImageMissing, url)
	}
	return readPackFile(f, maxPackImageSize)
}

// resolveImage 返回包中的图片导入后的地址, 不保存; 不是包中的图片时原样返回
//
// 本站已有内容相同的这张图片时直接复用(如导出后再导入), 否则按保存的规则检查并算出重新编码后的地址, 再次导入时与已保存的相同。
func (r *Run) resolveImage(purpose, url string) (*packImageRef, error) {
	key, ok := strings.CutPrefix(url, packImageDir)
	if !ok {
		return &packImageRef{url: url, ready: true}, nil
	}
	cacheKey := purpose + ":" + url
	if ref, ok := r.images[cacheKey]; ok {
		return ref, nil
	}
	data, err := r.imageData(url)
	if err != nil {
		return nil, err
	}
	ref := &packImageRef{}
	if ref.url, ref.ready, err = upload.Stored(r.ctx, purpose, key, data); err != nil {
		return nil, err
	}
	if !ref.ready {
		req := &upload.SaveRequest{UserID: r.operatorID, IsAdmin: true, Purpose: purpose, Data: data}
		if ref.url, err = upload.Check(r.ctx, req); err != nil {
			return nil, err
		}
	}
	r.images[cacheKey] = ref
	return ref, nil
}

// Context 请求的上下文
func (r *Run) Context() context.Context { return r.ctx }

// OperatorID 导入的管理员
func (r *Run) OperatorID() uint64 { return r.operatorID }

// DryRun 是否为预览, 预览在回滚的事务中进行, 不保存图片
func (r *Run) DryRun() bool { return r.dryRun }

// Image 返回包中的图片导入后的地址, 导入时保存本站还没有的图片, 预览时只检查; 在 Entry.Apply 中使用
func (r *Run) Image(purpose, url string) (string, error) {
	ref, err := r.resolveImage(purpose, url)
	if err != nil {
		return "", err
	}
	if r.dryRun || ref.ready {
		return ref.url, nil
	}
	data, err := r.imageData(url)
	if err != nil {
		return "", err
	}
	view, err := upload.Save(r.ctx, &upload.SaveRequest{UserID: r.operatorID, IsAdmin: true, Purpose: purpose, Data: data})
	if err != nil {
		return "", err
	}
	ref.url, ref.ready = view.URL, true
	return ref.url, nil
}

// CompareImage 把包中的图片换成导入后的地址, 与已有配置的地址比较; 图片不符合要求时返回冲突的原因。在 Entry.Compare 中使用
func (r *Run) CompareImage(purpose string, url *string) (string, error) {
	ref, err := r.resolveImage(purpose, *url)
	if err != nil {
		if packFailure(err) {
			return err.Error(), nil
		}
		return "", err
	}
	*url = ref.url
	return "", nil
}

// SameImage 已有配置的图片地址原样比较, 用作已有配置转为包中格式时的 image 参数
func SameImage(url string) string { return url }

// CompareStatus 与已有配置比较: 没有时为新增, 内容相同时为相同, 否则为覆盖; 内容按 JSON 比较, 不受字段顺序影响
func CompareStatus(found bool, current, next any) string {
	if !found {
		return PackStatusNew
	}
	normalize := func(v any) (any, bool) {
		data, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		var out any
		return out, json.Unmarshal(data, &out) == nil
	}
	a, okA := normalize(current)
	b, okB := normalize(next)
	if okA && okB && reflect.DeepEqual(a, b) {
		return PackStatusSame
	}
	return PackStatusChanged
}

// ValidateFields 按保存时的请求参数规则校验, 不符合时返回第一个字段的错误
func ValidateFields(obj any) error {
	err := binding.Validator.ValidateStruct(obj)
	var fieldErrs validator.ValidationErrors
	if errors.As(err, &fieldErrs) && len(fieldErrs) > 0 {
		return fmt.Errorf(PackFieldInvalid, snakeCase(fieldErrs[0].Field()))
	}
	return err
}

// snakeCase 字段名转为 JSON 中的写法, 如 SellPrice 转为 sell_price, ImageURL 转为 image_url
func snakeCase(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) {
			// 单词的开头: 前一个是小写, 或者连续大写的最后一个且后面跟着小写
			if i > 0 && (unicode.IsLower(runes[i-1]) || i+1 < len(runes) && unicode.IsLower(runes[i+1]) && unicode.IsUpper(runes[i-1])) {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// packPlainError errors.New 创建的错误, 各模块的错误常量都是这种
var packPlainError = reflect.TypeOf(errors.New(""))

// packFailure 写入一项配置时的错误是否为校验错误(包括唯一约束冲突), 其余为数据库等内部错误, 直接中止
func packFailure(err error) bool {
	var (
		packInvalid      *PackError
		conditionInvalid *condition.InvalidError
		uploadInvalid    *upload.InvalidError
	)
	if db.IsUniqueViolation(err) || reflect.TypeOf(err) == packPlainError ||
		errors.As(err, &packInvalid) || errors.As(err, &conditionInvalid) || errors.As(err, &uploadInvalid) {
		return true
	}
	for _, section := range sectionList() {
		if section.Invalid != nil && section.Invalid(err) {
			return true
		}
	}
	return false
}

// RenameScope 统计范围中的地图代码改名, 如 venue:dock 和 venue:dock:pve 中的 dock
func RenameScope(scope, old, next string) string {
	prefix := "venue:" + old
	if scope == prefix {
		return "venue:" + next
	}
	if rest, ok := strings.CutPrefix(scope, prefix+":"); ok {
		return "venue:" + next + ":" + rest
	}
	return scope
}

// RenameConditionScopes 条件参数中统计范围的地图代码改名
func RenameConditionScopes(conditions []condition.Condition, old, next string) {
	for i := range conditions {
		var params map[string]json.RawMessage
		if json.Unmarshal(conditions[i].Params, &params) != nil {
			continue
		}
		var scope string
		if raw, ok := params["scope"]; !ok || json.Unmarshal(raw, &scope) != nil {
			continue
		}
		renamed := RenameScope(scope, old, next)
		if renamed == scope {
			continue
		}
		params["scope"], _ = json.Marshal(renamed)
		if data, err := json.Marshal(params); err == nil {
			conditions[i].Params = data
		}
	}
}

// entries 按依赖顺序列出包中的每一项, 同一种配置的代码重复时返回 PackKeyDuplicate
func (r *Run) entries() ([]*Entry, error) {
	var entries []*Entry
	for _, section := range r.pack.manifest.Sections {
		for _, e := range r.pack.contents[section].Entries(r) {
			e.section = section
			entries = append(entries, e)
		}
	}

	seen := map[string]struct{}{}
	for _, e := range entries {
		id := e.section + ":" + e.Key
		if _, ok := seen[id]; ok {
			return nil, packErrorf(PackKeyDuplicate, sectionLabel(e.section), e.Key)
		}
		seen[id] = struct{}{}
	}
	return entries, nil
}

// releasePackSavePoint 释放保存点, 一个事务中保存点过多时会拖慢数据库
func releasePackSavePoint(tx *gorm.DB) error {
	return tx.Exec("RELEASE SAVEPOINT " + packSavePoint).Error
}

// chooseActions 按提交的处理方式改名, 包中引用它的配置一起改为新代码; 返回每一项的处理方式, 没有对应项的处理方式忽略
func (r *Run) chooseActions(entries []*Entry, actions []PackAction) (map[*Entry]PackAction, error) {
	byID := map[string]*Entry{}
	for _, e := range entries {
		byID[e.section+":"+e.Key] = e
	}
	chosen := map[*Entry]PackAction{}
	for _, action := range actions {
		switch action.Action {
		case PackActionOverwrite, PackActionSkip, PackActionRename:
		default:
			return nil, packErrorf(PackActionInvalid)
		}
		if e := byID[action.Section+":"+action.Key]; e != nil {
			chosen[e] = action
		}
	}

	// 新代码不能与同种配置在包中的代码或其他新代码相同, 这样改名时引用不会串到别的配置上
	codes := map[string]struct{}{}
	for _, e := range entries {
		codes[e.section+":"+e.Key] = struct{}{}
	}
	for _, e := range entries {
		action, ok := chosen[e]
		if !ok || action.Action != PackActionRename {
			continue
		}
		if e.Rename == nil || !sectionOf(e.section).Renamable {
			return nil, packErrorf(PackRenameNotAllowed, sectionLabel(e.section))
		}
		id := e.section + ":" + action.Code
		if _, taken := codes[id]; taken || action.Code == "" {
			return nil, packErrorf(PackRenameInvalid, sectionLabel(e.section), e.Key)
		}
		codes[id] = struct{}{}
	}
	for _, e := range entries {
		action, ok := chosen[e]
		if !ok || action.Action != PackActionRename {
			continue
		}
		e.Rename(action.Code)
		for _, section := range r.pack.manifest.Sections {
			r.pack.contents[section].Renamed(e.section, e.Key, action.Code)
		}
	}
	return chosen, nil
}

// run 在调用方的事务中按依赖顺序处理每一项: 跳过、与已有配置相同或冲突的项不写入, 其余在保存点中写入, 校验失败时回滚到保存点并标记为冲突
//
// 返回的错误为包或处理方式不符合要求(可以直接展示)或数据库错误。
func (r *Run) run(tx *gorm.DB, actions []PackAction) ([]PackEntryResult, error) {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", packLockKey).Error; err != nil {
		return nil, err
	}
	entries, err := r.entries()
	if err != nil {
		return nil, err
	}
	chosen, err := r.chooseActions(entries, actions)
	if err != nil {
		return nil, err
	}

	results := make([]PackEntryResult, 0, len(entries))
	for _, e := range entries {
		action := chosen[e]
		if action.Action == "" {
			action.Action = PackActionOverwrite
		}
		result := PackEntryResult{Section: e.section, Key: e.Key, Name: e.Name, Action: action.Action}
		if action.Action == PackActionRename {
			result.Code = action.Code
		}
		if result.Status, result.Reason, err = e.Compare(tx); err != nil {
			return nil, err
		}
		// 改名是为了与已有配置并存, 新代码已被使用时不覆盖
		if action.Action == PackActionRename && (result.Status == PackStatusSame || result.Status == PackStatusChanged) {
			result.Status, result.Reason = PackStatusConflict, fmt.Sprintf(PackCodeTaken, action.Code, sectionLabel(e.section))
		}
		if action.Action != PackActionSkip && (result.Status == PackStatusNew || result.Status == PackStatusChanged) {
			if err := tx.SavePoint(packSavePoint).Error; err != nil {
				return nil, err
			}
			if err := e.Apply(tx, result.Status == PackStatusChanged); err != nil {
				if !packFailure(err) {
					return nil, err
				}
				if err := tx.RollbackTo(packSavePoint).Error; err != nil {
					return nil, err
				}
				result.Status, result.Reason = PackStatusConflict, err.Error()
				if db.IsUniqueViolation(err) && e.Taken != "" {
					result.Reason = e.Taken
				}
			}
			if err := releasePackSavePoint(tx); err != nil {
				return nil, err
			}
		}
		results = append(results, result)
	}
	return results, nil
}

// doPreviewPack 解析配置包并在回滚的事务中按处理方式试着导入, 返回每一项的结果; 图片只检查不保存
func doPreviewPack(ctx context.Context, operatorID uint64, data []byte, actions []PackAction) (*PackPreview, error) {
	p, err := readPack(data)
	if err != nil {
		return nil, err
	}
	r := &Run{ctx: ctx, pack: p, operatorID: operatorID, dryRun: true, images: map[string]*packImageRef{}}
	var results []PackEntryResult
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if results, err = r.run(tx, actions); err != nil {
			return err
		}
		return errPackRollback
	})
	if err != nil && !errors.Is(err, errPackRollback) {
		return nil, err
	}
	return &PackPreview{Manifest: p.manifest, Digest: p.digest, Size: p.size, Entries: results}, nil
}

// doImportPack 在一个事务中按处理方式导入配置包并保存导入记录, 全部成功或全部回滚
//
// 有未跳过的冲突项时返回 PackConflicts 和每一项的结果, 不写入任何内容。
func doImportPack(ctx context.Context, operatorID uint64, data []byte, actions []PackAction) (*Import, []PackEntryResult, error) {
	p, err := readPack(data)
	if err != nil {
		return nil, nil, err
	}
	r := &Run{ctx: ctx, pack: p, operatorID: operatorID, images: map[string]*packImageRef{}}
	var (
		record  *Import
		results []PackEntryResult
	)
	err = db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if results, err = r.run(tx, actions); err != nil {
			return err
		}
		record = &Import{
			UserID:  operatorID,
			Name:    p.manifest.Name,
			Author:  p.manifest.Author,
			Version: int32(p.manifest.Version),
			Digest:  p.digest,
			Size:    p.size,
			Entries: results,
		}
		conflicts := 0
		for _, result := range results {
			switch {
			case result.Action == PackActionSkip || result.Status == PackStatusSame:
				record.Skipped++
			case result.Status == PackStatusConflict:
				conflicts++
			case result.Status == PackStatusNew:
				record.Created++
			default:
				record.Updated++
			}
		}
		if conflicts > 0 {
			return packErrorf(PackConflicts, conflicts)
		}
		return tx.Create(record).Error
	})
	if err != nil {
		return nil, results, err
	}
	return record, results, nil
}

// QueryPackImports 获取导入记录列表, 不含每一项的结果
func QueryPackImports(ctx context.Context, req *ListPackImportsRequest) (int64, []Import, error) {
	offset := (req.Current - 1) * req.Size
	query := db.DB(ctx).Model(&Import{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	imports := []Import{}
	if err := query.Omit("entries").Order("id DESC").Offset(offset).Limit(req.Size).Find(&imports).Error; err != nil {
		return 0, nil, err
	}
	return total, imports, nil
}

// queryPackImport 获取一条导入记录, 包括每一项的结果
func queryPackImport(ctx context.Context, importID uint64) (*Import, error) {
	var record Import
	if err := db.DB(ctx).Where("id = ?", importID).Limit(1).Find(&record).Error; err != nil {
		return nil, err
	}
	if record.ID == 0 {
		return nil, errors.New(ImportNotFound)
	}
	return &record, nil
}
