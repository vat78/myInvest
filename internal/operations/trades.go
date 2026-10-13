package operations

import (
	"errors"
	"math/big"
	"sort"
	"strings"
	"time"
	"unicode"
)

type Asset struct{ ID, Ticker, TypeCode string }

func Ticker(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "", &FieldError{"ticker", "Введите тикер"}
	}
	for _, c := range value {
		if unicode.IsSpace(c) || unicode.IsControl(c) {
			return "", &FieldError{"ticker", "Тикер не должен содержать пробелы и управляющие символы"}
		}
	}
	return value, nil
}
func IsTrade(kind string) bool { return kind == "buy" || kind == "sell" }
func ValidateTrade(in Input) (Input, error) {
	if !IsTrade(in.Kind) {
		return in, &FieldError{"kind", "Выберите покупку или продажу"}
	}
	d, err := time.Parse("2006-01-02", in.Date)
	if err != nil || d.Year() < 1 || d.Format("2006-01-02") != in.Date {
		return in, &FieldError{"date", "Введите существующую календарную дату YYYY-MM-DD"}
	}
	if in.AssetID == "" {
		return in, &FieldError{"assetID", "Выберите актив"}
	}
	in.Quantity, err = Amount(in.Quantity)
	if err != nil {
		return in, &FieldError{"quantity", "Введите количество больше нуля: цифры и точка или запятая"}
	}
	in.Amount, err = Amount(in.Amount)
	if err != nil {
		return in, &FieldError{"amount", err.Error()}
	}
	if in.Expenses == "" {
		in.Expenses = "0"
	}
	in.Expenses = strings.ReplaceAll(strings.TrimSpace(in.Expenses), ",", ".")
	if !decimal.MatchString(in.Expenses) {
		return in, &FieldError{"expenses", "Введите неотрицательные расходы"}
	}
	in.Expenses = exact(rational(in.Expenses))
	// Currency and cash-flow fields are never accepted from a trade client.
	in.CurrencyCode = ""
	in.PortfolioAmount = ""
	return in, nil
}
func AveragePrice(amount, quantity string) (string, bool, error) {
	a, err := Amount(amount)
	if err != nil {
		return "", false, err
	}
	q, err := Amount(quantity)
	if err != nil {
		return "", false, err
	}
	ratio := new(big.Rat).Quo(rational(a), rational(q))
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	n := new(big.Int).Mul(ratio.Num(), scale)
	rounded, rem := new(big.Int), new(big.Int)
	rounded.QuoRem(n, ratio.Denom(), rem)
	changed := rem.Sign() != 0
	if new(big.Int).Lsh(rem, 1).Cmp(ratio.Denom()) >= 0 {
		rounded.Add(rounded, big.NewInt(1))
	}
	return exact(new(big.Rat).SetFrac(rounded, scale)), changed, nil
}
func MoneyChange(op Operation) string {
	if IsTrade(op.Kind) {
		v := rational(op.Amount)
		e := rational(op.Expenses)
		if op.Kind == "buy" {
			v.Add(v, e)
			v.Neg(v)
		} else {
			v.Sub(v, e)
		}
		return exact(v)
	}
	v := rational(op.PortfolioAmount)
	if op.Kind == "withdrawal" {
		v.Neg(v)
	}
	return exact(v)
}

type Day struct {
	Date, Balance string
	Quantities    map[string]string
}
type Warning struct{ OperationID, Date, Ticker, Available, Quantity string }
type CashWarning struct{ Date, Balance string }

type History struct {
	Balance      string
	Days         []Day
	Warnings     []Warning
	CashWarnings []CashWarning
}

// Rebuild consumes a single portfolio, using calendar date then numeric ID.
// Every end-of-day snapshot includes all assets encountered so far, even zeroes.
func Rebuild(operations []Operation) (History, error) {
	ops := append([]Operation(nil), operations...)
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Date != ops[j].Date {
			return ops[i].Date < ops[j].Date
		}
		a, ok := new(big.Int).SetString(ops[i].ID, 10)
		b, ok2 := new(big.Int).SetString(ops[j].ID, 10)
		return ok && ok2 && a.Cmp(b) < 0
	})
	out := History{Balance: "0", Days: []Day{}, Warnings: []Warning{}}
	cash := new(big.Rat)
	positions := map[string]*big.Rat{}
	for i, op := range ops {
		if IsTrade(op.Kind) {
			normalized, err := ValidateTrade(op.Input)
			if err != nil {
				return out, err
			}
			op.Input = normalized
			v := positions[op.AssetID]
			if v == nil {
				v = new(big.Rat)
				positions[op.AssetID] = v
			}
			q := rational(op.Quantity)
			if op.Kind == "sell" {
				if v.Cmp(q) < 0 {
					out.Warnings = append(out.Warnings, Warning{op.ID, op.Date, op.Ticker, exact(v), op.Quantity})
				}
				v.Sub(v, q)
			} else {
				v.Add(v, q)
			}
		} else {
			if op.Kind != "deposit" && op.Kind != "withdrawal" {
				return out, errors.New("invalid stored operation kind")
			}
			value, err := Amount(op.PortfolioAmount)
			if err != nil {
				return out, err
			}
			op.PortfolioAmount = value
		}
		cash.Add(cash, rational(MoneyChange(op)))
		if i+1 == len(ops) || ops[i+1].Date != op.Date {
			day := Day{Date: op.Date, Balance: exact(cash), Quantities: map[string]string{}}
			if cash.Sign() < 0 {
				out.CashWarnings = append(out.CashWarnings, CashWarning{op.Date, day.Balance})
			}
			for id, v := range positions {
				day.Quantities[id] = exact(v)
			}
			out.Days = append(out.Days, day)
		}
	}
	out.Balance = exact(cash)
	return out, nil
}
