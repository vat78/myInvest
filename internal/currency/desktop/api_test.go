package desktop

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"myinvest/internal/currency"
	currencysqlite "myinvest/internal/currency/sqlite"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/migrations"
)

func TestCurrencyBridge(t *testing.T) {
	var service *currency.Service
	app := NewAPI(func(fn func(*currency.Service) error) error {
		if service == nil {
			return fmt.Errorf("база данных ещё не подготовлена")
		}
		return fn(service)
	})
	if _, err := app.ListCurrencies(); err == nil {
		t.Fatal("unready bridge accepted")
	}
	store, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "test.sqlite3"), migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service = currency.NewService(currencysqlite.NewRepository(store.DB()))
	codes, err := app.ListCurrencies()
	if err != nil || len(codes) != 2 {
		t.Fatalf("initial directory: %v %v", codes, err)
	}
	if code, err := app.AddCurrency(" eur "); err != nil || code != "EUR" {
		t.Fatalf("add: %s %v", code, err)
	}
	pair := CurrencyPairDTO{Date: "2026-10-12", Base: "USD", Quote: "RUB"}
	if rate, err := app.GetCurrencyRate(pair); err != nil || rate.Found {
		t.Fatalf("initial rate: %+v %v", rate, err)
	}
	value := "95,123456789012345678901234567890000"
	if err := app.SaveCurrencyRate(SaveCurrencyRateDTO{Date: pair.Date, Base: "usd", Quote: " rub ", Value: value}); err != nil {
		t.Fatal(err)
	}
	if rate, err := app.GetCurrencyRate(pair); err != nil || !rate.Found || rate.Value != "95.123456789012345678901234567890000" {
		t.Fatalf("exact DTO: %+v %v", rate, err)
	}
	if err := app.SaveCurrencyRate(SaveCurrencyRateDTO{Date: pair.Date, Base: "USD", Quote: "RUB", Value: "0"}); err == nil {
		t.Fatal("invalid DTO accepted")
	}
	inverse, err := app.GetCurrencyRate(CurrencyPairDTO{Date: pair.Date, Base: "RUB", Quote: "USD"})
	if err != nil || !inverse.Found || inverse.Value != "0.010512654120823639" {
		t.Fatalf("inverse DTO: %+v %v", inverse, err)
	}
	service = nil
	if _, err := app.ListCurrencies(); err == nil {
		t.Fatal("closed bridge accepted")
	}
	if _, err := app.AddCurrency("GBP"); err == nil {
		t.Fatal("closed add accepted")
	}
	if _, err := app.GetCurrencyRate(pair); err == nil {
		t.Fatal("closed read accepted")
	}
	if err := app.SaveCurrencyRate(SaveCurrencyRateDTO{}); err == nil {
		t.Fatal("closed save accepted")
	}
}
