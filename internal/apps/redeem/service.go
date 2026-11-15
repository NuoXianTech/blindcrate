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

package redeem

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition"
	"github.com/NuoXianTech/blindcrate/internal/apps/ledger"
	"github.com/NuoXianTech/blindcrate/internal/apps/reward"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	ledger.RegisterBizType(ledger.KindWallet, BizRedeem, "兑换码")
	ledger.RegisterBizType(ledger.KindInventory, BizRedeem, "兑换码")
	audit.RegisterAction(ActionCreate, "生成兑换码")
	audit.RegisterAction(ActionUpdate, "修改兑换码批次")
	audit.RegisterAction(ActionDelete, "删除兑换码批次")
}

// InvalidError 批次配置不合法, Error 返回的提示可以直接展示给管理员
type InvalidError struct {
	Reason string
}

func (e *InvalidError) Error() string { return e.Reason }

// RejectedError 不能兑换, Error 返回的提示可以直接展示给玩家
type RejectedError struct {
	Reason string
}

func (e *RejectedError) Error() string { return e.Reason }

// Validate 保存批次前校验配置, 校验后合并奖励中的同类项
//
// 配置不合法时返回 *InvalidError; 其他错误为查库失败等内部错误。名称需要调用方先去掉首尾空白。
func Validate(tx *gorm.DB, b *Batch) error {
	if n := utf8.RuneCountInString(b.Name); n < 1 || n > maxNameLength {
		return &InvalidError{Reason: NameInvalid}
	}
	if b.MaxUses < 0 || b.MaxUses > maxCodeUses {
		return &InvalidError{Reason: MaxUsesInvalid}
	}
	if b.UserLimit < 0 || b.UserLimit > MaxBatchCodes {
		return &InvalidError{Reason: UserLimitInvalid}
	}
	bundle, err := reward.Check(tx, b.Reward)
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
	b.Reward = bundle
	if b.Requirements == nil {
		b.Requirements = []condition.Condition{}
	}
	if err := condition.Validate(tx, b.Requirements); err != nil {
		var conditionErr *condition.InvalidError
		if errors.As(err, &conditionErr) {
			return &InvalidError{Reason: fmt.Sprintf(RequirementInvalid, conditionErr.Error())}
		}
		return err
	}
	return nil
}

// Normalize 规范化兑换码: 去掉空白和连字符, 字母转为大写
//
// 保存和兑换时都先规范化, 玩家输入时不必区分大小写, 也可以带上分隔用的空格和连字符。
func Normalize(code string) string {
	return strings.ToUpper(strings.Map(func(r rune) rune {
		if r == '-' || unicode.IsSpace(r) {
			return -1
		}
		return r
	}, code))
}

// ValidCode 规范化后的兑换码是否合法: 4-32 个字母、数字或汉字
func ValidCode(code string) bool {
	n := 0
	for _, r := range code {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
		n++
	}
	return n >= minCodeLength && n <= maxCodeLength
}

// Generate 随机生成 count 个互不相同的兑换码, 使用 crypto/rand, 无法被预测; 是否与已有的兑换码重复由保存时的唯一索引判断
func Generate(count int) []string {
	codes := make([]string, 0, count)
	seen := make(map[string]struct{}, count)
	buf := make([]byte, randomLength)
	for len(codes) < count {
		_, _ = rand.Read(buf)
		for i, b := range buf {
			// 字符表恰好 32 个字符, 取模后每个字符出现的概率相同
			buf[i] = codeAlphabet[int(b)%len(codeAlphabet)]
		}
		code := string(buf)
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	return codes
}

// Result 兑换的结果
type Result struct {
	Name     string        `json:"name"`     // 批次的名称
	Received []reward.Line `json:"received"` // 实际得到的货币和物品, 已下架的物品按回收价折成了货币
}

// Redeem 用兑换码兑换奖励, input 为玩家输入的兑换码, 不需要先规范化
//
// 同一用户的兑换在咨询锁上排队, 兑换码加行锁, 可使用次数和每人可兑换的个数都不会被并发请求绕过; 批次加共享锁, 与后台修改和删除互斥。
// 兑换记录的唯一索引防止同一用户重复兑换同一个兑换码, 发放的幂等键再兜底一次。
// 兑换码不存在、批次已停用或已删除时都返回 CodeNotFound; 不满足使用条件或达到每人可兑换的个数时返回 *RejectedError。
// 奖励中已下架或已删除的物品按回收价折成货币; 货币已停用时返回 RewardUnavailable, 不计次数, 可以之后再兑换。
func Redeem(ctx context.Context, userID uint64, input string) (*Result, error) {
	code := Normalize(input)
	if !ValidCode(code) {
		return nil, errors.New(CodeNotFound)
	}
	var (
		batch    Batch
		delivery *reward.Delivery
	)
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := db.Lock(tx, fmt.Sprintf("redeem:%d", userID)); err != nil {
			return err
		}
		// 输错兑换码、兑换码所在的批次已停用都很常见, 用 Find 查询, 查不到时不会被当成查询错误记进日志
		var target Code
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", code).Limit(1).Find(&target).Error; err != nil {
			return err
		}
		if target.ID == 0 {
			return errors.New(CodeNotFound)
		}
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("id = ? AND is_enabled", target.BatchID).Limit(1).Find(&batch).Error; err != nil {
			return err
		}
		if batch.ID == 0 {
			return errors.New(CodeNotFound)
		}
		// 先看是否兑换过, 兑换过的用户在兑换码到期或兑换完之后看到的仍是已兑换
		var redeemed int64
		if err := tx.Model(&Record{}).Where("code_id = ? AND user_id = ?", target.ID, userID).Count(&redeemed).Error; err != nil {
			return err
		}
		if redeemed > 0 {
			return errors.New(AlreadyRedeemed)
		}
		if batch.Expired(time.Now()) {
			return errors.New(CodeExpired)
		}
		if batch.MaxUses > 0 && target.Used >= batch.MaxUses {
			return errors.New(CodeUsedUp)
		}
		if batch.UserLimit > 0 {
			var count int64
			if err := tx.Model(&Record{}).Where("batch_id = ? AND user_id = ?", batch.ID, userID).Count(&count).Error; err != nil {
				return err
			}
			if count >= int64(batch.UserLimit) {
				return &RejectedError{Reason: fmt.Sprintf(UserLimitReached, batch.UserLimit)}
			}
		}
		unmet, err := condition.FirstUnmet(tx, userID, batch.Requirements)
		if err != nil {
			return err
		}
		if unmet != nil {
			return &RejectedError{Reason: fmt.Sprintf(RequirementUnmet, unmet.Text)}
		}
		// 兑换记录中保存合并后实际发放的内容; 奖励发不出时不记兑换, 奖励改好后还可以兑换
		bundle, err := reward.Prepare(tx, batch.Reward)
		if err != nil {
			return undeliverable(err)
		}
		record := Record{BatchID: batch.ID, CodeID: target.ID, UserID: userID, Reward: bundle}
		if err := tx.Create(&record).Error; err != nil {
			// 咨询锁之外的兜底, 正常不会走到
			if db.IsUniqueViolation(err) {
				return errors.New(AlreadyRedeemed)
			}
			return err
		}
		if delivery, err = reward.Deliver(tx, &reward.ApplyRequest{
			UserID:         userID,
			Bundle:         bundle,
			BizType:        BizRedeem,
			BizID:          strconv.FormatUint(record.ID, 10),
			IdempotencyKey: fmt.Sprintf("redeem:%d:%d", target.ID, userID),
			Remark:         batch.Name,
		}); err != nil {
			return undeliverable(err)
		}
		return tx.Model(&target).UpdateColumn("used", gorm.Expr("used + 1")).Error
	})
	if err != nil {
		return nil, err
	}
	delivery.Push(ctx)
	received, err := delivery.Received(db.DB(ctx))
	if err != nil {
		return nil, err
	}
	return &Result{Name: batch.Name, Received: received}, nil
}

// undeliverable 奖励发不出时换成给玩家的提示, 其他错误原样返回
func undeliverable(err error) error {
	switch reward.FailureOf(err) {
	case reward.Unavailable:
		return errors.New(RewardUnavailable)
	case reward.TooLarge:
		return errors.New(RewardTooLarge)
	}
	return err
}
