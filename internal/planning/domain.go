// Package planning models theoretical annual investment plans.
package planning

import (
	"fmt"
	"math/big"
	"myinvest/internal/currency"
	"myinvest/internal/valuation"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Parameters struct {
	CurrencyCode     string `json:"currencyCode"`
	Revision         string `json:"revision"`
	StartYear        string `json:"startYear"`
	Years            string `json:"years"`
	Mode             string `json:"mode"`
	Goal             string `json:"goal"`
	Contribution     string `json:"contribution"`
	ReturnPercent    string `json:"returnPercent"`
	InflationPercent string `json:"inflationPercent"`
	BenchmarkID      string `json:"benchmarkID"`
	AllowEstimated   bool   `json:"allowEstimated"`
	InitialCapital   string `json:"initialCapital"`
}
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Message }
func field(f, m string) error       { return &FieldError{f, m} }

var decimal = regexp.MustCompile(`^-?[0-9]+([.][0-9]+)?$`)

func rat(s string) *big.Rat { r, _ := new(big.Rat).SetString(s); return r }
func Validate(p Parameters) (Parameters, error) {
	code, err := currency.Code(p.CurrencyCode)
	if err != nil {
		return p, field("currencyCode", "Выберите валюту")
	}
	p.CurrencyCode = code
	if p.Revision != "" && valuation.ValidatePortfolioID(p.Revision) != nil {
		return p, field("revision", "Некорректная версия плана")
	}
	y, e := strconv.Atoi(p.StartYear)
	if e != nil || y < 2 || y > 9998 {
		return p, field("startYear", "Начальный год: от 2 до 9998")
	}
	n, e := strconv.Atoi(p.Years)
	if e != nil || n < 1 || n > 200 || y+n > 9999 {
		return p, field("years", "Горизонт: 1–200 лет, конечный год не позднее 9999")
	}
	p.StartYear = strconv.Itoa(y)
	p.Years = strconv.Itoa(n)
	if p.Mode != "goal" && p.Mode != "contribution" {
		return p, field("mode", "Выберите режим расчёта")
	}
	if valuation.ValidatePortfolioID(p.BenchmarkID) != nil {
		return p, field("benchmarkID", "Выберите индексный бенчмарк покупательной способности валюты плана")
	}
	for _, v := range []struct {
		name         string
		value        *string
		rate, signed bool
	}{
		{"goal", &p.Goal, false, false}, {"contribution", &p.Contribution, false, false},
		{"returnPercent", &p.ReturnPercent, true, true}, {"inflationPercent", &p.InflationPercent, true, true},
		{"initialCapital", &p.InitialCapital, false, true},
	} {
		if v.name == "goal" && p.Mode != "goal" {
			p.Goal = "0"
			continue
		}
		if v.name == "contribution" && p.Mode != "contribution" {
			p.Contribution = "0"
			continue
		}
		s := strings.ReplaceAll(strings.TrimSpace(*v.value), ",", ".")
		if len(s) > 120 || !decimal.MatchString(s) {
			return p, field(v.name, "Введите десятичное значение (до 120 символов)")
		}
		r := rat(s)
		if (!v.signed && r.Sign() < 0) || (v.rate && r.Cmp(big.NewRat(-100, 1)) <= 0) {
			return p, field(v.name, "Сумма должна быть неотрицательной; ставка — больше −100%")
		}
		*v.value = s
	}
	return p, nil
}

type Factor struct {
	Year   string `json:"year"`
	Value  string `json:"value"` // Exact rational, persisted for reproducibility; never chart input.
	Source string `json:"source"`
}
type Row struct {
	Year         string `json:"year"`
	Before       string `json:"before"`
	Contribution string `json:"contribution"`
	Inflation    string `json:"inflation"`
	Source       string `json:"source"`
	Income       string `json:"income"`
	Capital      string `json:"capital"`
	RealCapital  string `json:"realCapital"`
}
type Point struct {
	Date    string `json:"date"`
	Nominal string `json:"nominal"`
	Real    string `json:"real"`
}
type Scenario struct {
	Complete          bool     `json:"complete"`
	Issues            []string `json:"issues"`
	FirstContribution string   `json:"firstContribution"`
	FinalCapital      string   `json:"finalCapital"`
	FinalRealCapital  string   `json:"finalRealCapital"`
	Excess            string   `json:"excess"`
	GoalReached       bool     `json:"goalReached"`
	Rows              []Row    `json:"rows"`
	Points            []Point  `json:"points"`
	Factors           []Factor `json:"factors"`
}
type IndexPoint struct{ Date, Value string }

func Resolve(p Parameters, points []IndexPoint, today string) ([]Factor, error) {
	d, e := time.Parse("2006-01-02", today)
	if e != nil || d.Format("2006-01-02") != today {
		return nil, field("date", "Введите календарную дату расчёта")
	}
	values := map[string]*big.Rat{}
	for _, pt := range points {
		v := rat(pt.Value)
		if v == nil || v.Sign() <= 0 {
			return nil, fmt.Errorf("invalid stored index")
		}
		values[pt.Date] = v
	}
	y, _ := strconv.Atoi(p.StartYear)
	n, _ := strconv.Atoi(p.Years)
	expected := new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).Quo(rat(p.InflationPercent), big.NewRat(100, 1)))
	factors := []Factor{}
	for k := 0; k < n; k++ {
		year := y + k
		f := Factor{Year: strconv.Itoa(year), Value: expected.RatString(), Source: "expected"}
		if year < d.Year() {
			a, b := values[fmt.Sprintf("%04d-12-31", year-1)], values[fmt.Sprintf("%04d-12-31", year)]
			if a != nil && b != nil {
				f.Value = new(big.Rat).Quo(b, a).RatString()
				f.Source = "historical"
			} else if p.AllowEstimated {
				f.Source = "estimated"
			} else {
				f.Value = ""
				f.Source = "missing"
			}
		}
		factors = append(factors, f)
	}
	return factors, nil
}
func amount(r *big.Rat) string { return valuation.Format(r, 18) }
func ceilCent(r *big.Rat) *big.Rat {
	n := new(big.Int).Mul(r.Num(), big.NewInt(100))
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(n, r.Denom(), rem)
	if rem.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	return new(big.Rat).SetFrac(q, big.NewInt(100))
}

// Calculate never feeds presentation strings back into arithmetic.
func Calculate(p Parameters, initial *big.Rat, factors []Factor) (Scenario, error) {
	p, e := Validate(p)
	if e != nil {
		return Scenario{}, e
	}
	n, _ := strconv.Atoi(p.Years)
	y, _ := strconv.Atoi(p.StartYear)
	if initial == nil || len(factors) != n {
		return Scenario{}, fmt.Errorf("invalid plan snapshot")
	}
	out := Scenario{Complete: true, Issues: []string{}, Rows: []Row{}, Points: []Point{}, Factors: append([]Factor{}, factors...)}
	index := big.NewRat(1, 1)
	indices := []*big.Rat{new(big.Rat).Set(index)}
	usable := n
	for k, f := range factors {
		if f.Year != strconv.Itoa(y+k) {
			return out, fmt.Errorf("invalid factor year")
		}
		if f.Value == "" {
			usable = k
			out.Complete = false
			out.Issues = append(out.Issues, fmt.Sprintf("Нет границ индекса на 31 декабря %d и %d", y+k-1, y+k))
			break
		}
		r := rat(f.Value)
		if r == nil || r.Sign() <= 0 {
			return out, fmt.Errorf("invalid inflation factor")
		}
		index.Mul(index, r)
		indices = append(indices, new(big.Rat).Set(index))
	}
	q := new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).Quo(rat(p.ReturnPercent), big.NewRat(100, 1)))
	a := rat(p.Contribution)
	if p.Mode == "goal" {
		if !out.Complete {
			return out, nil
		}
		// Horner recurrence gives W in O(N) exact operations, without repeated powers.
		w := new(big.Rat)
		power := big.NewRat(1, 1)
		for k := 0; k < n; k++ {
			w.Add(w, indices[k])
			w.Mul(w, q)
			power.Mul(power, q)
		}
		need := new(big.Rat).Sub(new(big.Rat).Mul(rat(p.Goal), index), new(big.Rat).Mul(initial, power))
		a = new(big.Rat)
		if need.Sign() > 0 {
			a = ceilCent(new(big.Rat).Quo(need, w))
		}
	}
	out.FirstContribution = amount(a)
	if p.Mode == "contribution" {
		out.FirstContribution = p.Contribution
	}
	capital := new(big.Rat).Set(initial)
	out.Points = append(out.Points, Point{fmt.Sprintf("%04d-01-01", y), amount(capital), amount(capital)})
	for k := 0; k < usable; k++ {
		contribution := new(big.Rat).Mul(a, indices[k])
		before := amount(capital)
		base := new(big.Rat).Add(capital, contribution)
		income := new(big.Rat).Mul(base, new(big.Rat).Sub(q, big.NewRat(1, 1)))
		capital = new(big.Rat).Add(base, income)
		real := new(big.Rat).Quo(capital, indices[k+1])
		inflation := new(big.Rat).Mul(new(big.Rat).Sub(rat(factors[k].Value), big.NewRat(1, 1)), big.NewRat(100, 1))
		out.Rows = append(out.Rows, Row{strconv.Itoa(y + k), before, amount(contribution), valuation.Format(inflation, 6), factors[k].Source, amount(income), amount(capital), amount(real)})
		out.Points = append(out.Points, Point{fmt.Sprintf("%04d-01-01", y+k+1), amount(capital), amount(real)})
	}
	if out.Complete {
		real := new(big.Rat).Quo(capital, index)
		out.FinalCapital = amount(capital)
		out.FinalRealCapital = amount(real)
		if p.Mode == "goal" {
			excess := new(big.Rat).Sub(real, rat(p.Goal))
			out.GoalReached = excess.Sign() >= 0
			out.Excess = amount(excess)
			if !out.GoalReached {
				return out, fmt.Errorf("rounded contribution failed goal")
			}
		}
	}
	return out, nil
}
