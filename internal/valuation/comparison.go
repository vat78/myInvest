package valuation

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
)

type ReturnPoint struct {
	Date       string `json:"date"`
	Annual     string `json:"annual"`
	Cumulative string `json:"cumulative"`
	Issue      string `json:"issue"`
}
type ReturnSeries struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	StartDate string        `json:"startDate"`
	Points    []ReturnPoint `json:"points"`
}
type ReturnComparison struct {
	BaseDate string         `json:"baseDate"`
	Dates    []string       `json:"dates"`
	Series   []ReturnSeries `json:"series"`
}

func percent(v *big.Rat) string {
	return Format(new(big.Rat).Mul(new(big.Rat).Sub(v, big.NewRat(1, 1)), big.NewRat(100, 1)), 18)
}

type returnContext struct {
	prices *priceIndex
	rates  map[string][]Rate
}

func (c returnContext) rub(currency, date string) (*big.Rat, string, error) {
	if currency == "" {
		return big.NewRat(1, 1), "", nil
	}
	_, v, err := rateAt(c.rates, currency, date)
	if err != nil {
		return nil, "", err
	}
	if v == nil {
		return nil, fmt.Sprintf("Нет курса %s → RUB на %s", currency, date), nil
	}
	return v, "", nil
}
func (c returnContext) benchmark(b ComparisonBenchmark, date string) (*big.Rat, string, error) {
	i := sort.Search(len(b.Points), func(i int) bool { return b.Points[i].Date > date }) - 1
	if i < 0 {
		return nil, "Нет истории бенчмарка на " + date, nil
	}
	v, err := decimal(b.Points[i].Value)
	if err != nil {
		return nil, "", err
	}
	if v.Sign() <= 0 {
		return nil, "Неположительная стоимость бенчмарка", nil
	}
	fx, issue, err := c.rub(b.Benchmark.CurrencyCode, date)
	if fx == nil || err != nil {
		return nil, issue, err
	}
	return v.Mul(v, fx), "", nil
}
func (c returnContext) portfolio(p Portfolio, date string) (*big.Rat, string, error) {
	i := sort.Search(len(p.Days), func(i int) bool { return p.Days[i].Date > date }) - 1
	if i < 0 {
		return new(big.Rat), "", nil
	}
	a, err := dayAmounts(p, p.Days[i], c.prices, date)
	if err != nil {
		return nil, "", err
	}
	if len(a.issues) > 0 {
		return nil, a.issues[0], nil
	}
	fx, issue, err := c.rub(p.Currency, date)
	if a.total.Sign() == 0 {
		return a.total, "", nil
	}
	if fx == nil || err != nil {
		return nil, issue, err
	}
	return a.total.Mul(a.total, fx), "", nil
}

func BuildReturnComparison(s ComparisonSnapshot, inflationID, date string) (ReturnComparison, error) {
	out := ReturnComparison{Dates: []string{}, Series: []ReturnSeries{}}
	if err := ValidateDate(date); err != nil {
		return out, err
	}
	prices, err := preparePrices(s.Pricing)
	if err != nil {
		return out, err
	}
	c := returnContext{prices: prices, rates: map[string][]Rate{}}
	for _, r := range s.Rates {
		c.rates[r.Currency] = append(c.rates[r.Currency], r)
	}
	for _, rs := range c.rates {
		sort.Slice(rs, func(i, j int) bool { return rs[i].Date < rs[j].Date })
	}
	// Copy before sorting so the caller's snapshot remains unchanged.
	bs := append([]ComparisonBenchmark(nil), s.Benchmarks...)
	var inflation *ComparisonBenchmark
	for i := range bs {
		bs[i].Points = append(bs[i].Points[:0:0], bs[i].Points...)
		sort.Slice(bs[i].Points, func(a, b int) bool { return bs[i].Points[a].Date < bs[i].Points[b].Date })
		if bs[i].Benchmark.ID == inflationID {
			inflation = &bs[i]
		}
	}
	if inflation == nil || (inflation.Benchmark.CurrencyCode != "" && inflation.Benchmark.CurrencyCode != "RUB") {
		return out, fmt.Errorf("Выберите бенчмарк рублёвой инфляции (индекс или RUB)")
	}
	if len(inflation.Points) == 0 || inflation.Points[0].Date > date {
		return out, fmt.Errorf("Нет истории выбранной инфляции до даты расчёта")
	}
	out.BaseDate = inflation.Points[0].Date
	baseInflation, issue, err := c.benchmark(*inflation, out.BaseDate)
	if err != nil {
		return out, err
	}
	if issue != "" {
		return out, fmt.Errorf("%s", issue)
	}
	firstYear, _ := strconv.Atoi(out.BaseDate[:4])
	lastYear, _ := strconv.Atoi(date[:4])
	for year := firstYear; year <= lastYear; year++ {
		end := fmt.Sprintf("%04d-12-31", year)
		if end > date {
			end = date
		}
		out.Dates = append(out.Dates, end)
	}
	anchor := func(start string) (*big.Rat, string, error) {
		v, msg, e := c.benchmark(*inflation, start)
		if v != nil {
			v.Quo(v, baseInflation)
		}
		return v, msg, e
	}
	for _, b := range bs {
		series := ReturnSeries{ID: "benchmark:" + b.Benchmark.ID, Name: "Бенчмарк: " + b.Benchmark.Name, Points: []ReturnPoint{}}
		if len(b.Points) == 0 || b.Points[0].Date > date {
			out.Series = append(out.Series, series)
			continue
		}
		start := b.Points[0].Date
		if start < out.BaseDate {
			start = out.BaseDate
		}
		series.StartDate = start
		base, msg, e := c.benchmark(b, start)
		if e != nil {
			return out, e
		}
		offset, anchorIssue, e := anchor(start)
		if e != nil {
			return out, e
		}
		if msg == "" {
			msg = anchorIssue
		}
		previous := base
		for _, end := range out.Dates {
			p := ReturnPoint{Date: end}
			if end < start {
				p.Issue = "Ряд ещё не начался"
			} else if msg != "" {
				p.Issue = msg
			} else {
				v, why, e := c.benchmark(b, end)
				if e != nil {
					return out, e
				}
				p.Issue = why
				if v != nil {
					p.Cumulative = percent(new(big.Rat).Mul(offset, new(big.Rat).Quo(v, base)))
					if previous != nil {
						p.Annual = percent(new(big.Rat).Quo(v, previous))
					} else {
						p.Issue = "Нет оценки на начало годового периода"
					}
				}
				previous = v
			}
			series.Points = append(series.Points, p)
		}
		out.Series = append(out.Series, series)
	}
	for _, original := range s.Portfolios {
		p := original
		p.Days = append([]Day(nil), p.Days...)
		sort.Slice(p.Days, func(i, j int) bool { return p.Days[i].Date < p.Days[j].Date })
		series, e := c.portfolioReturns(p, s.Flows, out, anchor)
		if e != nil {
			return out, e
		}
		out.Series = append(out.Series, series)
	}
	return out, nil
}

func (c returnContext) portfolioReturns(p Portfolio, flows []ComparisonFlow, out ReturnComparison, anchor func(string) (*big.Rat, string, error)) (ReturnSeries, error) {
	series := ReturnSeries{ID: "portfolio:" + p.ID, Name: "Портфель: " + p.Name, Points: []ReturnPoint{}}
	if len(p.Days) == 0 {
		return series, nil
	}
	start := p.Days[0].Date
	if start > out.Dates[len(out.Dates)-1] {
		return series, nil
	}
	if start < out.BaseDate {
		start = out.BaseDate
	}
	series.StartDate = start
	offset, issue, err := anchor(start)
	if err != nil {
		return series, err
	}
	net := map[string]*big.Rat{}
	for _, f := range flows {
		if f.PortfolioID != p.ID || f.Date < start || f.Date > out.Dates[len(out.Dates)-1] {
			continue
		}
		amount, e := decimal(f.Amount)
		if e != nil {
			return series, e
		}
		if f.Kind == "withdrawal" {
			amount.Neg(amount)
		} else if f.Kind != "deposit" {
			return series, fmt.Errorf("invalid external flow")
		}
		if net[f.Date] == nil {
			net[f.Date] = new(big.Rat)
		}
		net[f.Date].Add(net[f.Date], amount)
	}
	events := map[string]bool{start: true}
	for d := range net {
		events[d] = true
	}
	for _, d := range out.Dates {
		if d >= start {
			events[d] = true
		}
	}
	dates := []string{}
	for d := range events {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	cumulative, yearBase := big.NewRat(1, 1), big.NewRat(1, 1)
	var previous *big.Rat
	values := map[string]ReturnPoint{}
	for _, d := range dates {
		point := ReturnPoint{Date: d}
		if issue == "" {
			v, why, e := c.portfolio(p, d)
			if e != nil {
				return series, e
			}
			issue = why
			if issue == "" && v.Sign() < 0 {
				issue = "TWR недоступна при отрицательной стоимости"
			}
			if issue == "" {
				flow := new(big.Rat)
				if f := net[d]; f != nil && f.Sign() != 0 {
					fx, why, e := c.rub(p.Currency, d)
					if e != nil {
						return series, e
					}
					issue = why
					if fx != nil {
						flow.Mul(f, fx)
					}
				}
				if issue == "" {
					if previous == nil && start == out.BaseDate && p.Days[0].Date < start {
						previous = new(big.Rat).Set(v)
					} else {
						denominator := previous
						numerator := new(big.Rat).Sub(v, flow)
						if denominator == nil || denominator.Sign() == 0 {
							if flow.Sign() > 0 {
								denominator = flow
								numerator = new(big.Rat).Set(v)
							} else if v.Sign() == 0 && flow.Sign() == 0 && denominator != nil {
								denominator = big.NewRat(1, 1)
								numerator = big.NewRat(1, 1)
							} else {
								issue = "Нет положительного капитала для расчёта TWR"
							}
						}
						if issue == "" {
							if denominator.Sign() <= 0 || v.Sign() < 0 || numerator.Sign() < 0 {
								issue = "TWR недоступна при неположительной базе или отрицательной стоимости"
							} else {
								cumulative.Mul(cumulative, new(big.Rat).Quo(numerator, denominator))
							}
						}
						previous = v
					}
				}
			}
		}
		point.Issue = issue
		if issue == "" {
			point.Cumulative = percent(new(big.Rat).Mul(offset, cumulative))
			if yearBase.Sign() > 0 {
				point.Annual = percent(new(big.Rat).Quo(cumulative, yearBase))
			} else {
				point.Issue = "Нулевая база годовой доходности"
			}
		}
		values[d] = point
		// Only year endpoints reset the annual base; flow dates do not.
		i := sort.SearchStrings(out.Dates, d)
		if i < len(out.Dates) && out.Dates[i] == d {
			yearBase.Set(cumulative)
		}
	}
	for _, d := range out.Dates {
		point, ok := values[d]
		if !ok {
			point = ReturnPoint{Date: d, Issue: "Портфель ещё не сформирован"}
		}
		series.Points = append(series.Points, point)
	}
	return series, nil
}
