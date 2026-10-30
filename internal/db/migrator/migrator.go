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

package migrator

import (
	"context"
	"embed"
	"errors"
	"log"

	"github.com/NuoXianTech/blindcrate/internal/apps/auction"
	"github.com/NuoXianTech/blindcrate/internal/apps/audit"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/mailbox"
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"github.com/NuoXianTech/blindcrate/internal/apps/ranked"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/shop"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/apps/upload"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migrate 执行数据库迁移
//
// 版本化 SQL 迁移是表结构的唯一来源, 所有环境都会执行;
// 开发环境额外跑一次 AutoMigrate, 方便在写正式 SQL 之前快速迭代模型。
func Migrate() {
	if !config.Config.Database.Enabled {
		return
	}

	runVersioned()

	if config.Config.App.Env == "development" {
		runAutoMigrate()
	}
}

func runVersioned() {
	ctx := context.Background()
	sqlDB, err := db.DB(ctx).DB()
	if err != nil {
		log.Fatalf("[PostgreSQL] load sql db failed: %v\n", err)
	}
	// 从 GORM 连接池借出一个连接; WithConnection 模式下 Close 只归还该连接, 不会关闭连接池
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		log.Fatalf("[PostgreSQL] acquire migrate connection failed: %v\n", err)
	}

	src, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		log.Fatalf("[PostgreSQL] load migrations failed: %v\n", err)
	}
	driver, err := migratepg.WithConnection(ctx, conn, &migratepg.Config{})
	if err != nil {
		log.Fatalf("[PostgreSQL] init migrate driver failed: %v\n", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		log.Fatalf("[PostgreSQL] init migrate failed: %v\n", err)
	}
	defer m.Close()

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("[PostgreSQL] versioned migrate failed: %v\n", err)
	}
	version, dirty, _ := m.Version()
	log.Printf("[PostgreSQL] versioned migrate success, version=%d dirty=%v\n", version, dirty)
}

func runAutoMigrate() {
	if err := db.DB(context.Background()).AutoMigrate(
		&auth.User{},
		&auth.UserIdentity{},
		&audit.Log{},
		&wallet.Currency{},
		&wallet.Balance{},
		&wallet.Transaction{},
		&settings.Setting{},
		&settings.Revision{},
		&item.Quality{},
		&item.Item{},
		&inventory.UserItem{},
		&inventory.Transaction{},
		&inventory.SellOrder{},
		&inventory.SellOrderItem{},
		&loot.Pool{},
		&loot.Entry{},
		&match.Match{},
		&match.Player{},
		&stats.Stat{},
		&leaderboard.Board{},
		&season.Season{},
		&ranked.Rating{},
		&ranked.Tier{},
		&venue.Venue{},
		&auction.Match{},
		&auction.Player{},
		&auction.Bid{},
		&auction.WarehouseItem{},
		&auction.Use{},
		&auction.SuspectReview{},
		&shop.Item{},
		&shop.Order{},
		&upload.Upload{},
		&pack.Import{},
		&mailbox.Mail{},
		&mailbox.Broadcast{},
	); err != nil {
		log.Fatalf("[PostgreSQL] auto migrate failed: %v\n", err)
	}
	log.Printf("[PostgreSQL] auto migrate (development) success\n")
}
