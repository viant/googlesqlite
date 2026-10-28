package googlesqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
)

func assertBackendCached(t *testing.T, name string, want bool) {
	t.Helper()
	nameToValueMapMu.Lock()
	defer nameToValueMapMu.Unlock()
	_, db := nameToDBMap[name]
	_, catalog := nameToCatalogMap[name]
	if db != want || catalog != want {
		t.Fatalf("backend retained=%v catalog retained=%v want=%v", db, catalog, want)
	}
}
func TestFileLifecycleReleasesLastOwnerAndPreservesDiskData(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "durable.db")
	first, err := sql.Open("googlesqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	second, err := sql.Open("googlesqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = first.ExecContext(ctx, "CREATE TABLE durable_value(value INT64)"); err != nil {
		t.Fatal(err)
	}
	if _, err = first.ExecContext(ctx, "INSERT INTO durable_value VALUES(42)"); err != nil {
		t.Fatal(err)
	}
	assertBackendCached(t, dsn, true)
	if err = first.Close(); err != nil {
		t.Fatal(err)
	}
	assertBackendCached(t, dsn, true)
	var value int64
	if err = second.QueryRowContext(ctx, "SELECT value FROM durable_value").Scan(&value); err != nil || value != 42 {
		t.Fatalf("remaining owner: %d %v", value, err)
	}
	if err = second.Close(); err != nil {
		t.Fatal(err)
	}
	assertBackendCached(t, dsn, false)
	reopened, err := sql.Open("googlesqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err = reopened.QueryRowContext(ctx, "SELECT value FROM durable_value").Scan(&value); err != nil || value != 42 {
		t.Fatalf("disk reopen: %d %v", value, err)
	}
	reopened.Close()
	assertBackendCached(t, dsn, false)
}
func TestRepeatedFileLifecycleDoesNotRetainBackends(t *testing.T) {
	ctx := context.Background()
	for i := 0; i < 8; i++ {
		dsn := "file:" + filepath.Join(t.TempDir(), "lifecycle.db")
		db, err := sql.Open("googlesqlite", dsn)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.ExecContext(ctx, "SELECT 1"); err != nil {
			t.Fatal(err)
		}
		if err = db.Close(); err != nil {
			t.Fatal(err)
		}
		assertBackendCached(t, dsn, false)
	}
}
func TestLegacyDriverOpenReleasesBackend(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "legacy.db")
	conn, err := new(Driver).Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err = conn.Close(); err != nil {
		t.Fatal(err)
	}
	assertBackendCached(t, dsn, false)
}

func TestFileLifecycleConcurrentOwners(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "shared.db")
	seed, err := sql.Open("googlesqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = seed.ExecContext(ctx, "CREATE TABLE shared_value(value INT64)"); err != nil {
		t.Fatal(err)
	}
	if _, err = seed.ExecContext(ctx, "INSERT INTO shared_value VALUES(42)"); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() {
			db, err := sql.Open("googlesqlite", dsn)
			if err != nil {
				done <- err
				return
			}
			var value int64
			err = db.QueryRowContext(ctx, "SELECT value FROM shared_value").Scan(&value)
			if err == nil && value != 42 {
				err = fmt.Errorf("shared data changed: %d", value)
			}
			closeErr := db.Close()
			if err == nil {
				err = closeErr
			}
			done <- err
		}()
	}
	for i := 0; i < 8; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	var value int64
	if err = seed.QueryRowContext(ctx, "SELECT value FROM shared_value").Scan(&value); err != nil || value != 42 {
		t.Fatalf("live owner lost backend: %d %v", value, err)
	}
	seed.Close()
	assertBackendCached(t, dsn, false)
}
func TestFailedLegacyOpenReleasesBackend(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "hook.db")
	driver := &Driver{ConnectHook: func(*Conn) error { return fmt.Errorf("rejected hook") }}
	if _, err := driver.Open(dsn); err == nil {
		t.Fatal("hook failure lost")
	}
	assertBackendCached(t, dsn, false)
}
