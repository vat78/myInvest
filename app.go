package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"myinvest/internal/currency"
	"myinvest/internal/currency/desktop"
	currencysqlite "myinvest/internal/currency/sqlite"
	"myinvest/internal/storage/sqlite"
	"myinvest/migrations"
)

type App struct {
	ready      chan struct{}
	mu         sync.Mutex
	store      *sqlite.Store
	initErr    error
	currencies *desktop.API
}

type AppInfo struct {
	Name string `json:"name"`
}

func newApp() *App {
	a := &App{ready: make(chan struct{})}
	a.currencies = desktop.NewAPI(func(fn func(*currency.Service) error) error {
		return a.withStorage(func(db *sql.DB) error {
			return fn(currency.NewService(currencysqlite.NewRepository(db)))
		})
	})
	return a
}

// Guard bound APIs against incomplete initialization and concurrent shutdown.
func (a *App) withStorage(fn func(*sql.DB) error) error {
	if a.ready != nil {
		<-a.ready
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.initErr != nil {
		return fmt.Errorf("база данных не готова: %w", a.initErr)
	}
	if a.store == nil {
		return fmt.Errorf("база данных ещё не подготовлена")
	}
	return fn(a.store.DB())
}

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
