package valuation

import (
	"errors"
	"fmt"
	"math/big"
	"myinvest/internal/quotes"
	"sort"
)

// Pricing is loaded together with quantities and trade prices on one DB snapshot.
type Pricing struct {
	Quotes []quotes.Quote
	FX     []ExchangeRate
	Splits []PriceSplit
}
type ExchangeRate struct{ Date, Base, Quote, Value string }
type PriceSplit struct{ PortfolioID, AssetID, Date, OldUnits, NewUnits string }
type splitFactor struct {
	date   string
	factor *big.Rat
}
type indexedQuote struct {
	quotes.Quote
	value *big.Rat
}
type indexedRate struct {
	ExchangeRate
	value *big.Rat
}
type priceIndex struct {
	quotes map[string][]indexedQuote
	fx     map[string][]indexedRate
	splits map[string][]splitFactor
}
type selectedPrice struct {
	Source, Date, Currency, Rate, RateDate string
	value                                  *big.Rat
	issue                                  string
}

func priceKey(a, b string) string { return a + "\x00" + b }
func preparePrices(p Pricing) (*priceIndex, error) {
	idx := &priceIndex{quotes: map[string][]indexedQuote{}, fx: map[string][]indexedRate{}, splits: map[string][]splitFactor{}}
	for _, q := range p.Quotes {
		v, err := decimal(q.Price)
		if err != nil || v.Sign() <= 0 {
			return nil, errors.New("invalid stored quote")
		}
		if err := ValidateDate(q.Date); err != nil {
			return nil, err
		}
		idx.quotes[q.AssetID] = append(idx.quotes[q.AssetID], indexedQuote{q, v})
	}
	for _, list := range idx.quotes {
		sort.Slice(list, func(i, j int) bool {
			if list[i].Date != list[j].Date {
				return list[i].Date < list[j].Date
			}
			return list[i].CurrencyCode < list[j].CurrencyCode
		})
	}
	for _, f := range p.FX {
		v, err := decimal(f.Value)
		if err != nil || v.Sign() <= 0 {
			return nil, errors.New("invalid stored exchange rate")
		}
		if err := ValidateDate(f.Date); err != nil {
			return nil, err
		}
		k := priceKey(f.Base, f.Quote)
		idx.fx[k] = append(idx.fx[k], indexedRate{f, v})
	}
	for _, list := range idx.fx {
		sort.Slice(list, func(i, j int) bool { return list[i].Date < list[j].Date })
	}
	splits := map[string][]PriceSplit{}
	for _, s := range p.Splits {
		if err := ValidateDate(s.Date); err != nil {
			return nil, err
		}
		k := priceKey(s.PortfolioID, s.AssetID)
		splits[k] = append(splits[k], s)
	}
	for k, list := range splits {
		sort.Slice(list, func(i, j int) bool { return list[i].Date < list[j].Date })
		cumulative := big.NewRat(1, 1)
		for _, s := range list {
			old, e1 := decimal(s.OldUnits)
			next, e2 := decimal(s.NewUnits)
			if e1 != nil || e2 != nil || old.Sign() <= 0 || next.Sign() <= 0 {
				return nil, errors.New("invalid stored split")
			}
			cumulative.Mul(cumulative, new(big.Rat).Quo(old, next))
			idx.splits[k] = append(idx.splits[k], splitFactor{s.Date, new(big.Rat).Set(cumulative)})
		}
	}
	return idx, nil
}
func factorAt(list []splitFactor, date string) *big.Rat {
	i := sort.Search(len(list), func(i int) bool { return list[i].date > date }) - 1
	if i < 0 {
		return big.NewRat(1, 1)
	}
	return list[i].factor
}

// choose compares original source dates; a split does not make an old price newer.
// Quotes on the trade date take precedence and refer to end-of-day units.
func (idx *priceIndex) choose(pid, asset, currency, date, tradeDate string, tradePrice *big.Rat) (selectedPrice, error) {
	out := selectedPrice{Source: "trade", Date: tradeDate, Currency: currency, value: tradePrice}
	list := idx.quotes[asset]
	end := sort.Search(len(list), func(i int) bool { return list[i].Date > date })
	if end == 0 {
		return out, nil
	}
	day := list[end-1].Date
	start := sort.Search(end, func(i int) bool { return list[i].Date >= day })
	chosen := list[start] // Same-day ties prefer portfolio currency, otherwise lexical currency code.
	for _, q := range list[start:end] {
		if q.CurrencyCode == currency {
			chosen = q
			break
		}
	}
	if tradePrice != nil && day < tradeDate {
		return out, nil
	}
	price := new(big.Rat).Set(chosen.value)
	out = selectedPrice{Source: "quote", Date: day, Currency: chosen.CurrencyCode, value: price}
	factors := idx.splits[priceKey(pid, asset)]
	out.value.Mul(out.value, new(big.Rat).Quo(factorAt(factors, date), factorAt(factors, day)))
	if chosen.CurrencyCode != currency {
		rates := idx.fx[priceKey(chosen.CurrencyCode, currency)]
		i := sort.Search(len(rates), func(i int) bool { return rates[i].Date > date }) - 1
		if i < 0 {
			out.value = nil
			out.issue = fmt.Sprintf("нет курса %s → %s на %s для котировки от %s", chosen.CurrencyCode, currency, date, day)
			return out, nil
		}
		rate := rates[i]
		out.value.Mul(out.value, rate.value)
		out.Rate = rate.Value
		out.RateDate = rate.Date
	}
	return out, nil
}
