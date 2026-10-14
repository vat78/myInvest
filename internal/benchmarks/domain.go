// Package benchmarks defines independent historical reference series.
package benchmarks

import (
	"math/big"
	"myinvest/internal/currency"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Benchmark struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CurrencyCode string `json:"currencyCode"`
}
type Point struct {
	BenchmarkID string `json:"benchmarkID"`
	Date        string `json:"date"`
	Mode        string `json:"mode"`
	Input       string `json:"input"`
	Value       string `json:"value"`
	Growth      string `json:"growth"` // Empty for the initial value: no previous observation.
}
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Message }
func validID(id string) bool {
	n, e := strconv.ParseInt(id, 10, 64)
	return e == nil && n > 0 && strconv.FormatInt(n, 10) == id
}
func ValidateBenchmark(b Benchmark) (Benchmark, error) {
	b.Name = strings.TrimSpace(b.Name)
	if b.Name == "" || utf8.RuneCountInString(b.Name) > 120 {
		return b, &FieldError{"name", "Введите название до 120 символов"}
	}
	if b.CurrencyCode != "" {
		var err error
		b.CurrencyCode, err = currency.Code(b.CurrencyCode)
		if err != nil {
			return b, &FieldError{"currencyCode", "Выберите валюту"}
		}
	}
	b.ID = ""
	return b, nil
}

var signedDecimal = regexp.MustCompile(`^-?[0-9]+([.][0-9]+)?$`)

func ValidatePoint(p Point) (Point, error) {
	if !validID(p.BenchmarkID) {
		return p, &FieldError{"benchmarkID", "Выберите бенчмарк"}
	}
	d, e := time.Parse("2006-01-02", p.Date)
	if e != nil || d.Year() < 1 || d.Format("2006-01-02") != p.Date {
		return p, &FieldError{"date", "Введите существующую календарную дату YYYY-MM-DD"}
	}
	p.Input = strings.ReplaceAll(strings.TrimSpace(p.Input), ",", ".")
	switch p.Mode {
	case "value":
		p.Input, e = currency.Decimal(p.Input)
		if e != nil {
			return p, &FieldError{"input", "Введите стоимость больше нуля"}
		}
	case "growth":
		if !signedDecimal.MatchString(p.Input) {
			return p, &FieldError{"input", "Введите процент: цифры и точка или запятая; допускается минус"}
		}
		r, _ := new(big.Rat).SetString(p.Input)
		if r.Cmp(big.NewRat(-100, 1)) <= 0 {
			return p, &FieldError{"input", "Прирост должен быть больше −100%"}
		}
		p.Input = exact(r)
	default:
		return p, &FieldError{"mode", "Выберите стоимость или прирост"}
	}
	p.Value = ""
	p.Growth = ""
	return p, nil
}

// exact renders terminating decimals without rounding intermediate calculations.
func exact(r *big.Rat) string {
	scale := 0
	den := new(big.Int).Set(r.Denom())
	rem := new(big.Int)
	for _, factor := range []int64{2, 5} {
		count := 0
		f := big.NewInt(factor)
		for {
			q := new(big.Int)
			q.QuoRem(den, f, rem)
			if rem.Sign() != 0 {
				break
			}
			den = q
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
	if s == "-0" {
		return "0"
	}
	return s
}

// Recalculate expects chronological observations and preserves the entered mode.
// Only derived growth is rounded, half away from zero, to 18 fractional places.
func Recalculate(points []Point) ([]Point, error) {
	result := append([]Point{}, points...)
	var previous *big.Rat
	for i := range result {
		p := &result[i]
		validated, err := ValidatePoint(*p)
		if err != nil {
			return nil, err
		}
		p.Input = validated.Input
		input, _ := new(big.Rat).SetString(p.Input)
		if p.Mode == "growth" {
			if previous == nil {
				return nil, &FieldError{"input", "Для первой даты задайте начальную стоимость"}
			}
			factor := new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).Quo(input, big.NewRat(100, 1)))
			value := new(big.Rat).Mul(previous, factor)
			p.Value = exact(value)
			p.Growth = p.Input
			previous = value
		} else {
			p.Value = p.Input
			p.Growth = ""
			if previous != nil {
				g := new(big.Rat).Sub(new(big.Rat).Quo(input, previous), big.NewRat(1, 1))
				g.Mul(g, big.NewRat(100, 1))
				p.Growth = trim(g.FloatString(18))
			}
			previous = input
		}
	}
	return result, nil
}
