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

package reward

import (
	"errors"
	"fmt"
	"testing"

	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
)

func TestFailureOf(t *testing.T) {
	_, tooLarge := Bundle{Currencies: []CurrencyLine{{Currency: "chip", Amount: wallet.MaxAmount}, {Currency: "chip", Amount: 1}}}.Normalize()
	cases := []struct {
		name string
		err  error
		want Failure
	}{
		{"no error", nil, Delivered},
		{"nothing to deliver", errNothing, Unavailable},
		{"wrapped nothing to deliver", fmt.Errorf("grant user 1: %w", errNothing), Unavailable},
		{"item deleted", errors.New(item.ItemNotFound), Unavailable},
		{"item delisted", errors.New(item.ItemDisabled), Unavailable},
		{"currency missing", errors.New(wallet.CurrencyNotFound), Unavailable},
		{"currency disabled", errors.New(wallet.CurrencyDisabled), Unavailable},
		{"amount over the limit", errors.New(wallet.AmountInvalid), TooLarge},
		{"quantity over the limit", errors.New(inventory.QuantityInvalid), TooLarge},
		{"normalize over the limit", tooLarge, TooLarge},
		{"insufficient balance", errors.New(wallet.InsufficientBalance), Delivered},
		{"insufficient quantity", errors.New(inventory.InsufficientQuantity), Delivered},
		{"idempotency conflict", errors.New(IdempotencyConflict), Delivered},
		{"other", errors.New("connection reset"), Delivered},
	}
	for _, c := range cases {
		if got := FailureOf(c.err); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}
