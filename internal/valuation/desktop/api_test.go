package desktop

import (
	"errors"
	"myinvest/internal/valuation"
	"strings"
	"testing"
)

func TestAPIValidationAndPrivateErrors(t *testing.T) {
	called := false
	a := NewAPI(func(func(*valuation.Service) error) error { called = true; return errors.New("private SQL path") })
	if _, err := a.GetOverview("2026-02-30"); err == nil || called {
		t.Fatal(err, called)
	}
	if _, err := a.GetOverview("2026-01-01"); err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal(err)
	}
}

func TestAnalysisValidationAndPrivateErrors(t *testing.T) {
	called := false
	a := NewAPI(func(func(*valuation.Service) error) error { called = true; return errors.New("private SQL path") })
	for _, args := range [][2]string{{"", "2026-01-01"}, {"-1", "2026-01-01"}, {"1 OR 1=1", "2026-01-01"}, {"1", "2026-02-30"}} {
		if _, err := a.GetPortfolioAnalysis(args[0], args[1]); err == nil || called {
			t.Fatal(args, err, called)
		}
	}
	if _, err := a.GetPortfolioAnalysis("1", "2026-01-01"); err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal(err)
	}
}
