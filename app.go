package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"myinvest/internal/storage/sqlite"
	"myinvest/migrations"
)

type App struct {
	ready   chan struct{}
	mu      sync.Mutex
	store   *sqlite.Store
	initErr error
}

type AppInfo struct {
	Name string `json:"name"`
}

func newApp() *App { return &App{ready: make(chan struct{})} }

func (a *App) initialize(ctx context.Context) error {
	if a.ready != nil {
		defer close(a.ready)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	path, err := sqlite.DefaultPath()
	if err == nil {
		a.store, err = sqlite.Open(ctx, path, migrations.Files)
	}
	a.initErr = err
	return err
}

func (a *App) startup(ctx context.Context) {
	_ = a.initialize(ctx)
}

// Native dialogs must wait until the window and its event loop are ready.
func (a *App) domReady(ctx context.Context) {
	if a.ready != nil {
		<-a.ready
	}
	a.mu.Lock()
	err := a.initErr
	a.mu.Unlock()
	if err != nil {
		_, _ = runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type: runtime.ErrorDialog, Title: "myInvest — ошибка запуска",
			Buttons: []string{"Закрыть"}, DefaultButton: "Закрыть", CancelButton: "Закрыть",
			Message: fmt.Sprintf("Не удалось подготовить локальную базу данных. Приложение будет закрыто.\n\n%s", err),
		})
		runtime.Quit(ctx)
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store != nil {
		if err := a.store.Close(); err != nil {
			runtime.LogErrorf(ctx, "Не удалось закрыть базу: %v", err)
		}
		a.store = nil
	}
}

// GetAppInfo is the minimal typed bridge used to verify that Go is ready.
func (a *App) GetAppInfo() (AppInfo, error) {
	if a.ready != nil {
		<-a.ready
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.initErr != nil {
		return AppInfo{}, fmt.Errorf("база данных не готова: %w", a.initErr)
	}
	if a.store == nil {
		return AppInfo{}, fmt.Errorf("база данных ещё не подготовлена")
	}
	return AppInfo{Name: "myInvest"}, nil
}
