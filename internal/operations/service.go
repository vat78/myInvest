package operations

import "context"

type Repository interface {
	Transaction(context.Context, func(Repository) error) error
	PortfolioCurrency(context.Context, string) (string, error)
	CurrencyExists(context.Context, string) (bool, error)
	List(context.Context, string) ([]Operation, error)
	Find(context.Context, string, string) (Operation, error)
	Save(context.Context, Operation) (Operation, error)
	Delete(context.Context, string, string) error
	ListAssets(context.Context) ([]Asset, error)
	AddAsset(context.Context, string, string) (Asset, error)
	UpdateAssetType(context.Context, string, string) (Asset, error)
	AssetExists(context.Context, string) (bool, error)
}
type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo} }

type Snapshot struct {
	CurrencyCode, Balance string
	Operations            []Operation
	Warnings              []Warning
	CashWarnings          []CashWarning
}

func (s *Service) List(ctx context.Context, pid string) (Snapshot, error) {
	var out Snapshot
	err := s.repo.Transaction(ctx, func(r Repository) error {
		var err error
		out.CurrencyCode, err = r.PortfolioCurrency(ctx, pid)
		if err != nil {
			return err
		}
		out.Operations, err = r.List(ctx, pid)
		if err != nil {
			return err
		}
		history, err := Rebuild(out.Operations)
		if err != nil {
			return err
		}
		for i := range out.Operations {
			if q, ok := history.Splits[out.Operations[i].ID]; ok {
				out.Operations[i].QuantityBefore, out.Operations[i].QuantityAfter = q.Before, q.After
			}
		}
		out.Balance = history.Balance
		out.Warnings = history.Warnings
		out.CashWarnings = history.CashWarnings
		return nil
	})
	return out, err
}
func (s *Service) Save(ctx context.Context, pid, id string, in Input) (Operation, error) {
	var out Operation
	err := s.repo.Transaction(ctx, func(r Repository) error {
		code, err := r.PortfolioCurrency(ctx, pid)
		if err != nil {
			return err
		}
		if id != "" {
			old, findErr := r.Find(ctx, pid, id)
			err = findErr
			if err != nil {
				return err
			}
			if (IsTrade(old.Kind) != IsTrade(in.Kind)) || ((old.Kind == "split") != (in.Kind == "split")) {
				return &FieldError{"kind", "Нельзя изменить вид операции"}
			}
		}
		if in.Kind == "split" {
			in, err = ValidateSplit(in)
			if err != nil {
				return err
			}
			exists, err := r.AssetExists(ctx, in.AssetID)
			if err != nil {
				return err
			}
			if !exists {
				return &FieldError{"assetID", "Выберите актив из справочника"}
			}
			out, err = r.Save(ctx, Operation{ID: id, PortfolioID: pid, Input: in})
			return err
		}
		if IsTrade(in.Kind) {
			in, err = ValidateTrade(in)
			if err != nil {
				return err
			}
			exists, err := r.AssetExists(ctx, in.AssetID)
			if err != nil {
				return err
			}
			if !exists {
				return &FieldError{"assetID", "Выберите актив из справочника"}
			}
			price, rounded, err := AveragePrice(in.Amount, in.Quantity)
			if err != nil {
				return err
			}
			out, err = r.Save(ctx, Operation{ID: id, PortfolioID: pid, Input: in, AveragePrice: price, Rounded: rounded})
			return err
		}
		in, err = Validate(in, code)
		if err != nil {
			return err
		}
		exists, err := r.CurrencyExists(ctx, in.CurrencyCode)
		if err != nil {
			return err
		}
		if !exists {
			return &FieldError{"currencyCode", "Выберите валюту из справочника"}
		}
		out, err = r.Save(ctx, Operation{ID: id, PortfolioID: pid, Input: in})
		return err
	})
	return out, err
}
func (s *Service) Delete(ctx context.Context, pid, id string) error {
	return s.repo.Transaction(ctx, func(r Repository) error {
		if _, err := r.PortfolioCurrency(ctx, pid); err != nil {
			return err
		}
		return r.Delete(ctx, pid, id)
	})
}

func (s *Service) ListAssets(ctx context.Context) ([]Asset, error) { return s.repo.ListAssets(ctx) }
func (s *Service) AddAsset(ctx context.Context, value, typeCode string) (Asset, error) {
	ticker, err := Ticker(value)
	if err != nil {
		return Asset{}, err
	}
	if err := ValidateAssetType(typeCode); err != nil {
		return Asset{}, err
	}
	var out Asset
	err = s.repo.Transaction(ctx, func(r Repository) error {
		var err error
		out, err = r.AddAsset(ctx, ticker, typeCode)
		return err
	})
	return out, err
}

func (s *Service) UpdateAssetType(ctx context.Context, id, typeCode string) (Asset, error) {
	if err := ValidateAssetType(typeCode); err != nil {
		return Asset{}, err
	}
	var out Asset
	err := s.repo.Transaction(ctx, func(r Repository) error {
		var err error
		out, err = r.UpdateAssetType(ctx, id, typeCode)
		return err
	})
	return out, err
}
