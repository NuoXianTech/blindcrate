//go:build integration

package inventory

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/fault"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/playersettings"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/db"
)

// 仅在已迁移的独立测试库运行。显式传 BLINDCRATE_TEST_DATABASE=<数据库名>,
// CONFIG_PATH 指向其配置; 数据库名必须包含 _test 或 _e2e_。
func TestSellCollectionProtection(t *testing.T) {
	name := os.Getenv("BLINDCRATE_TEST_DATABASE")
	if name == "" {
		t.Skip("requires explicit test database")
	}
	if name != config.Config.Database.Database || (!strings.Contains(name, "_test") && !strings.Contains(name, "_e2e_")) {
		t.Fatal("refusing to run outside the named test database")
	}
	ctx := context.Background()
	tx := db.DB(ctx)
	suffix := fmt.Sprint(time.Now().UnixNano())
	user := auth.User{Username: "sell_test_" + suffix}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	currency := wallet.Currency{Code: "test_" + suffix, Name: "test", IsEnabled: true}
	if err := tx.Create(&currency).Error; err != nil {
		t.Fatal(err)
	}
	items := []item.Item{
		{Code: "test", Name: "test_a_" + suffix, Quality: 1, Category: item.CategoryCollectible, SellCurrency: currency.Code, SellPrice: 100},
		{Code: "test", Name: "test_b_" + suffix, Quality: 1, Category: item.CategoryCollectible, SellCurrency: currency.Code, SellPrice: 100},
		{Code: "test", Name: "test_tool_" + suffix, Quality: 1, Category: "consumable", SellCurrency: currency.Code, SellPrice: 100},
	}
	if err := tx.Create(&items).Error; err != nil {
		t.Fatal(err)
	}
	ids := []uint64{items[0].ID, items[1].ID, items[2].ID}
	t.Cleanup(func() {
		// 仅清理本测试创建的账号、物品和货币。
		tx.Exec("DELETE FROM inventory_sell_order_items WHERE order_id IN (SELECT id FROM inventory_sell_orders WHERE user_id = ?)", user.ID)
		for _, table := range []string{"inventory_sell_orders", "inventory_transactions", "inventory_items", "wallet_transactions", "wallet_balances", "player_game_settings"} {
			tx.Table(table).Where("user_id = ?", user.ID).Delete(map[string]interface{}{})
		}
		tx.Unscoped().Where("id IN ?", ids).Delete(&item.Item{})
		tx.Where("code = ?", currency.Code).Delete(&wallet.Currency{})
		tx.Where("id = ?", user.ID).Delete(&auth.User{})
	})
	for i, quantity := range []int64{5, 1, 1} {
		if err := tx.Create(&UserItem{UserID: user.ID, ItemID: ids[i], Quantity: quantity}).Error; err != nil {
			t.Fatal(err)
		}
	}
	preferences, err := playersettings.Read(tx, user.ID)
	if err != nil || preferences.Version != 0 || preferences.KeepOneCollectible {
		t.Fatalf("default: %+v %v", preferences, err)
	}
	update := func(keep bool, version int64) *playersettings.Settings {
		t.Helper()
		next, _, err := playersettings.Update(ctx, user.ID, &playersettings.UpdateRequest{Version: &version, KeepOneCollectible: &keep})
		if err != nil {
			t.Fatal(err)
		}
		return next
	}
	preferences = update(true, 0)
	line := func(index int, quantity int64) SellItemsRequestItem {
		return SellItemsRequestItem{ItemID: ids[index], Quantity: quantity, SellCurrency: currency.Code, SellPrice: 100}
	}
	request := func(lines ...SellItemsRequestItem) *SellItemsRequest {
		return &SellItemsRequest{Items: lines, IdempotencyKey: fmt.Sprint(time.Now().UnixNano())}
	}
	quantity := func(index int) int64 {
		var row UserItem
		if err := tx.Where("user_id = ? AND item_id = ?", user.ID, ids[index]).Take(&row).Error; err != nil {
			t.Fatal(err)
		}
		return row.Quantity
	}
	if _, _, err := doSellItems(ctx, user.ID, request(line(0, 2), line(1, 1))); fault.KindOf(err) != fault.KindConflict {
		t.Fatalf("batch: %v", err)
	}
	if quantity(0) != 5 || quantity(1) != 1 {
		t.Fatal("partial batch committed")
	}
	req := request(line(0, 4), line(2, 1))
	order, _, err := doSellItems(ctx, user.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	if quantity(0) != 1 || quantity(2) != 0 || !order.KeepOneCollectible {
		t.Fatal("wrong protected sale")
	}
	preferences = update(false, preferences.Version)
	replayed, replay, err := doSellItems(ctx, user.ID, req)
	if err != nil || !replay || replayed.ID != order.ID || !replayed.KeepOneCollectible {
		t.Fatalf("replay: %+v %t %v", replayed, replay, err)
	}
	preferences = update(true, preferences.Version)
	if err := tx.Model(&UserItem{}).Where("user_id = ? AND item_id = ?", user.ID, ids[0]).Update("quantity", 2).Error; err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		r := request(line(0, 1))
		r.IdempotencyKey += fmt.Sprint(i)
		wg.Go(func() { <-start; _, _, err := doSellItems(ctx, user.ID, r); errors <- err })
	}
	close(start)
	wg.Wait()
	close(errors)
	passed, rejected := 0, 0
	for err := range errors {
		if err == nil {
			passed++
		} else if fault.KindOf(err) == fault.KindConflict {
			rejected++
		} else {
			t.Fatal(err)
		}
	}
	if passed != 1 || rejected != 1 || quantity(0) != 1 {
		t.Fatal("concurrent sale removed final collectible")
	}
	stale := request(line(1, 1))
	old := preferences.Version - 1
	stale.GameSettingsVersion = &old
	if _, _, err := doSellItems(ctx, user.ID, stale); fault.KindOf(err) != fault.KindConflict {
		t.Fatalf("stale settings: %v", err)
	}
}
