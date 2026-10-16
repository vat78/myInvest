package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"myinvest/internal/benchmarks"
	benchmarksdesktop "myinvest/internal/benchmarks/desktop"
	benchmarkssqlite "myinvest/internal/benchmarks/sqlite"
	"myinvest/internal/currency"
	"myinvest/internal/currency/desktop"
	currencysqlite "myinvest/internal/currency/sqlite"
	"myinvest/internal/operations"
	operationsdesktop "myinvest/internal/operations/desktop"
	operationssqlite "myinvest/internal/operations/sqlite"
	"myinvest/internal/planning"
	planningdesktop "myinvest/internal/planning/desktop"
	planningsqlite "myinvest/internal/planning/sqlite"
	"myinvest/internal/portfolio"
	portfoliodesktop "myinvest/internal/portfolio/desktop"
	portfoliosqlite "myinvest/internal/portfolio/sqlite"
	"myinvest/internal/quotes"
	quotesdesktop "myinvest/internal/quotes/desktop"
	quotessqlite "myinvest/internal/quotes/sqlite"
	"myinvest/internal/storage/sqlite"
	"myinvest/internal/valuation"
	valuationdesktop "myinvest/internal/valuation/desktop"
	valuationsqlite "myinvest/internal/valuation/sqlite"
	"myinvest/migrations"
)

type App struct {
	planning   *planningdesktop.PlanningAPI
	comparison *valuationdesktop.ComparisonAPI
	benchmarks *benchmarksdesktop.BenchmarksAPI
	allocation *valuationdesktop.AllocationAPI
	quotes     *quotesdesktop.QuotesAPI
	overview   *valuationdesktop.OverviewAPI
	cashFlows  *operationsdesktop.CashFlowAPI
	ready      chan struct{}
	mu         sync.Mutex
	store      *sqlite.Store
	initErr    error
	currencies *desktop.API
	portfolios *portfoliodesktop.PortfolioAPI
}

type AppInfo struct {
	Name string `json:"name"`
}

func newApp() *App {
	a := &App{ready: make(chan struct{})}
	a.planning = planningdesktop.NewAPI(func(fn func(*planning.Service) error) error {
		return a.withStorage(func(db *sql.DB) error { return fn(planning.NewService(planningsqlite.NewRepository(db))) })
	})
	a.comparison = valuationdesktop.NewComparisonAPI(func(fn func(*valuation.ComparisonService) error) error {
		return a.withStorage(func(db *sql.DB) error { return fn(valuation.NewComparisonService(valuationsqlite.NewRepository(db))) })
	})
	a.benchmarks = benchmarksdesktop.NewAPI(func(fn func(*benchmarks.Service) error) error {
		return a.withStorage(func(db *sql.DB) error { return fn(benchmarks.NewService(benchmarkssqlite.NewRepository(db))) })
	})
	a.currencies = desktop.NewAPI(func(fn func(*currency.Service) error) error {
		return a.withStorage(func(db *sql.DB) error {
			return fn(currency.NewService(currencysqlite.NewRepository(db)))
		})
	})
	a.portfolios = portfoliodesktop.NewAPI(func(fn func(*portfolio.Service) error) error {
		return a.withStorage(func(db *sql.DB) error { return fn(portfolio.NewService(portfoliosqlite.NewRepository(db))) })
	})
	a.cashFlows = operationsdesktop.NewAPI(func(fn func(*operations.Service) error) error {
		return a.withStorage(func(db *sql.DB) error { return fn(operations.NewService(operationssqlite.NewRepository(db))) })
	})
	a.overview = valuationdesktop.NewAPI(func(fn func(*valuation.Service) error) error {
		return a.withStorage(func(db *sql.DB) error { return fn(valuation.NewService(valuationsqlite.NewRepository(db))) })
	})
	a.quotes = quotesdesktop.NewAPI(func(fn func(*quotes.Service) error) error {
		return a.withStorage(func(db *sql.DB) error { return fn(quotes.NewService(quotessqlite.NewRepository(db))) })
	})
	a.allocation = valuationdesktop.NewAllocationAPI(func(fn func(*valuation.AllocationService) error) error {
		return a.withStorage(func(db *sql.DB) error { return fn(valuation.NewAllocationService(valuationsqlite.NewRepository(db))) })
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
