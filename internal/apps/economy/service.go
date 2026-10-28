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

package economy

import (
	"strconv"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/logger"
	"gorm.io/gorm"
)

func init() {
	auth.OnRegister(grantSignup)
}

// grantSignup 在注册的事务内发放新手资金, 金额为 0 时不发放
//
// 发放失败只记录日志, 不影响注册; 在保存点中执行, 失败时只回滚发放本身。
// 新用户还没有连接, 不推送余额变动。
func grantSignup(tx *gorm.DB, user *auth.User) error {
	ctx := tx.Statement.Context
	economy, err := settings.Economy.Get(ctx)
	if err != nil {
		logger.ErrorF(ctx, "[Economy] grant signup to %d failed: %v", user.ID, err)
		return nil
	}
	if economy.SignupGrant == 0 {
		return nil
	}

	userID := strconv.FormatUint(user.ID, 10)
	err = tx.Transaction(func(tx *gorm.DB) error {
		_, _, err := wallet.Change(tx, &wallet.ChangeRequest{
			UserID:         user.ID,
			Currency:       economy.BaseCurrency,
			Amount:         int64(economy.SignupGrant),
			BizType:        wallet.BizSignupGrant,
			BizID:          userID,
			IdempotencyKey: signupGrantKeyPrefix + userID,
		})
		return err
	})
	if err != nil {
		logger.ErrorF(ctx, "[Economy] grant signup to %d failed: %v", user.ID, err)
	}
	return nil
}
