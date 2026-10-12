package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(original) {
		t.Fatal("failed initialization changed original data")
	}
}
