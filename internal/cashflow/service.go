package cashflow

import "context"

type Repository interface {
	Transaction(context.Context, func(Repository) error) error
	PortfolioCurrency(context.Context, string) (string, error)
	CurrencyExists(context.Context, string) (bool, error)
	List(context.Context, string) ([]Operation, error)
	Find(context.Context, string, string) (Operation, error)
	Save(context.Context, Operation) (Operation, error)
	Delete(context.Context, string, string) error
}
type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo} }

type Snapshot struct {
	CurrencyCode, Balance string
	Operations            []Operation
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
		out.Balance = Balance(out.Operations)
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
			if _, err = r.Find(ctx, pid, id); err != nil {
				return err
			}
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
