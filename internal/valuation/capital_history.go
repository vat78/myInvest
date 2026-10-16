package valuation

import (
	"context"
	"fmt"
	"math/big"
	"myinvest/internal/currency"
	"sort"
	"time"
)

// CapitalPoint is a nominal valuation in the requested currency at end of day.
type CapitalPoint struct {
	Date     string   `json:"date"`
	Value    string   `json:"value"`
	Complete bool     `json:"complete"`
	Issues   []string `json:"issues"`
}

func (s *Service) CapitalHistory(ctx context.Context, code, start, end string) ([]CapitalPoint, error) {
	code, err := currency.Code(code)
	if err != nil {
		return nil, err
	}
	if err = ValidateDate(start); err != nil {
		return nil, err
	}
	if err = ValidateDate(end); err != nil {
		return nil, err
	}
	if start > end {
		return []CapitalPoint{}, nil
	}
	snap, err := s.repo.Load(ctx, end)
	if err != nil {
		return nil, err
	}
	return BuildCapitalHistory(snap, code, start, end)
}

func BuildCapitalHistory(s Snapshot, code, start, end string) ([]CapitalPoint, error) {
	code, err := currency.Code(code)
	if err != nil {
		return nil, err
	}
	for _, date := range []string{start, end} {
		if err = ValidateDate(date); err != nil {
			return nil, err
		}
	}
	out := []CapitalPoint{}
	if start > end {
		return out, nil
	}
	prices, err := preparePrices(s.Pricing)
	if err != nil {
		return nil, err
	}
	portfolios := make([]Portfolio, 0, len(s.Portfolios))
	first := ""
	for _, p := range s.Portfolios {
		p.Days = append([]Day(nil), p.Days...)
		sort.Slice(p.Days, func(i, j int) bool { return p.Days[i].Date < p.Days[j].Date })
		portfolios = append(portfolios, p)
		if len(p.Days) > 0 && (first == "" || p.Days[0].Date < first) {
			first = p.Days[0].Date
		}
	}
	if first == "" || first > end {
		return out, nil
	}
	if first > start {
		start = first
	}
	if err = ValidateDate(start); err != nil {
		return nil, err
	}
	from, _ := time.Parse("2006-01-02", start)
	to, _ := time.Parse("2006-01-02", end)
	changes := valuationDates(s, end)
	var cached CapitalPoint
	cachedChange := -1
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		date := d.Format("2006-01-02")
		change := sort.Search(len(changes), func(i int) bool { return changes[i] > date })
		if change == cachedChange && cached.Complete {
			point := cached
			point.Date = date
			out = append(out, point)
			continue
		}
		point := CapitalPoint{Date: date, Complete: true, Issues: []string{}}
		total := new(big.Rat)
		for _, p := range portfolios {
			i := sort.Search(len(p.Days), func(i int) bool { return p.Days[i].Date > date }) - 1
			if i < 0 {
				continue
			}
			a, err := dayAmounts(p, p.Days[i], prices, date)
			if err != nil {
				return nil, err
			}
			point.Issues = append(point.Issues, a.issues...)
			if a.total.Sign() == 0 || p.Currency == code {
				total.Add(total, a.total)
				continue
			}
			rates := prices.fx[priceKey(p.Currency, code)]
			j := sort.Search(len(rates), func(i int) bool { return rates[i].Date > date }) - 1
			if j < 0 {
				point.Issues = append(point.Issues, fmt.Sprintf("%s: нет курса %s → %s на %s", p.Name, p.Currency, code, date))
				continue
			}
			total.Add(total, new(big.Rat).Mul(a.total, rates[j].value))
		}
		point.Complete = len(point.Issues) == 0
		if point.Complete {
			point.Value = Format(total, 18)
		}
		out = append(out, point)
		cached, cachedChange = point, change
	}
	return out, nil
}
