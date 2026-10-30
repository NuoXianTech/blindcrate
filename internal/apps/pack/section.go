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
	"cmp"
	"context"
	"fmt"
	"regexp"
	"slices"
	"sync"

	"gorm.io/gorm"
)

// Section 一种配置, 由所属模块在 init 中注册
type Section struct {
	Code      string // 包中的文件名(加上 .json), 也是导入结果中的 section
	Label     string // 名称, 用于提示
	Hint      string // 导出时的说明, 如按什么对应、带不带图片
	Order     int    // 依赖顺序, 导出和导入时从小到大处理, 被引用的配置排在前面
	Renamable bool   // 是否可以改名导入, 代码或 ID 被别处直接引用、或者不能修改的配置不能改名

	// Export 在只读事务中读取全部配置, 返回写入包中的内容和条数; 本站图片的地址用 image 转换后写入
	Export func(ctx context.Context, tx *gorm.DB, image func(string) string) (any, int, error)
	// Decode 严格解析包中的文件, 可以使用 DecodeList
	Decode func(data []byte) (Content, error)
	// Invalid 可选, 写入时返回的错误是否为这个模块的校验错误; errors.New 创建的错误和唯一约束冲突已经算作校验错误
	Invalid func(err error) bool
}

// Content 包中一种配置解析后的内容
type Content interface {
	// Len 条数
	Len() int
	// Entries 按包中的顺序生成每一项
	Entries(r *Run) []*Entry
	// Renamed 包中另一项配置改名导入时, 把这里对它的引用一起改为新代码
	Renamed(section, old, next string)
}

// Entry 包中的一项配置
type Entry struct {
	Key   string // 见 PackEntryResult.Key
	Name  string
	Taken string // 写入时撞上唯一约束的原因
	// Rename 改名导入, 只改这一项的代码; 引用它的配置由各自的 Content.Renamed 修改。只有可以改名的配置种类有
	Rename func(code string)
	// Compare 与已有配置比较, 返回 PackStatus 开头的常量, 冲突时带上原因
	Compare func(tx *gorm.DB) (status, reason string, err error)
	// Apply 写入这一项, exists 为是否覆盖已有配置; 返回校验错误时由调用方回滚到保存点并标记为冲突
	Apply func(tx *gorm.DB, exists bool) error

	section string
}

// SectionView 配置种类, 用于后台展示
type SectionView struct {
	Code      string `json:"code"`
	Label     string `json:"label"`
	Hint      string `json:"hint"`
	Renamable bool   `json:"renamable"`
}

var (
	sectionsMu  sync.RWMutex
	sectionsMap = map[string]*Section{}
)

// sectionCodePattern 配置种类的代码, 用作包中的文件名
var sectionCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

// RegisterSection 注册一种配置, 代码不合法、重复或缺少导出和解析函数时 panic
func RegisterSection(s Section) {
	if !sectionCodePattern.MatchString(s.Code) || s.Label == "" || s.Export == nil || s.Decode == nil {
		panic(fmt.Sprintf("pack: invalid section %q", s.Code))
	}
	sectionsMu.Lock()
	defer sectionsMu.Unlock()
	if _, ok := sectionsMap[s.Code]; ok {
		panic(fmt.Sprintf("pack: section %q already registered", s.Code))
	}
	sectionsMap[s.Code] = &s
}

// sectionList 全部配置种类, 按依赖顺序
func sectionList() []*Section {
	sectionsMu.RLock()
	defer sectionsMu.RUnlock()
	list := make([]*Section, 0, len(sectionsMap))
	for _, s := range sectionsMap {
		list = append(list, s)
	}
	slices.SortFunc(list, func(a, b *Section) int {
		return cmp.Or(cmp.Compare(a.Order, b.Order), cmp.Compare(a.Code, b.Code))
	})
	return list
}

// sectionOf 按代码获取配置种类, 没有注册时返回 nil
func sectionOf(code string) *Section {
	sectionsMu.RLock()
	defer sectionsMu.RUnlock()
	return sectionsMap[code]
}

// sectionLabel 配置种类的名称, 用于提示
func sectionLabel(code string) string {
	if s := sectionOf(code); s != nil {
		return s.Label
	}
	return code
}

// Sections 全部配置种类, 按依赖顺序
func Sections() []SectionView {
	list := sectionList()
	views := make([]SectionView, len(list))
	for i, s := range list {
		views[i] = SectionView{Code: s.Code, Label: s.Label, Hint: s.Hint, Renamable: s.Renamable}
	}
	return views
}

// ListContent 以 JSON 数组保存的配置, 由 DecodeList 生成
type ListContent[T any] struct {
	Items   []T
	entry   func(r *Run, req *T) *Entry
	renamed func(req *T, section, old, next string)
}

// DecodeList 返回按 JSON 数组严格解析的 Decode 函数; entry 生成每一项, renamed 修改一项中对其他配置的引用, 可以为空
func DecodeList[T any](entry func(r *Run, req *T) *Entry, renamed func(req *T, section, old, next string)) func(data []byte) (Content, error) {
	return func(data []byte) (Content, error) {
		c := &ListContent[T]{entry: entry, renamed: renamed}
		if err := decodePackJSON(data, &c.Items); err != nil {
			return nil, err
		}
		return c, nil
	}
}

func (c *ListContent[T]) Len() int { return len(c.Items) }

func (c *ListContent[T]) Entries(r *Run) []*Entry {
	entries := make([]*Entry, len(c.Items))
	for i := range c.Items {
		entries[i] = c.entry(r, &c.Items[i])
	}
	return entries
}

func (c *ListContent[T]) Renamed(section, old, next string) {
	if c.renamed == nil {
		return
	}
	for i := range c.Items {
		c.renamed(&c.Items[i], section, old, next)
	}
}
