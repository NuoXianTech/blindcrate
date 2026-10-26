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
	"log"

	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

// Models 所有需要建表的模型都在这里登记
//
// AutoMigrate 只做加法：建表、加列、放宽长度、建索引、加约束。
// 它不会删列、不会改名、也不会感知字段重命名，这几种情况必须写进 Migrations。
var Models = []interface{}{}

// Migrations 需要显式版本化执行的变更
//
// 适合放 AutoMigrate 做不到的事：删列、改字段名、拆分数据、回填默认值。
// ID 用「序号_说明」，一旦提交就不允许再改，改了会导致已上线的库重复执行或漏执行。
var Migrations = []*gormigrate.Migration{
	// 示例：
	// {
	// 	ID: "0001_rename_user_nick",
	// 	Migrate: func(tx *gorm.DB) error {
	// 		return tx.Exec("ALTER TABLE users RENAME COLUMN nick TO nickname").Error
	// 	},
	// 	Rollback: func(tx *gorm.DB) error {
	// 		return tx.Exec("ALTER TABLE users RENAME COLUMN nickname TO nick").Error
	// 	},
	// },
}

// advisoryLockKey 迁移期间的全局互斥锁
//
// 多副本同时启动时，只让一个副本执行迁移，其余副本等待它结束后再走一遍空跑。
const advisoryLockKey int64 = 0x626C696E64637261

func Migrate() {
	if !config.Config.Database.Enabled {
		return
	}

	ctx := context.Background()
	conn := db.DB(ctx)

	if err := conn.Exec("SELECT pg_advisory_lock(?)", advisoryLockKey).Error; err != nil {
		log.Fatalf("[PostgreSQL] acquire migrate lock failed: %v\n", err)
	}
	defer func() {
		if err := conn.Exec("SELECT pg_advisory_unlock(?)", advisoryLockKey).Error; err != nil {
			log.Printf("[PostgreSQL] release migrate lock failed: %v\n", err)
		}
	}()

	// 显式变更先跑，避免 AutoMigrate 在改名之前就把新列建出来
	if len(Migrations) > 0 {
		m := gormigrate.New(conn, gormigrate.DefaultOptions, Migrations)
		if err := m.Migrate(); err != nil {
			log.Fatalf("[PostgreSQL] migrate failed: %v\n", err)
		}
		log.Printf("[PostgreSQL] migrate success\n")
	}

	// 再做结构对齐
	if len(Models) > 0 {
		if err := conn.AutoMigrate(Models...); err != nil {
			log.Fatalf("[PostgreSQL] auto migrate failed: %v\n", err)
		}
		log.Printf("[PostgreSQL] auto migrate success\n")
	}
}
