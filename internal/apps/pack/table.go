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
	"context"
	"fmt"

	"gorm.io/gorm"
)

// Table 按表保存的配置种类: 包中是一个 JSON 数组, 每一项按自然键对应表中的一行; 由 RegisterTable 注册
//
// M 为表的模型, R 为包中一项的格式, 通常就是后台保存时的请求, 这样导入与后台保存用同一套规则。
// 比较、图片、改名、已删除的同名配置和字段校验在这里统一处理, 模块只说明怎么读、怎么找和怎么写。
type Table[M, R any] struct {
	Code  string // 见 Section.Code
	Label string
	Hint  string
	Order int

	// Query 导出时按顺序读取全部行的查询, 如 tx.Order("sort_order, id")
	Query func(tx *gorm.DB) *gorm.DB
	// Pack 一行写入包中的内容; image 转换本站图片的地址, 没有图片的配置不用它
	Pack func(row *M, image func(string) string) R
	// Export 可选, 代替 Query 和 Pack 导出全部内容, 需要批量查询其他表时使用
	Export func(tx *gorm.DB, image func(string) string) ([]R, error)
	// Current 可选, 已有的一行转为包中的内容用于比较, 默认为 Pack(row, SameImage); 需要查询其他表时使用
	Current func(tx *gorm.DB, row *M) (R, error)

	// Key、Name 一项的自然键和名称, 见 PackEntryResult
	Key  func(req *R) string
	Name func(req *R) string
	// Taken 写入时撞上唯一约束的原因
	Taken string
	// Rename 改名导入时修改这一项的代码; 为 nil 时这种配置不能改名
	Rename func(req *R, code string)
	// Renamed 可选, 包中的一项配置改名导入时, 修改这一项对它的引用; section 为改名的配置种类
	Renamed func(req *R, section, old, next string)
	// Prepare 可选, 解析后整理一次, 让写法与导出的一致再比较, 如时间取整
	Prepare func(req *R)

	// Find 按自然键查找已有的一行写入 current, 返回是否找到; 有软删除的表要把已删除的也找出来
	Find func(tx *gorm.DB, req *R, current *M) (bool, error)
	// Deleted 可选, 找到的行是否已软删除; 已删除的不能覆盖, 能改名的提示改名导入
	Deleted func(row *M) bool
	// Check 可选, 比较前的额外检查, 返回不能导入的原因; found 为 false 时 current 是零值
	Check func(tx *gorm.DB, current *M, found bool, req *R) (string, error)
	// Keep 可选, 覆盖时沿用已有行中不随配置变化的值, 如商品的剩余库存
	Keep func(current *M, req *R)
	// Image 可选, 这一项的图片用途和地址; 比较和写入前地址会换成导入后的地址
	Image func(req *R) (purpose string, url *string)

	// Lock 可选, 覆盖前锁定并重新读取已有的一行, 防止并发修改互相覆盖; 需要先锁其他东西的配置在 Save 中自己加锁
	Lock func(tx *gorm.DB, current *M) error
	// Save 写入这一项: current 为 nil 时新建, 否则覆盖 current; req 已通过字段校验, 图片已换成导入后的地址
	Save func(tx *gorm.DB, r *Run, current *M, req *R) error
}

// RegisterTable 注册一种按表保存的配置, 需要在启动前调用; 缺少必要的函数时 panic, 其余同 RegisterSection
func RegisterTable[M, R any](t Table[M, R]) {
	if (t.Export == nil && (t.Query == nil || t.Pack == nil)) || (t.Current == nil && t.Pack == nil) ||
		t.Key == nil || t.Name == nil || t.Find == nil || t.Save == nil {
		panic(fmt.Sprintf("pack: incomplete table section %q", t.Code))
	}
	RegisterSection(Section{
		Code: t.Code, Label: t.Label, Hint: t.Hint, Order: t.Order, Renamable: t.Rename != nil,
		Export: func(_ context.Context, tx *gorm.DB, image func(string) string) (any, int, error) {
			list, err := t.export(tx, image)
			return list, len(list), err
		},
		Decode: DecodeList(t.entry, t.Renamed),
	})
}

// export 导出全部内容, 没有内容时为空数组
func (t *Table[M, R]) export(tx *gorm.DB, image func(string) string) ([]R, error) {
	if t.Export != nil {
		list, err := t.Export(tx, image)
		if list == nil {
			list = []R{}
		}
		return list, err
	}
	var rows []M
	if err := t.Query(tx).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]R, len(rows))
	for i := range rows {
		list[i] = t.Pack(&rows[i], image)
	}
	return list, nil
}

// entry 包中的一项
func (t *Table[M, R]) entry(r *Run, req *R) *Entry {
	if t.Prepare != nil {
		t.Prepare(req)
	}
	var current M
	e := &Entry{
		Key:   t.Key(req),
		Name:  t.Name(req),
		Taken: t.Taken,
		Compare: func(tx *gorm.DB) (string, string, error) {
			found, err := t.Find(tx, req, &current)
			if err != nil {
				return "", "", err
			}
			if found && t.Deleted != nil && t.Deleted(&current) {
				if t.Rename == nil {
					return PackStatusConflict, fmt.Sprintf(PackDeleted, t.Label), nil
				}
				return PackStatusConflict, fmt.Sprintf(PackCodeDeleted, t.Label), nil
			}
			if t.Check != nil {
				reason, err := t.Check(tx, &current, found, req)
				if err != nil {
					return "", "", err
				}
				if reason != "" {
					return PackStatusConflict, reason, nil
				}
			}
			next := *req
			if found && t.Keep != nil {
				t.Keep(&current, &next)
			}
			if t.Image != nil {
				purpose, url := t.Image(&next)
				reason, err := r.CompareImage(purpose, url)
				if err != nil {
					return "", "", err
				}
				if reason != "" {
					return PackStatusConflict, reason, nil
				}
			}
			if !found {
				return PackStatusNew, "", nil
			}
			var packed R
			if t.Current != nil {
				if packed, err = t.Current(tx, &current); err != nil {
					return "", "", err
				}
			} else {
				packed = t.Pack(&current, SameImage)
			}
			return CompareStatus(true, packed, &next), "", nil
		},
		Apply: func(tx *gorm.DB, exists bool) error {
			next := *req
			if t.Image != nil {
				purpose, url := t.Image(&next)
				saved, err := r.Image(purpose, *url)
				if err != nil {
					return err
				}
				*url = saved
			}
			if err := ValidateFields(&next); err != nil {
				return err
			}
			if !exists {
				return t.Save(tx, r, nil, &next)
			}
			if t.Lock != nil {
				if err := t.Lock(tx, &current); err != nil {
					return err
				}
			}
			// 加锁后再沿用: 这些值可能在比较之后变过, 如商品被买走了几件
			if t.Keep != nil {
				t.Keep(&current, &next)
			}
			return t.Save(tx, r, &current, &next)
		},
	}
	if t.Rename != nil {
		e.Rename = func(code string) { t.Rename(req, code) }
	}
	return e
}
