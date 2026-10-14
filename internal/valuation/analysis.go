package valuation

import (
	"errors"
	"math/big"
	"myinvest/internal/operations"
	"sort"
)

type AnalysisSnapshot struct {
	Pricing    Pricing
	Currency   string
	Operations []operations.Operation
}
type AnalysisAsset struct {
	PriceSource   string `json:"priceSource"`
	PriceDate     string `json:"priceDate"`
	PriceCurrency string `json:"priceCurrency"`
	PriceRate     string `json:"priceRate"`
	PriceRateDate string `json:"priceRateDate"`
	ID            string `json:"id"`
	Ticker        string `json:"ticker"`
	Quantity      string `json:"quantity"`
	CurrentValue  string `json:"currentValue"`
	Percent       string `json:"percent"`
	AveragePrice  string `json:"averagePrice"`
	AverageValue  string `json:"averageValue"`
	ChartValue    string `json:"chartValue"`
}
type Analysis struct {
	PortfolioID string          `json:"portfolioID"`
	Currency    string          `json:"currency"`
	Date        string          `json:"date"`
	Total       string          `json:"total"`
	Complete    bool            `json:"complete"`
	Negative    bool            `json:"negative"`
	Assets      []AnalysisAsset `json:"assets"`
	Issues      []string        `json:"issues"`
}
type purchasePosition struct {
	ticker, priceDate               string
	quantity, bought, amount, price *big.Rat
}

func ValidatePortfolioID(id string) error {
	n, ok := new(big.Int).SetString(id, 10)
	if !ok || n.Sign() <= 0 || n.String() != id {
		return errors.New("Выберите портфель")
	}
	return nil
}

// BuildAnalysis keeps all intermediate values rational, including split-adjusted prices.
func BuildAnalysis(s AnalysisSnapshot, pid, date string) (Analysis, error) {
	out := Analysis{PortfolioID: pid, Currency: s.Currency, Date: date, Complete: true, Assets: []AnalysisAsset{}, Issues: []string{}}
	if err := ValidateDate(date); err != nil {
		return out, err
	}
	if err := ValidatePortfolioID(pid); err != nil {
		return out, err
	}
	positions, prices, err := prepareAnalysis(s, pid, date, false)
	if err != nil {
		return out, err
	}
	ids := make([]string, 0, len(positions))
	for id, p := range positions {
		if p.quantity.Sign() != 0 {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return positions[ids[i]].ticker < positions[ids[j]].ticker })
	values := map[string]*big.Rat{}
	total := new(big.Rat)
	for _, id := range ids {
		p := positions[id]
		row := AnalysisAsset{ID: id, Ticker: p.ticker, Quantity: operations.ExactDecimal(p.quantity)}
		if p.quantity.Sign() < 0 {
			out.Negative = true
		}
		chosen, err := prices.choose(pid, id, s.Currency, date, p.priceDate, p.price)
		if err != nil {
			return out, err
		}
		row.PriceSource = chosen.Source
		row.PriceDate = chosen.Date
		row.PriceCurrency = chosen.Currency
		row.PriceRate = chosen.Rate
		row.PriceRateDate = chosen.RateDate
		if chosen.value == nil {
			out.Complete = false
			issue := chosen.issue
			if issue == "" {
				issue = "нет цены для оценки"
			}
			out.Issues = append(out.Issues, p.ticker+": "+issue)
		} else {
			v := new(big.Rat).Mul(p.quantity, chosen.value)
			values[id] = v
			total.Add(total, v)
			row.CurrentValue = Format(v, 2)
		}
		if p.bought.Sign() != 0 {
			avg := new(big.Rat).Quo(p.amount, p.bought)
			row.AveragePrice = Format(avg, 2)
			row.AverageValue = Format(new(big.Rat).Mul(p.quantity, avg), 2)
		}
		out.Assets = append(out.Assets, row)
	}
	if out.Complete {
		out.Total = Format(total, 2)
	}
	if out.Complete && !out.Negative && total.Sign() > 0 {
		for i := range out.Assets {
			row := &out.Assets[i]
			share := new(big.Rat).Mul(new(big.Rat).Quo(values[row.ID], total), big.NewRat(100, 1))
			row.Percent = Format(share, 2)
			row.ChartValue = Format(share, 18)
			if row.ChartValue == "0" && share.Sign() != 0 {
				row.ChartValue = ""
			}
		}
	}
	return out, nil
}

// prepareAnalysis preserves exact positions and source prices for calculations.
// purchasesOnly keeps sales out of the candidate price for target allocation.
func prepareAnalysis(s AnalysisSnapshot, pid, date string, purchasesOnly bool) (map[string]*purchasePosition, *priceIndex, error) {
	ops := append([]operations.Operation(nil), s.Operations...)
	for _, op := range ops {
		if op.PortfolioID != pid {
			return nil, nil, errors.New("mixed portfolio history")
		}
		if err := ValidateDate(op.Date); err != nil {
			return nil, nil, err
		}
		if err := ValidatePortfolioID(op.ID); err != nil {
			return nil, nil, err
		}
	}
	pricing := s.Pricing
	pricing.Splits = nil
	for _, op := range ops {
		if op.Kind == "split" && op.Date <= date {
			pricing.Splits = append(pricing.Splits, PriceSplit{PortfolioID: pid, AssetID: op.AssetID, Date: op.Date, OldUnits: op.OldUnits, NewUnits: op.NewUnits})
		}
	}
	prices, err := preparePrices(pricing)
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(ops, func(i, j int) bool {
		a, b := ops[i], ops[j]
		if a.Date != b.Date {
			return a.Date < b.Date
		}
		if (a.Kind == "split") != (b.Kind == "split") {
			return a.Kind == "split"
		}
		x, _ := new(big.Int).SetString(a.ID, 10)
		y, _ := new(big.Int).SetString(b.ID, 10)
		return x.Cmp(y) < 0
	})
	positions := map[string]*purchasePosition{}
	for _, op := range ops {
		if op.Date > date {
			continue
		}
		if op.Kind == "deposit" || op.Kind == "withdrawal" {
			continue
		}
		if op.Kind == "split" {
			in, err := operations.ValidateSplit(op.Input)
			if err != nil {
				return nil, nil, err
			}
			op.Input = in
		} else if operations.IsTrade(op.Kind) {
			in, err := operations.ValidateTrade(op.Input)
			if err != nil {
				return nil, nil, err
			}
			op.Input = in
		} else {
			return nil, nil, errors.New("invalid operation kind")
		}
		p := positions[op.AssetID]
		if p == nil {
			p = &purchasePosition{ticker: op.Ticker, quantity: new(big.Rat), bought: new(big.Rat), amount: new(big.Rat)}
			positions[op.AssetID] = p
		}
		if op.Kind == "split" {
			after, err := operations.SplitQuantity(operations.ExactDecimal(p.quantity), op.OldUnits, op.NewUnits)
			if err != nil {
				return nil, nil, err
			}
			p.quantity.SetString(after)
			old, _ := decimal(op.OldUnits)
			next, _ := decimal(op.NewUnits)
			factor := new(big.Rat).Quo(next, old)
			p.bought.Mul(p.bought, factor)
			if p.price != nil {
				p.price.Quo(p.price, factor)
			}
		} else {
			q, _ := decimal(op.Quantity)
			if op.Kind == "sell" {
				p.quantity.Sub(p.quantity, q)
			} else {
				p.quantity.Add(p.quantity, q)
				p.bought.Add(p.bought, q)
				a, _ := decimal(op.Amount)
				p.amount.Add(p.amount, a)
			}
			if !purchasesOnly || op.Kind == "buy" {
				if purchasesOnly {
					// Allocation uses the exact purchase price; round only its output.
					amount, _ := decimal(op.Amount)
					p.price = new(big.Rat).Quo(amount, q)
				} else {
					price, _, err := operations.AveragePrice(op.Amount, op.Quantity)
					if err != nil {
						return nil, nil, err
					}
					p.price, _ = decimal(price)
				}
				p.priceDate = op.Date
			}
		}
	}
	return positions, prices, nil
}
