package valuation

import (
	"errors"
	"fmt"
	"math/big"
	"myinvest/internal/operations"
	"regexp"
	"strconv"
	"strings"
)

type Target struct {
	AssetID string `json:"assetID"`
	Percent string `json:"percent"`
}
type AllocationPlan struct {
	Generation  string   `json:"generation"`
	Revision    string   `json:"revision"`
	CashPercent string   `json:"cashPercent"`
	Targets     []Target `json:"targets"`
}
type AllocationFieldError struct{ Field, Message string }

func (e *AllocationFieldError) Error() string { return e.Message }

var ErrAllocationConflict = errors.New("План уже изменён. Загрузите актуальный план перед повторным сохранением")
var percentPattern = regexp.MustCompile(`^[0-9]+([.][0-9]{1,2})?$`)

func PercentUnits(s string) (int64, error) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	if !percentPattern.MatchString(s) {
		return 0, errors.New("Введите процент от 0 до 100, не более двух знаков после запятой")
	}
	x, ok := new(big.Rat).SetString(s)
	if !ok || x.Sign() < 0 || x.Cmp(big.NewRat(100, 1)) > 0 {
		return 0, errors.New("Доля должна быть от 0 до 100%")
	}
	x.Mul(x, big.NewRat(100, 1))
	return x.Num().Int64(), nil
}
func PercentString(n int64) string { return Format(big.NewRat(n, 100), 2) }

type AllocationPreview struct {
	Total      string            `json:"total"`
	Difference string            `json:"difference"`
	Errors     map[string]string `json:"errors"`
}

func PreviewAllocation(p AllocationPlan) AllocationPreview {
	out := AllocationPreview{Errors: map[string]string{}}
	total, err := PercentUnits(p.CashPercent)
	if err != nil {
		out.Errors["cashPercent"] = err.Error()
	}
	seen := map[string]bool{}
	for i, t := range p.Targets {
		key := fmt.Sprintf("targets.%d.", i)
		if ValidatePortfolioID(t.AssetID) != nil {
			out.Errors[key+"assetID"] = "Выберите актив"
		} else if seen[t.AssetID] {
			out.Errors[key+"assetID"] = "Актив уже добавлен"
		}
		seen[t.AssetID] = true
		n, e := PercentUnits(t.Percent)
		if e != nil {
			out.Errors[key+"percent"] = e.Error()
		} else {
			total += n
		}
	}
	if len(out.Errors) == 0 {
		out.Total = PercentString(total)
		out.Difference = PercentString(10000 - total)
	}
	return out
}
func ValidateAllocation(p AllocationPlan) (AllocationPlan, error) {
	preview := PreviewAllocation(p)
	if message := preview.Errors["cashPercent"]; message != "" {
		return p, &AllocationFieldError{"cashPercent", message}
	}
	for i := range p.Targets {
		for _, field := range []string{"assetID", "percent"} {
			key := fmt.Sprintf("targets.%d.%s", i, field)
			if message := preview.Errors[key]; message != "" {
				return p, &AllocationFieldError{key, message}
			}
		}
	}
	if preview.Total != "100" {
		return p, &AllocationFieldError{"total", "Сумма долей активов и денег должна быть ровно 100%"}
	}
	if (p.Generation == "") != (p.Revision == "") {
		return p, ErrAllocationConflict
	}
	if p.Revision != "" {
		n, e := strconv.ParseInt(p.Revision, 10, 64)
		if e != nil || n <= 0 || strconv.FormatInt(n, 10) != p.Revision {
			return p, ErrAllocationConflict
		}
	}
	n, _ := PercentUnits(p.CashPercent)
	p.CashPercent = PercentString(n)
	targets := make([]Target, 0, len(p.Targets))
	for _, t := range p.Targets {
		n, _ = PercentUnits(t.Percent)
		targets = append(targets, Target{t.AssetID, PercentString(n)})
	}
	p.Targets = targets
	return p, nil
}

type AllocationSnapshot struct {
	Analysis AnalysisSnapshot
	Assets   []operations.Asset
	Plan     AllocationPlan
}
type AllocationRow struct {
	ID                string `json:"id"`
	Ticker            string `json:"ticker"`
	Cash              bool   `json:"cash"`
	Quantity          string `json:"quantity"`
	CurrentValue      string `json:"currentValue"`
	Percent           string `json:"percent"`
	TargetValue       string `json:"targetValue"`
	TargetQuantity    string `json:"targetQuantity"`
	BuyQuantity       string `json:"buyQuantity"`
	BuyValue          string `json:"buyValue"`
	RebalanceQuantity string `json:"rebalanceQuantity"`
	RebalanceValue    string `json:"rebalanceValue"`
	PriceSource       string `json:"priceSource"`
	PriceDate         string `json:"priceDate"`
	PriceCurrency     string `json:"priceCurrency"`
	PriceRate         string `json:"priceRate"`
	PriceRateDate     string `json:"priceRateDate"`
	Issue             string `json:"issue"`
}
type Allocation struct {
	PortfolioID      string             `json:"portfolioID"`
	Date             string             `json:"date"`
	Currency         string             `json:"currency"`
	Plan             AllocationPlan     `json:"plan"`
	Assets           []operations.Asset `json:"assets"`
	Rows             []AllocationRow    `json:"rows"`
	Total            string             `json:"total"`
	Cash             string             `json:"cash"`
	Reserve          string             `json:"reserve"`
	CashAfterBuys    string             `json:"cashAfterBuys"`
	ReserveShortfall string             `json:"reserveShortfall"`
	Complete         bool               `json:"complete"`
	CanCalculate     bool               `json:"canCalculate"`
	Issues           []string           `json:"issues"`
}

// roundedLabel distinguishes tiny nonzero values and repeating theoretical quantities.
func roundedLabel(x *big.Rat, places int) string {
	s := Format(x, places)
	y, _ := new(big.Rat).SetString(s)
	if x.Cmp(y) != 0 {
		if y.Sign() == 0 {
			if x.Sign() < 0 {
				return "≈−0"
			}
			return "≈+0"
		}
		return "≈" + s
	}
	return s
}
func BuildAllocation(s AllocationSnapshot, pid, date string) (Allocation, error) {
	out := Allocation{PortfolioID: pid, Date: date, Currency: s.Analysis.Currency, Plan: s.Plan, Assets: s.Assets, Rows: []AllocationRow{}, Issues: []string{}, Complete: true}
	if err := ValidatePortfolioID(pid); err != nil {
		return out, err
	}
	if err := ValidateDate(date); err != nil {
		return out, err
	}
	if s.Plan.Generation == "" {
		return out, nil
	}
	plan, err := ValidateAllocation(s.Plan)
	if err != nil {
		return out, err
	}
	out.Plan = plan
	positions, prices, err := prepareAnalysis(s.Analysis, pid, date, true)
	if err != nil {
		return out, err
	}
	// Validate cash operations before passing them to exact ledger arithmetic.
	ops := []operations.Operation{}
	for _, op := range s.Analysis.Operations {
		if op.Date > date {
			continue
		}
		if op.Kind == "deposit" || op.Kind == "withdrawal" {
			in, e := operations.Validate(op.Input, s.Analysis.Currency)
			if e != nil {
				return out, e
			}
			op.Input = in
		} else if operations.IsTrade(op.Kind) {
			in, e := operations.ValidateTrade(op.Input)
			if e != nil {
				return out, e
			}
			op.Input = in
		}
		ops = append(ops, op)
	}
	cash, _ := new(big.Rat).SetString(operations.Balance(ops))
	out.Cash = roundedLabel(cash, 2)
	negative := cash.Sign() < 0
	targets := map[string]int64{}
	for _, t := range plan.Targets {
		n, _ := PercentUnits(t.Percent)
		targets[t.AssetID] = n
	}
	quantities := []*big.Rat{}
	values := []*big.Rat{}
	rowPrices := []*big.Rat{}
	total := new(big.Rat).Set(cash)
	for _, a := range s.Assets {
		p := positions[a.ID]
		q := new(big.Rat)
		var candidate *big.Rat
		day := ""
		if p != nil {
			q.Set(p.quantity)
			candidate = p.price
			day = p.priceDate
		}
		weight, inPlan := targets[a.ID]
		if q.Sign() == 0 && !inPlan {
			continue
		}
		row := AllocationRow{ID: a.ID, Ticker: a.Ticker, Quantity: operations.ExactDecimal(q), Percent: PercentString(weight)}
		if q.Sign() < 0 {
			negative = true
		}
		chosen, e := prices.choose(pid, a.ID, s.Analysis.Currency, date, day, candidate)
		if e != nil {
			return out, e
		}
		row.PriceSource = chosen.Source
		row.PriceDate = chosen.Date
		row.PriceCurrency = chosen.Currency
		row.PriceRate = chosen.Rate
		row.PriceRateDate = chosen.RateDate
		var value *big.Rat
		if q.Sign() == 0 {
			value = new(big.Rat)
		} else if chosen.value != nil {
			value = new(big.Rat).Mul(q, chosen.value)
		}
		if value == nil {
			out.Complete = false
		} else {
			row.CurrentValue = roundedLabel(value, 2)
			total.Add(total, value)
		}
		if chosen.value == nil || chosen.value.Sign() <= 0 {
			row.Issue = chosen.issue
			if row.Issue == "" {
				row.Issue = "Нет положительной цены: добавьте котировку"
			}
		}
		out.Rows = append(out.Rows, row)
		quantities = append(quantities, q)
		values = append(values, value)
		rowPrices = append(rowPrices, chosen.value)
	}
	if !out.Complete {
		out.Issues = append(out.Issues, "Оценка текущих позиций неполная. Проверьте цены и прямые валютные курсы")
	}
	if negative {
		out.Issues = append(out.Issues, "Отрицательные остатки. Проверьте полноту истории; сценарии недоступны")
	}
	reserve := new(big.Rat)
	deficits := make([]*big.Rat, len(out.Rows))
	goals := make([]*big.Rat, len(out.Rows))
	d := new(big.Rat)
	out.CanCalculate = out.Complete && !negative
	if out.Complete {
		n, _ := PercentUnits(plan.CashPercent)
		reserve.Mul(total, big.NewRat(n, 10000))
		out.Total = roundedLabel(total, 2)
		out.Reserve = roundedLabel(reserve, 2)
		for i := range out.Rows {
			row := &out.Rows[i]
			goal := new(big.Rat).Mul(total, big.NewRat(targets[row.ID], 10000))
			goals[i] = goal
			row.TargetValue = roundedLabel(goal, 2)
			price := rowPrices[i]
			if goal.Sign() == 0 && quantities[i].Sign() == 0 {
				row.TargetQuantity = "0"
				row.Issue = ""
			} else if price == nil || price.Sign() <= 0 {
				out.CanCalculate = false
			} else {
				row.TargetQuantity = roundedLabel(new(big.Rat).Quo(goal, price), 18)
			}
			deficits[i] = new(big.Rat).Sub(goal, values[i])
			if deficits[i].Sign() < 0 {
				deficits[i].SetInt64(0)
			}
			d.Add(d, deficits[i])
		}
	}
	cashRow := AllocationRow{ID: "cash", Ticker: "Деньги", Cash: true, CurrentValue: out.Cash, Percent: plan.CashPercent, TargetValue: out.Reserve}
	if out.CanCalculate {
		budget := new(big.Rat).Sub(cash, reserve)
		if budget.Sign() < 0 {
			budget.SetInt64(0)
		}
		if budget.Cmp(d) > 0 {
			budget.Set(d)
		}
		for i := range out.Rows {
			row := &out.Rows[i]
			buy := new(big.Rat)
			if d.Sign() > 0 {
				buy.Quo(new(big.Rat).Mul(budget, deficits[i]), d)
			}
			delta := new(big.Rat).Sub(goals[i], values[i])
			row.BuyValue = roundedLabel(buy, 2)
			row.RebalanceValue = roundedLabel(delta, 2)
			if buy.Sign() == 0 {
				row.BuyQuantity = "0"
			} else {
				row.BuyQuantity = roundedLabel(new(big.Rat).Quo(buy, rowPrices[i]), 18)
			}
			if delta.Sign() == 0 {
				row.RebalanceQuantity = "0"
			} else {
				row.RebalanceQuantity = roundedLabel(new(big.Rat).Quo(delta, rowPrices[i]), 18)
			}
		}
		out.CashAfterBuys = roundedLabel(new(big.Rat).Sub(cash, budget), 2)
		shortfall := new(big.Rat).Sub(reserve, cash)
		if shortfall.Sign() < 0 {
			shortfall.SetInt64(0)
		}
		out.ReserveShortfall = roundedLabel(shortfall, 2)
		cashRow.BuyValue = roundedLabel(new(big.Rat).Neg(budget), 2)
		cashRow.RebalanceValue = roundedLabel(new(big.Rat).Sub(reserve, cash), 2)
	} else if out.Complete && !negative {
		out.Issues = append(out.Issues, "Сценарии недоступны: не всем целевым активам задана положительная цена и доступен прямой курс")
	}
	out.Rows = append(out.Rows, cashRow)
	return out, nil
}
