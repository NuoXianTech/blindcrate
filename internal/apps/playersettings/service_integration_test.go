//go:build integration

package playersettings

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
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/db"
)

func TestConcurrentSettingsUpdates(t *testing.T) {
	name := os.Getenv("BLINDCRATE_TEST_DATABASE")
	if name == "" {
		t.Skip("requires explicit test database")
	}
	if name != config.Config.Database.Database || (!strings.Contains(name, "_test") && !strings.Contains(name, "_e2e_")) {
		t.Fatal("refusing to run outside the named test database")
	}
	ctx := context.Background()
	tx := db.DB(ctx)
	user := auth.User{Username: fmt.Sprintf("settings_test_%d", time.Now().UnixNano())}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		tx.Where("user_id = ?", user.ID).Delete(&Settings{})
		tx.Where("id = ?", user.ID).Delete(&auth.User{})
	})
	initial, err := Read(tx, user.ID)
	if err != nil || initial.Version != 0 || initial.KeepOneCollectible || initial.UpdatedAt != nil {
		t.Fatalf("default: %+v %v", initial, err)
	}
	var count int64
	tx.Model(&Settings{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 0 {
		t.Fatal("read must not create row")
	}
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Go(func() {
			<-start
			version, keep := int64(0), true
			_, _, err := Update(ctx, user.ID, &UpdateRequest{Version: &version, KeepOneCollectible: &keep})
			errors <- err
		})
	}
	close(start)
	wg.Wait()
	close(errors)
	passed, conflicts := 0, 0
	for err := range errors {
		if err == nil {
			passed++
		} else if fault.KindOf(err) == fault.KindConflict {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if passed != 1 || conflicts != 1 {
		t.Fatalf("updates: %d successful, %d conflicting", passed, conflicts)
	}
	current, err := Read(tx, user.ID)
	if err != nil || current.Version != 1 || !current.KeepOneCollectible || current.UpdatedAt == nil {
		t.Fatalf("persisted: %+v %v", current, err)
	}
	version, keep := current.Version, true
	unchanged, changed, err := Update(ctx, user.ID, &UpdateRequest{Version: &version, KeepOneCollectible: &keep})
	if err != nil || changed || unchanged.Version != version {
		t.Fatalf("no-op: %+v %t %v", unchanged, changed, err)
	}
	keep = false
	disabled, changed, err := Update(ctx, user.ID, &UpdateRequest{Version: &version, KeepOneCollectible: &keep})
	if err != nil || !changed || disabled.KeepOneCollectible || disabled.Version != version+1 {
		t.Fatalf("disable: %+v %t %v", disabled, changed, err)
	}
}
