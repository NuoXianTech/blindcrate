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

// 供 pack_test 包中的测试使用: 配置种类由各模块注册, 引用这些模块的测试放在本包中会循环引用

const (
	PackFormat       = packFormat
	PackManifestFile = packManifest
	PackVersion      = packVersion
	MaxPackEntries   = maxPackEntries
)

var (
	ReadPack          = readPack
	OrderPackSections = orderPackSections
	PackFailure       = packFailure
	SectionLabel      = sectionLabel
	PackErrorf        = packErrorf
)

// Manifest 配置包的说明
func (p *pack) Manifest() PackManifest { return p.manifest }

// Digest 文件的 sha256
func (p *pack) Digest() string { return p.digest }

// HasImage 包中是否有这张图片
func (p *pack) HasImage(path string) bool { return p.images[path] != nil }

// Content 一种配置解析后的内容, 包中没有时为 nil
func (p *pack) Content(section string) Content { return p.contents[section] }

// DecodeSection 用注册的 Decode 解析一种配置
func DecodeSection(section string, data []byte) (Content, error) {
	return sectionOf(section).Decode(data)
}

// NewRun 由各种配置的内容组成配置包, 返回一次不连数据库的导入
func NewRun(contents map[string]Content) (*Run, error) {
	sections := make([]string, 0, len(contents))
	for section := range contents {
		sections = append(sections, section)
	}
	ordered, err := orderPackSections(sections)
	if err != nil {
		return nil, err
	}
	return &Run{pack: &pack{manifest: PackManifest{Sections: ordered}, contents: contents}}, nil
}

// Entries 按依赖顺序列出包中的每一项
func (r *Run) Entries() ([]*Entry, error) { return r.entries() }

// ChooseActions 按处理方式改名, 返回有处理方式的项数
func (r *Run) ChooseActions(entries []*Entry, actions []PackAction) (int, error) {
	chosen, err := r.chooseActions(entries, actions)
	return len(chosen), err
}

// Content 一种配置解析后的内容
func (r *Run) Content(section string) Content { return r.pack.contents[section] }
