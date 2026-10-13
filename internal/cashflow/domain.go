// Package cashflow defines the unified portfolio operation history.
package cashflow

import (
	"errors"
	"math/big"
	"regexp"
	"strings"
	"time"
)

type Input struct {
	Kind, Date, CurrencyCode, Amount, PortfolioAmount, Comment string
	AssetID, Quantity, Expenses                                string
}
type Operation struct {
	ID, PortfolioID      string
	Ticker, AveragePrice string
	Rounded              bool
	Input
}
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Message }

var ErrNotFound = errors.New("Операция не найдена в выбранном портфеле")
var ErrPortfolio = errors.New("Портфель не найден")
var decimal = regexp.MustCompile(`^[0-9]+([.][0-9]+)?$`)

func Amount(value string) (string, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
	if !decimal.MatchString(value) || strings.Trim(value, "0.") == "" {
		return "", errors.New("Введите сумму больше нуля: цифры и точка или запятая")
	}
	return exact(rational(value)), nil
}
func rational(value string) *big.Rat {
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		panic("invalid validated decimal")
	}
	return r
}

// exact only receives sums of finite decimal inputs. No rounding occurs.
func exact(r *big.Rat) string {
	d := new(big.Int).Set(r.Denom())
	scale := 0
	for _, factor := range []int64{2, 5} {
		count := 0
		f := big.NewInt(factor)
		for new(big.Int).Mod(d, f).Sign() == 0 {
			d.Quo(d, f)
			count++
		}
		if count > scale {
			scale = count
		}
	}
	return trim(r.FloatString(scale))
}
func trim(s string) string {
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}
func Rate(amount, portfolioAmount string) (string, error) {
	a, err := Amount(amount)
	if err != nil {
		return "", err
	}
	b, err := Amount(portfolioAmount)
	if err != nil {
		return "", err
	}
	r := new(big.Rat).Quo(rational(b), rational(a))
	n := new(big.Int).Mul(r.Num(), new(big.Int).Exp(big.NewInt(10), big.NewInt(4), nil))
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(n, r.Denom(), rem)
	if rem.Lsh(rem, 1).Cmp(r.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	return exact(new(big.Rat).SetFrac(q, new(big.Int).Exp(big.NewInt(10), big.NewInt(4), nil))), nil
}
func Validate(in Input, portfolioCurrency string) (Input, error) {
	if in.Kind != "deposit" && in.Kind != "withdrawal" {
		return in, &FieldError{"kind", "Выберите пополнение или вывод"}
	}
	d, err := time.Parse("2006-01-02", in.Date)
	if err != nil || d.Year() < 1 || d.Format("2006-01-02") != in.Date {
		return in, &FieldError{"date", "Введите существующую календарную дату YYYY-MM-DD"}
	}
	if in.CurrencyCode == "" {
		return in, &FieldError{"currencyCode", "Выберите валюту"}
	}
	in.Amount, err = Amount(in.Amount)
	if err != nil {
		return in, &FieldError{"amount", err.Error()}
	}
	in.PortfolioAmount, err = Amount(in.PortfolioAmount)
	if err != nil {
		return in, &FieldError{"portfolioAmount", err.Error()}
	}
	if in.CurrencyCode == portfolioCurrency && in.Amount != in.PortfolioAmount {
		return in, &FieldError{"portfolioAmount", "При одинаковых валютах суммы должны совпадать"}
	}
	return in, nil
}
func Balance(operations []Operation) string {
	total := new(big.Rat)
	for _, op := range operations {
		total.Add(total, rational(MoneyChange(op)))
	}
	return exact(total)
}
