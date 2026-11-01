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

package codex

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/ledger"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	ledger.RegisterBizType(ledger.KindWallet, BizReward, "图鉴奖励")
	ledger.RegisterBizType(ledger.KindInventory, BizReward, "图鉴奖励")
	audit.RegisterAction(ActionCreate, "新建收集组")
	audit.RegisterAction(ActionUpdate, "修改收集组")
	audit.RegisterAction(ActionDelete, "删除收集组")
}

// codePattern 收集组代码, 小写字母、数字和下划线
var codePattern = regexp.MustCompile(`^[a-z0-9_]{1,32}$`)

// maxMissingShown 物品不合法时提示中最多列出的物品 ID 数
const maxMissingShown = 10

// InvalidError 收集组配置不合法, Error 返回的提示可以直接展示给管理员
type InvalidError struct {
	Reason string
}

func (e *InvalidError) Error() string { return e.Reason }

// Validate 保存收集组前校验配置, 校验后物品去重并从小到大排列, 奖励合并同类项
//
// 配置不合法时返回 *InvalidError; 其他错误为查库失败等内部错误。名称等文本字段需要调用方先去掉首尾空白。
// 新加入的物品需为未删除的藏品, 可以是已下架的: 下架期间不计入进度, 重新上架后再计入。
// kept 为修改前已在组内的物品, 其中之后被删除或不再是藏品的可以保留, 同样不计入进度, 启停和修改其他字段时不必先移除。
func Validate(tx *gorm.DB, s *Set, kept []uint64) error {
	if !codePattern.MatchString(s.Code) {
		return &InvalidError{Reason: CodeInvalid}
	}
	if n := utf8.RuneCountInString(s.Name); n < 1 || n > maxNameLength {
		return &InvalidError{Reason: NameInvalid}
	}
	if utf8.RuneCountInString(s.Description) > maxDescriptionLength {
		return &InvalidError{Reason: DescriptionInvalid}
	}
	ids := slices.Clone(s.ItemIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if len(ids) == 0 {
		return &InvalidError{Reason: ItemsEmpty}
	}
	if len(ids) > MaxSetItems {
		return &InvalidError{Reason: ItemsTooMany}
	}
	var found []uint64
	if err := tx.Model(&item.Item{}).Where("id IN ? AND category = ?", ids, item.CategoryCollectible).Pluck("id", &found).Error; err != nil {
		return err
	}
	if len(found) != len(ids) {
		var missing []string
		for _, id := range ids {
			if !slices.Contains(found, id) && !slices.Contains(kept, id) && len(missing) < maxMissingShown {
				missing = append(missing, strconv.FormatUint(id, 10))
			}
		}
		if len(missing) > 0 {
			return &InvalidError{Reason: fmt.Sprintf(ItemsInvalid, strings.Join(missing, ", "))}
		}
	}
	s.ItemIDs = ids
	bundle, err := reward.Check(tx, s.Reward)
	if err != nil {
		var invalid *reward.InvalidError
		if errors.As(err, &invalid) {
			return &InvalidError{Reason: fmt.Sprintf(RewardInvalid, invalid.Reason)}
		}
		return err
	}
	if bundle.IsEmpty() {
		return &InvalidError{Reason: RewardEmpty}
	}
	s.Reward = bundle
	return nil
}

// EntryView 图鉴中的一种藏品
type EntryView struct {
	ItemID      uint64     `json:"item_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"` // 收集后才返回
	Quality     int16      `json:"quality"`
	ImageURL    string     `json:"image_url"` // 没有收集时也返回, 由前端显示为剪影
	Collected   bool       `json:"collected"`
	FirstAt     *time.Time `json:"first_at"` // 第一次获得的时间, 没有收集时为空
	Quantity    int64      `json:"quantity"` // 背包中现有的数量, 卖出或用掉后为 0
}

// SetView 玩家看到的收集组, 带收集进度和领取情况
type SetView struct {
	Code        string        `json:"code"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	ItemIDs     []uint64      `json:"item_ids"`  // 组内启用的藏品, 按图鉴中的顺序排列
	Collected   int           `json:"collected"` // 其中已收集的数量
	Reward      []reward.Line `json:"reward"`
	Claimed     bool          `json:"claimed"`
	Claimable   bool          `json:"claimable"` // 已集齐且未领取
}

// CodexData 玩家的图鉴
type CodexData struct {
	Items     []EntryView `json:"items"`     // 全部启用的藏品, 按品质从高到低、ID 排列
	Collected int         `json:"collected"` // 已收集的藏品数
	Sets      []SetView   `json:"sets"`      // 启用的收集组, 按排序值和 ID 排列; 组内没有启用的藏品时不返回
	Claimable int         `json:"claimable"` // 可以领取的收集组数
}

// entryRow 一种启用的藏品和用户的背包行, 没有收集时背包的字段为空
type entryRow struct {
	ID          uint64
	Name        string
	Description string
	Quality     int16
	ImageURL    string
	FirstAt     *time.Time
	Quantity    int64
}

// QueryCodex 获取用户的图鉴: 全部启用的藏品和收集情况, 以及启用的收集组的进度
func QueryCodex(ctx context.Context, userID uint64) (*CodexData, error) {
	tx := db.DB(ctx)
	var rows []entryRow
	if err := tx.Model(&item.Item{}).
		Select("items.id, items.name, items.description, items.quality, items.image_url, "+
			"inventory_items.created_at AS first_at, COALESCE(inventory_items.quantity, 0) AS quantity").
		Joins("LEFT JOIN inventory_items ON inventory_items.item_id = items.id AND inventory_items.user_id = ?", userID).
		Where("items.category = ? AND items.is_enabled", item.CategoryCollectible).
		Order("items.quality DESC, items.id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	data := &CodexData{Items: make([]EntryView, len(rows)), Sets: []SetView{}}
	for i, row := range rows {
		entry := EntryView{
			ItemID:    row.ID,
			Name:      row.Name,
			Quality:   row.Quality,
			ImageURL:  row.ImageURL,
			Collected: row.FirstAt != nil,
			FirstAt:   row.FirstAt,
			Quantity:  row.Quantity,
		}
		if entry.Collected {
			entry.Description = row.Description
			data.Collected++
		}
		data.Items[i] = entry
	}

	var sets []Set
	if err := tx.Where("is_enabled").Order("sort_order, id").Find(&sets).Error; err != nil {
		return nil, err
	}
	if len(sets) == 0 {
		return data, nil
	}
	claimed, err := queryClaimed(tx, userID, sets)
	if err != nil {
		return nil, err
	}
	bundles := make([]reward.Bundle, len(sets))
	for i := range sets {
		bundles[i] = sets[i].Reward
	}
	lines, err := reward.Lines(tx, bundles...)
	if err != nil {
		return nil, err
	}
	for i := range sets {
		s := &sets[i]
		ids, collected := members(data.Items, s.ItemIDs)
		if len(ids) == 0 {
			continue
		}
		view := SetView{
			Code:        s.Code,
			Name:        s.Name,
			Description: s.Description,
			ItemIDs:     ids,
			Collected:   collected,
			Reward:      lines[i],
			Claimed:     claimed[s.ID],
		}
		view.Claimable = !view.Claimed && collected == len(ids)
		if view.Claimable {
			data.Claimable++
		}
		data.Sets = append(data.Sets, view)
	}
	return data, nil
}

// members 收集组中启用的藏品和其中已收集的数量, 按 entries 的顺序排列; entries 为全部启用的藏品
func members(entries []EntryView, itemIDs []uint64) ([]uint64, int) {
	in := make(map[uint64]bool, len(itemIDs))
	for _, id := range itemIDs {
		in[id] = true
	}
	ids := []uint64{}
	collected := 0
	for _, entry := range entries {
		if in[entry.ItemID] {
			ids = append(ids, entry.ItemID)
			if entry.Collected {
				collected++
			}
		}
	}
	return ids, collected
}

// queryClaimed 一次查出用户领取过哪些收集组
func queryClaimed(tx *gorm.DB, userID uint64, sets []Set) (map[uint64]bool, error) {
	ids := make([]uint64, len(sets))
	for i := range sets {
		ids[i] = sets[i].ID
	}
	var rows []uint64
	if err := tx.Model(&ClaimRecord{}).Where("set_id IN ? AND user_id = ?", ids, userID).Pluck("set_id", &rows).Error; err != nil {
		return nil, err
	}
	claimed := make(map[uint64]bool, len(rows))
	for _, id := range rows {
		claimed[id] = true
	}
	return claimed, nil
}

// ClaimResult 领取的结果
type ClaimResult struct {
	Received []reward.Line `json:"received"` // 实际得到的货币和物品, 已下架的物品按回收价折成了货币
}

// Claim 领取收集组的奖励, 需要集齐组内全部启用的藏品
//
// 收集组加共享锁, 与后台修改和删除互斥; 领取记录的主键防止重复领取, 发放的幂等键再兜底一次。
// 奖励中已下架或已删除的物品按回收价折成货币; 货币已停用时返回 RewardUnavailable, 可以之后再领取。
func Claim(ctx context.Context, userID uint64, code string) (*ClaimResult, error) {
	var (
		result   *reward.Result
		replayed bool
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var s Set
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("code = ? AND is_enabled", code).First(&s).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(SetNotFound)
			}
			return err
		}
		var ids []uint64
		if len(s.ItemIDs) > 0 {
			if err := tx.Model(&item.Item{}).
				Where("id IN ? AND category = ? AND is_enabled", s.ItemIDs, item.CategoryCollectible).
				Pluck("id", &ids).Error; err != nil {
				return err
			}
		}
		// 组内没有启用的藏品时玩家看不到这个收集组
		if len(ids) == 0 {
			return errors.New(SetNotFound)
		}
		inserted := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&ClaimRecord{SetID: s.ID, UserID: userID})
		if inserted.Error != nil {
			return inserted.Error
		}
		if inserted.RowsAffected == 0 {
			return errors.New(AlreadyClaimed)
		}
		var collected int64
		if err := tx.Model(&inventory.UserItem{}).Where("user_id = ? AND item_id IN ?", userID, ids).Count(&collected).Error; err != nil {
			return err
		}
		if collected < int64(len(ids)) {
			return errors.New(NotCompleted)
		}
		bundle, err := reward.Deliverable(tx, s.Reward, nil)
		if err != nil {
			return err
		}
		// 奖励全是已下架且没有回收价的物品时什么也发不出, 不记领取, 奖励改好后还可以领
		if bundle.IsEmpty() {
			return errors.New(RewardUnavailable)
		}
		if result, replayed, err = reward.Apply(tx, &reward.ApplyRequest{
			UserID:         userID,
			Bundle:         bundle,
			BizType:        BizReward,
			BizID:          strconv.FormatUint(s.ID, 10),
			IdempotencyKey: fmt.Sprintf("codex:%d:%d", s.ID, userID),
			Remark:         s.Name,
		}); err != nil {
			switch err.Error() {
			case item.ItemNotFound, item.ItemDisabled, wallet.CurrencyNotFound, wallet.CurrencyDisabled:
				return errors.New(RewardUnavailable)
			case wallet.AmountInvalid, inventory.QuantityInvalid:
				return errors.New(RewardTooLarge)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !replayed {
		result.Push(ctx)
	}
	lines, err := reward.Lines(db.DB(ctx), result.Bundle)
	if err != nil {
		return nil, err
	}
	return &ClaimResult{Received: lines[0]}, nil
}
