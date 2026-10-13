package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBridgeRequiresPreparedStorage(t *testing.T) {
	app := &App{}
	if _, err := app.GetAppInfo(); err == nil {
		t.Fatal("bridge reports ready before initialization")
	}
	app = newApp()
	t.Setenv("MYINVEST_DATA_DIR", t.TempDir())
	if err := app.initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer app.shutdown(context.Background())
	info, err := app.GetAppInfo()
	if err != nil || info.Name != "myInvest" {
		t.Fatalf("bridge response: %+v %v", info, err)
	}
	app.shutdown(context.Background())
	if _, err := app.GetAppInfo(); err == nil {
		t.Fatal("bridge reports ready after shutdown")
	}
}

func TestInitializationFailureDoesNotExposeReadyBridge(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MYINVEST_DATA_DIR", dir)
	original := []byte("unreadable SQLite content")
	path := filepath.Join(dir, "myinvest.sqlite3")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	app := newApp()
	if err := app.initialize(context.Background()); err == nil {
		t.Fatal("invalid database initialized")
	}
	if _, err := app.GetAppInfo(); err == nil {
		t.Fatal("failed initialization exposed ready bridge")
	}
	if _, err := app.currencies.ListCurrencies(); err == nil {
		t.Fatal("failed initialization exposed currency API")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(original) {
		t.Fatal("failed initialization changed original data")
	}
}

func TestBoundAPIsFollowStorageLifecycle(t *testing.T) {
	t.Setenv("MYINVEST_DATA_DIR", t.TempDir())
	app := newApp()
	result := make(chan error, 1)
	go func() {
		codes, err := app.currencies.ListCurrencies()
		if err == nil && len(codes) != 2 {
			err = fmt.Errorf("initial directory: %v", codes)
		}
		result <- err
	}()
	if err := app.initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer app.shutdown(context.Background())
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("bound API did not become ready")
	}
	app.shutdown(context.Background())
	if _, err := app.currencies.ListCurrencies(); err == nil {
		t.Fatal("bound API accepted closed storage")
	}
}

func TestPortfolioAPIFollowsStorageLifecycle(t *testing.T) {
	t.Setenv("MYINVEST_DATA_DIR", t.TempDir())
	app := newApp()
	result := make(chan error, 1)
	go func() { _, err := app.portfolios.ListPortfolios(); result <- err }()
	if err := app.initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer app.shutdown(context.Background())
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("portfolio API did not become ready")
	}
	app.shutdown(context.Background())
	if _, err := app.portfolios.ListPortfolios(); err == nil {
		t.Fatal("portfolio API accepted closed storage")
	}
}
