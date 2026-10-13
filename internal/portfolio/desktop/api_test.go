package desktop

import (
	"context"
	"errors"
	"myinvest/internal/portfolio"
	portfoliosqlite "myinvest/internal/portfolio/sqlite"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/migrations"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPIValidationAndPersistence(t *testing.T) {
	store, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "db.sqlite3"), migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := portfolio.NewService(portfoliosqlite.NewRepository(store.DB()))
	api := NewAPI(func(fn func(*portfolio.Service) error) error { return fn(service) })
	input := PortfolioInputDTO{Name: "Основной", OpenedOn: "2024-02-29", CurrencyCode: "RUB"}
	created := api.CreatePortfolio(input)
	if created.Message != "" || len(created.Errors) != 0 || created.Portfolio.ID == "" {
		t.Fatalf("create: %+v", created)
	}
	input.CurrencyCode = "USD"
	result := api.UpdatePortfolio(created.Portfolio.ID, input)
	if result.Errors["currencyCode"] == "" {
		t.Fatalf("currency changed: %+v", result)
	}
	input.CurrencyCode = "RUB"
	input.Name = "основной"
	if result = api.CreatePortfolio(input); result.Errors["name"] != "Портфель с таким названием уже существует" {
		t.Fatalf("duplicate: %+v", result)
	}
	input.Name = "Edited"
	input.OpenedOn = "2099-01-01"
	result = api.UpdatePortfolio(created.Portfolio.ID, input)
	if result.Portfolio.ID != created.Portfolio.ID || result.Portfolio.CurrencyCode != "RUB" {
		t.Fatalf("edit: %+v", result)
	}
	for _, id := range []string{"", "bad", "0", "-1", "01", "999"} {
		result := api.UpdatePortfolio(id, input)
		if result.Message != "Портфель не найден" {
			t.Fatalf("missing id %s: %+v", id, result)
		}
	}
	rows, err := api.ListPortfolios()
	if err != nil || len(rows) != 1 || rows[0] != result.Portfolio {
		t.Fatalf("list %+v %v", rows, err)
	}
}
func TestAPIDoesNotExposeStorageDetails(t *testing.T) {
	api := NewAPI(func(func(*portfolio.Service) error) error { return errors.New("SQL secret path") })
	if _, err := api.ListPortfolios(); err == nil || strings.Contains(err.Error(), "SQL") {
		t.Fatalf("list error: %v", err)
	}
	result := api.CreatePortfolio(PortfolioInputDTO{})
	if result.Message == "" || strings.Contains(result.Message, "SQL") {
		t.Fatalf("save error: %+v", result)
	}
}
