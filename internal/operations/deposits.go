package operations

import (
	"math/big"
	"sort"
)

// DepositTotals returns end-of-day cumulative deposits in RUB. Empty values
// mean a deposit has neither an operation nor a portfolio amount in RUB.
// Withdrawals and trades never change this total. Input is not mutated.
func DepositTotals(ops []Operation, portfolioCurrency string) (map[string]string, error) {
	ordered := append([]Operation(nil), ops...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Date < ordered[j].Date })
	totals := map[string]string{}
	total := new(big.Rat)
	complete := true
	for _, op := range ordered {
		if op.Kind == "deposit" {
			value := ""
			if op.CurrencyCode == "RUB" {
				value = op.Amount
			} else if portfolioCurrency == "RUB" {
				value = op.PortfolioAmount
			}
			if value == "" {
				complete = false
			} else {
				normalized, err := Amount(value)
				if err != nil {
					return nil, err
				}
				total.Add(total, rational(normalized))
			}
		}
		if complete {
			totals[op.Date] = exact(total)
		} else {
			totals[op.Date] = ""
		}
	}
	return totals, nil
}

// ExactDecimal formats a finite decimal sum without rounding.
func ExactDecimal(value *big.Rat) string { return exact(value) }
