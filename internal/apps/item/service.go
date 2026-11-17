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

package item

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/fault"
	"github.com/NuoXianTech/blindcrate/internal/apps/params"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/upload"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	audit.RegisterAction(ActionItemCreate, "新建物品")
	audit.RegisterAction(ActionItemUpdate, "修改物品")
	audit.RegisterAction(ActionItemDelete, "删除物品")
	audit.RegisterAction(ActionQualityCreate, "新建品质")
	audit.RegisterAction(ActionQualityUpdate, "修改品质")
	audit.RegisterAction(ActionQualityDelete, "删除品质")
	params.RegisterSource(SourceItem, params.Source{Options: itemOptions})
	params.RegisterSource(SourceQuality, params.Source{Options: qualityOptions})
	upload.RegisterPurpose(ImagePurpose, true, false, 512)
	upload.RegisterReference("items", "image_url")
}

// itemOptions 可以选的物品: 全部没有删除的物品, 按品质从高到低; 已下架的不能新选
func itemOptions(tx *gorm.DB) ([]params.Option, error) {
	var items []Item
	if err := tx.Select("id", "name", "quality", "is_enabled").Order("quality DESC, id").Find(&items).Error; err != nil {
		return nil, err
	}
	options := make([]params.Option, len(items))
	for i, it := range items {
		options[i] = params.Option{
			Value:    strconv.FormatUint(it.ID, 10),
			Label:    it.Name,
			Hint:     "#" + strconv.FormatUint(it.ID, 10),
			Quality:  it.Quality,
			Disabled: !it.IsEnabled,
		}
	}
	return options, nil
}

// qualityOptions 可以选的品质, 按等级从低到高
func qualityOptions(tx *gorm.DB) ([]params.Option, error) {
	var qualities []Quality
	if err := tx.Order("level").Find(&qualities).Error; err != nil {
		return nil, err
	}
	options := make([]params.Option, len(qualities))
	for i, q := range qualities {
		options[i] = params.Option{Value: strconv.Itoa(int(q.Level)), Label: q.Name, Quality: q.Level}
	}
	return options, nil
}

// Placeholder 生成占位物品, 保留原物品 ID, 按 currency 回收
func Placeholder(id uint64, currency string) *Item {
	return &Item{
		ID:            id,
		Name:          placeholderName(id),
		Quality:       PlaceholderQuality,
		Category:      CategoryCollectible,
		SellCurrency:  currency,
		SellPrice:     PlaceholderSellPrice,
		Width:         1,
		Height:        1,
		IsPlaceholder: true,
	}
}

// placeholderName 占位物品名称, 带上原物品 ID, 删了多个物品时玩家能区分
func placeholderName(id uint64) string {
	return fmt.Sprintf("未知物品 #%d", id)
}

// Resolve 按 ID 批量获取物品, 已删除或不存在的物品返回占位物品, 占位物品按基础货币回收; 没有选择基础货币时占位物品不能出售
//
// 背包展示、卖出、货仓暗拍等读取用户持有的物品时都应走这里, 占位规则只在这一处决定。
func Resolve(tx *gorm.DB, ids []uint64) (map[uint64]*Item, error) {
	result := make(map[uint64]*Item, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	var items []Item
	if err := tx.Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		result[items[i].ID] = &items[i]
	}
	var currency string
	for _, id := range ids {
		if result[id] != nil {
			continue
		}
		if currency == "" {
			economy, err := settings.Economy.Get(tx.Statement.Context)
			if err != nil {
				return nil, err
			}
			currency = economy.BaseCurrency
		}
		result[id] = Placeholder(id, currency)
	}
	return result, nil
}

var (
	// codePattern 品质和类别的代码, 小写字母、数字和下划线
	codePattern = regexp.MustCompile(`^[a-z0-9_]{1,32}$`)
	// qualityColorPattern 品质颜色, 统一为小写的 #rrggbb
	qualityColorPattern = regexp.MustCompile(`^#[0-9a-f]{6}$`)

	// categories 已注册的物品类别, 按注册顺序, 只在启动前写入
	categories = []Category{
		{Code: CategoryCollectible, Label: "藏品"},
	}
)

// RegisterCategory 注册一个物品类别, 如道具、门票, 需要在启动前调用; 代码格式不正确或重复时 panic
func RegisterCategory(code, label string) {
	if !codePattern.MatchString(code) {
		panic(fmt.Sprintf("item: invalid category %q", code))
	}
	if FindCategory(code) != nil {
		panic(fmt.Sprintf("item: category %s already registered", code))
	}
	categories = append(categories, Category{Code: code, Label: label})
}

// Categories 全部物品类别, 按注册顺序
func Categories() []Category {
	return slices.Clone(categories)
}

// FindCategory 按代码查找物品类别, 没有注册时返回 nil
func FindCategory(code string) *Category {
	for i := range categories {
		if categories[i].Code == code {
			return &categories[i]
		}
	}
	return nil
}

// QueryQualities 获取全部品质, 按排序值和等级排列
func QueryQualities(ctx context.Context) ([]Quality, error) {
	qualities := []Quality{}
	if err := db.DB(ctx).Order("sort_order, level").Find(&qualities).Error; err != nil {
		return nil, err
	}
	return qualities, nil
}

// ValidateQuality 校验品质的代码、名称、颜色和特效
func ValidateQuality(q *Quality) error {
	if !codePattern.MatchString(q.Code) {
		return fault.Invalid(QualityCodeInvalid)
	}
	if n := utf8.RuneCountInString(q.Name); n < 1 || n > 32 {
		return fault.Invalid(QualityNameInvalid)
	}
	if !qualityColorPattern.MatchString(q.Color) {
		return fault.Invalid(QualityColorInvalid)
	}
	if q.Effect != "" && q.Effect != QualityEffectRainbow {
		return fault.Invalid(QualityEffectInvalid)
	}
	return nil
}

// CheckQuality 品质等级必须存在, 不存在时返回 QualityNotFound
//
// 对品质加共享锁, 防止保存物品时品质被并发删除。
func CheckQuality(tx *gorm.DB, level int16) error {
	var q Quality
	if err := q.Exact(tx.Clauses(clause.Locking{Strength: "SHARE"}), level); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fault.Invalid(QualityNotFound)
		}
		return err
	}
	return nil
}

// ScanDurations 每个品质等级的扫描时长, 毫秒
func ScanDurations(tx *gorm.DB) (map[int16]int32, error) {
	var qualities []Quality
	if err := tx.Select("level", "scan_ms").Find(&qualities).Error; err != nil {
		return nil, err
	}
	durations := make(map[int16]int32, len(qualities))
	for _, q := range qualities {
		durations[q.Level] = q.ScanMs
	}
	return durations, nil
}
