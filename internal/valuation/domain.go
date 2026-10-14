// Package valuation prepares exact dashboard values from persisted daily snapshots.
package valuation

import (
	"errors"
	"fmt"
	"math/big"
	"myinvest/internal/operations"
	"sort"
	"strings"
	"time"
)

type Position struct{ Ticker, TypeCode, Quantity, Numerator, Denominator string }
type Day struct {
	Date, Cash string
	Positions  []Position
}
type Portfolio struct {
	ID, Name, Currency string
	Days               []Day
}
type Rate struct{ Date, Currency, Value string }
type Snapshot struct {
	Portfolios []Portfolio
	Rates      []Rate
}

func ValidateDate(date string) error {
	d, e := time.Parse("2006-01-02", date)
	if e != nil || d.Year() < 1 || d.Format("2006-01-02") != date {
		return errors.New("Введите существующую календарную дату YYYY-MM-DD")
	}
	return nil
}

type Group struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Value   string `json:"value"`
	Percent string `json:"percent"`
}
type Summary struct {
	Value    string   `json:"value"`
	Complete bool     `json:"complete"`
	Negative bool     `json:"negative"`
	Groups   []Group  `json:"groups"`
	Issues   []string `json:"issues"`
}
type Card struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Currency string  `json:"currency"`
	RateDate string  `json:"rateDate"`
	Rate     string  `json:"rate"`
	Summary  Summary `json:"summary"`
}
type Point struct {
	Date     string   `json:"date"`
	Value    string   `json:"value"`
	Complete bool     `json:"complete"`
	Issues   []string `json:"issues"`
}
type Overview struct {
	Date       string  `json:"date"`
	Total      Summary `json:"total"`
	Portfolios []Card  `json:"portfolios"`
	Points     []Point `json:"points"`
}

// Format rounds only at the presentation boundary, half up (ties away from zero).
func Format(r *big.Rat, places int) string {
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)
	n := new(big.Int).Mul(new(big.Int).Abs(r.Num()), scale)
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(n, r.Denom(), rem)
	if rem.Lsh(rem, 1).Cmp(r.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if r.Sign() < 0 {
		q.Neg(q)
	}
	s := new(big.Rat).SetFrac(q, scale).FloatString(places)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	if s == "-0" {
		s = "0"
	}
	return s
}
func decimal(s string) (*big.Rat, error) {
	// Stored values must be decimals, never exponents or rational input.
	if s == "" {
		return nil, errors.New("missing decimal")
	}
	t := strings.TrimPrefix(s, "-")
	parts := strings.Split(t, ".")
	if len(parts) > 2 {
		return nil, errors.New("invalid decimal")
	}
	for _, p := range parts {
		if p == "" {
			return nil, errors.New("invalid decimal")
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return nil, errors.New("invalid decimal")
			}
		}
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, errors.New("invalid decimal")
	}
	return r, nil
}

type amounts struct {
	total    *big.Rat
	groups   map[string]*big.Rat
	negative bool
	issues   []string
}

func empty() amounts {
	return amounts{total: new(big.Rat), groups: map[string]*big.Rat{}, issues: []string{}}
}
func (a *amounts) add(code string, v *big.Rat) {
	if v.Sign() < 0 {
		a.negative = true
	}
	a.total.Add(a.total, v)
	if a.groups[code] == nil {
		a.groups[code] = new(big.Rat)
	}
	a.groups[code].Add(a.groups[code], v)
}
func (a amounts) summary() Summary {
	s := Summary{Complete: len(a.issues) == 0, Negative: a.negative, Groups: []Group{}, Issues: a.issues}
	if s.Complete {
		s.Value = Format(a.total, 18)
	}
	types := operations.AssetTypes()
	types = append(types, operations.AssetType{Code: "unclassified", Name: "Тип не указан"}, operations.AssetType{Code: "cash", Name: "Деньги"})
	for _, t := range types {
		v := a.groups[t.Code]
		if v == nil || v.Sign() == 0 {
			continue
		}
		g := Group{Code: t.Code, Name: t.Name, Value: Format(v, 18)}
		if s.Complete && !s.Negative && a.total.Sign() > 0 {
			g.Percent = Format(new(big.Rat).Mul(new(big.Rat).Quo(v, a.total), big.NewRat(100, 1)), 2)
		}
		s.Groups = append(s.Groups, g)
	}
	return s
}
func dayAmounts(p Portfolio, d Day) (amounts, error) {
	a := empty()
	cash, err := decimal(d.Cash)
	if err != nil {
		return a, err
	}
	a.add("cash", cash)
	for _, pos := range d.Positions {
		q, err := decimal(pos.Quantity)
		if err != nil {
			return a, err
		}
		if q.Sign() < 0 {
			a.negative = true
		}
		if q.Sign() == 0 {
			continue
		}
		if pos.Numerator == "" || pos.Denominator == "" {
			a.issues = append(a.issues, fmt.Sprintf("%s: нет цены %s на %s", p.Name, pos.Ticker, d.Date))
			continue
		}
		n, ok := new(big.Int).SetString(pos.Numerator, 10)
		den, ok2 := new(big.Int).SetString(pos.Denominator, 10)
		if !ok || !ok2 || den.Sign() <= 0 {
			return a, errors.New("invalid stored position value")
		}
		code := pos.TypeCode
		if code == "" {
			code = "unclassified"
		} else if operations.ValidateAssetType(code) != nil {
			return a, errors.New("invalid asset type")
		}
		a.add(code, new(big.Rat).SetFrac(n, den))
	}
	return a, nil
}
func rateAt(rates map[string][]Rate, currency, date string) (Rate, *big.Rat, error) {
	if currency == "RUB" {
		return Rate{Value: "1"}, big.NewRat(1, 1), nil
	}
	list := rates[currency]
	i := sort.Search(len(list), func(i int) bool { return list[i].Date > date }) - 1
	var chosen Rate
	if i >= 0 {
		chosen = list[i]
	}
	if chosen.Date == "" {
		return chosen, nil, nil
	}
	v, err := decimal(chosen.Value)
	if err == nil && v.Sign() <= 0 {
		err = errors.New("invalid rate")
	}
	return chosen, v, err
}

type preparedPortfolio struct {
	portfolio Portfolio
	days      []amounts
}

func at(portfolios []preparedPortfolio, rates map[string][]Rate, date string, cards bool) (amounts, []Card, error) {
	total := empty()
	list := []Card{}
	for _, prepared := range portfolios {
		p := prepared.portfolio
		i := sort.Search(len(p.Days), func(i int) bool { return p.Days[i].Date > date }) - 1
		a := empty()
		if i >= 0 {
			a = prepared.days[i]
		}
		rate, mult, err := rateAt(rates, p.Currency, date)
		if err != nil {
			return total, list, err
		}
		if cards {
			list = append(list, Card{ID: p.ID, Name: p.Name, Currency: p.Currency, RateDate: rate.Date, Rate: rate.Value, Summary: a.summary()})
		}
		total.issues = append(total.issues, a.issues...)
		total.negative = total.negative || a.negative
		// A fully zero portfolio needs no exchange rate.
		nonzero := false
		for _, v := range a.groups {
			if v.Sign() != 0 {
				nonzero = true
			}
		}
		if mult == nil {
			if nonzero {
				total.issues = append(total.issues, fmt.Sprintf("%s: нет курса %s → RUB на %s", p.Name, p.Currency, date))
			}
			continue
		}
		for code, v := range a.groups {
			total.add(code, new(big.Rat).Mul(v, mult))
		}
	}
	return total, list, nil
}
func Build(s Snapshot, date string) (Overview, error) {
	if err := ValidateDate(date); err != nil {
		return Overview{}, err
	}
	out := Overview{Date: date, Portfolios: []Card{}, Points: []Point{}}
	first := ""
	rates := map[string][]Rate{}
	for _, r := range s.Rates {
		rates[r.Currency] = append(rates[r.Currency], r)
	}
	for _, list := range rates {
		sort.Slice(list, func(i, j int) bool { return list[i].Date < list[j].Date })
	}
	prepared := []preparedPortfolio{}
	for _, p := range s.Portfolios {
		p.Days = append([]Day(nil), p.Days...)
		sort.Slice(p.Days, func(i, j int) bool { return p.Days[i].Date < p.Days[j].Date })
		pp := preparedPortfolio{portfolio: p}
		for _, d := range p.Days {
			a, err := dayAmounts(p, d)
			if err != nil {
				return out, err
			}
			pp.days = append(pp.days, a)
		}
		prepared = append(prepared, pp)
		for _, d := range p.Days {
			if d.Date <= date && (first == "" || d.Date < first) {
				first = d.Date
			}
		}
	}
	total, cards, err := at(prepared, rates, date, true)
	if err != nil {
		return out, err
	}
	out.Total = total.summary()
	out.Portfolios = cards
	if first != "" {
		if err = ValidateDate(first); err != nil {
			return out, err
		}
		start, _ := time.Parse("2006-01-02", first)
		end, _ := time.Parse("2006-01-02", date)
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			day := d.Format("2006-01-02")
			a, _, err := at(prepared, rates, day, false)
			if err != nil {
				return out, err
			}
			point := Point{Date: day, Complete: len(a.issues) == 0, Issues: a.issues}
			if point.Complete {
				point.Value = Format(a.total, 18)
			}
			out.Points = append(out.Points, point)
		}
	}
	return out, nil
}
