package cashflow

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

type SplitQuantities struct{ Before, After string }
type SplitError struct{ Ticker, Date, OldUnits, NewUnits string }

func (e *SplitError) Error() string {
	date, _ := time.Parse("2006-01-02", e.Date)
	return fmt.Sprintf("Изменение нельзя сохранить: split %s от %s (%s → %s) даёт количество, которое нельзя представить точной конечной десятичной дробью", e.Ticker, date.Format("02.01.2006"), e.OldUnits, e.NewUnits)
}

var integer = regexp.MustCompile(`^[0-9]+$`)

func splitUnits(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	n, ok := new(big.Int).SetString(value, 10)
	if !integer.MatchString(value) || !ok || n.Sign() <= 0 {
		return "", &FieldError{field, "Введите положительное целое число"}
	}
	return n.String(), nil
}
func ValidateSplit(in Input) (Input, error) {
	if in.Kind != "split" {
		return in, &FieldError{"kind", "Выберите сплит"}
	}
	d, err := time.Parse("2006-01-02", in.Date)
	if err != nil || d.Year() < 1 || d.Format("2006-01-02") != in.Date {
		return in, &FieldError{"date", "Введите существующую календарную дату YYYY-MM-DD"}
	}
	if in.AssetID == "" {
		return in, &FieldError{"assetID", "Выберите актив"}
	}
	in.OldUnits, err = splitUnits(in.OldUnits, "oldUnits")
	if err != nil {
		return in, err
	}
	in.NewUnits, err = splitUnits(in.NewUnits, "newUnits")
	if err != nil {
		return in, err
	}
	if in.OldUnits == in.NewUnits {
		return in, &FieldError{"newUnits", "Количество старых и новых акций должно различаться"}
	}
	return Input{Kind: in.Kind, Date: in.Date, AssetID: in.AssetID, OldUnits: in.OldUnits, NewUnits: in.NewUnits, Comment: in.Comment}, nil
}

// SplitQuantity checks the reduced denominator before formatting; no rounding or scale cap.
func SplitQuantity(quantity, oldUnits, newUnits string) (string, error) {
	old, err := splitUnits(oldUnits, "oldUnits")
	if err != nil {
		return "", err
	}
	next, err := splitUnits(newUnits, "newUnits")
	if err != nil {
		return "", err
	}
	if old == next {
		return "", &FieldError{"newUnits", "Количество старых и новых акций должно различаться"}
	}
	q := strings.ReplaceAll(strings.TrimSpace(quantity), ",", ".")
	if !decimal.MatchString(strings.TrimPrefix(q, "-")) {
		return "", errors.New("invalid quantity")
	}
	result := new(big.Rat).Mul(rational(q), rational(next))
	result.Quo(result, rational(old))
	d := new(big.Int).Set(result.Denom())
	for _, factor := range []int64{2, 5} {
		f := big.NewInt(factor)
		for new(big.Int).Mod(d, f).Sign() == 0 {
			d.Quo(d, f)
		}
	}
	if d.Cmp(big.NewInt(1)) != 0 {
		return "", errors.New("non-terminating split quantity")
	}
	return exact(result), nil
}

// PreviewSplit simulates the entire history in the same order and validates downstream splits.
// A new operation uses an ID greater than every existing ID; edits retain their position.
func (s *Service) PreviewSplit(ctx context.Context, pid, id string, in Input) (SplitQuantities, error) {
	var out SplitQuantities
	err := s.repo.Transaction(ctx, func(r Repository) error {
		if _, err := r.PortfolioCurrency(ctx, pid); err != nil {
			return err
		}
		normalized, err := ValidateSplit(in)
		if err != nil {
			return err
		}
		exists, err := r.AssetExists(ctx, normalized.AssetID)
		if err != nil {
			return err
		}
		if !exists {
			return &FieldError{"assetID", "Выберите актив из справочника"}
		}
		ops, err := r.List(ctx, pid)
		if err != nil {
			return err
		}
		candidate := Operation{ID: id, PortfolioID: pid, Input: normalized}
		assets, err := r.ListAssets(ctx)
		if err != nil {
			return err
		}
		for _, asset := range assets {
			if asset.ID == in.AssetID {
				candidate.Ticker = asset.Ticker
			}
		}
		found := false
		maxID := new(big.Int)
		for i, op := range ops {
			n, _ := new(big.Int).SetString(op.ID, 10)
			if n.Cmp(maxID) > 0 {
				maxID.Set(n)
			}
			if id != "" && op.ID == id {
				if op.Kind != "split" {
					return &FieldError{"kind", "Нельзя изменить вид операции"}
				}
				ops[i] = candidate
				found = true
			}
		}
		if id != "" && !found {
			return ErrNotFound
		}
		if id == "" {
			candidate.ID = maxID.Add(maxID, big.NewInt(1)).String()
			ops = append(ops, candidate)
		}
		history, err := Rebuild(ops)
		if err != nil {
			return err
		}
		out = history.Splits[candidate.ID]
		return nil
	})
	return out, err
}

func IsReverseSplit(o Operation) bool {
	if o.Kind != "split" {
		return false
	}
	old, ok := new(big.Int).SetString(o.OldUnits, 10)
	next, ok2 := new(big.Int).SetString(o.NewUnits, 10)
	return ok && ok2 && next.Cmp(old) < 0
}
