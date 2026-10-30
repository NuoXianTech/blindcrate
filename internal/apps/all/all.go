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

// Package all 导入全部业务模块, 让各模块 init 中的登记(玩法、任务、条件、统计指标、业务类型等)在每个进程中都生效
//
// api、worker 和 scheduler 的入口都导入这个包; 新增模块时在这里加一行。
package all

import (
	_ "github.com/NuoXianTech/blindcrate/internal/apps/auction"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/audit"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/auth"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/condition"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/economy"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/effect"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/item"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/leaderboard"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/ledger"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/loot"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/match"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/ranked"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/reward"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/season"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/settings"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/shop"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/stats"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/upload"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/venue"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/wallet"
)
