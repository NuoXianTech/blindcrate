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

package walletadmin

import (
	"errors"

	"github.com/NuoXianTech/blindcrate/internal/apps/fault"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CurrencyPack 配置包中的一种货币, 按代码对应
type CurrencyPack struct {
	Code string `json:"code" binding:"required,max=32"`
	CurrencyRequest
}

func init() {
	pack.RegisterTable(pack.Table[wallet.Currency, CurrencyPack]{
		Code: pack.PackCurrencies, Label: "货币", Hint: "按代码对应", Order: pack.OrderCurrencies,
		Query: func(tx *gorm.DB) *gorm.DB { return tx.Order("sort_order, code") },
		Pack: func(c *wallet.Currency, _ func(string) string) CurrencyPack {
			return CurrencyPack{Code: c.Code, CurrencyRequest: CurrencyRequest{
				Name:      c.Name,
				Decimals:  c.Decimals,
				SortOrder: c.SortOrder,
				IsEnabled: c.IsEnabled,
			}}
		},
		Key:   func(req *CurrencyPack) string { return req.Code },
		Name:  func(req *CurrencyPack) string { return req.Name },
		Taken: wallet.CurrencyTaken,
		Find: func(tx *gorm.DB, req *CurrencyPack, current *wallet.Currency) (bool, error) {
			err := tx.Where("code = ?", req.Code).Limit(1).Find(current).Error
			return current.Code != "", err
		},
		Lock: func(tx *gorm.DB, current *wallet.Currency) error {
			if err := current.Exact(tx.Clauses(clause.Locking{Strength: "UPDATE"}), current.Code); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fault.NotFound(wallet.CurrencyNotFound)
				}
				return err
			}
			return nil
		},
		Save: func(tx *gorm.DB, _ *pack.Run, current *wallet.Currency, req *CurrencyPack) error {
			if current == nil {
				target := &wallet.Currency{Code: req.Code}
				ApplyCurrency(target, &req.CurrencyRequest)
				if err := wallet.ValidateCurrency(target); err != nil {
					return err
				}
				return tx.Create(target).Error
			}
			old := *current
			ApplyCurrency(current, &req.CurrencyRequest)
			if err := CheckCurrencyChange(tx, &old, current); err != nil {
				return err
			}
			return tx.Save(current).Error
		},
	})
}
