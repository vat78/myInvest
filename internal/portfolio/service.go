package portfolio

import "context"

type Repository interface {
	List(context.Context) ([]Portfolio, error)
	Find(context.Context, string) (Portfolio, error)
	CurrencyExists(context.Context, string) (bool, error)
	NameExists(context.Context, string, string) (bool, error)
	Create(context.Context, Portfolio) (Portfolio, error)
	Update(context.Context, Portfolio) (Portfolio, error)
}
type Service struct{ repo Repository }

func NewService(repo Repository) *Service                        { return &Service{repo} }
func (s *Service) List(ctx context.Context) ([]Portfolio, error) { return s.repo.List(ctx) }
func (s *Service) validate(ctx context.Context, id string, input Input) (Portfolio, error) {
	p, err := Validate(input)
	if err != nil {
		return Portfolio{}, err
	}
	p.ID = id
	exists, err := s.repo.CurrencyExists(ctx, p.CurrencyCode)
	if err != nil {
		return Portfolio{}, err
	}
	if !exists {
		return Portfolio{}, &FieldError{"currencyCode", "Выберите валюту из справочника"}
	}
	exists, err = s.repo.NameExists(ctx, p.NameKey, id)
	if err != nil {
		return Portfolio{}, err
	}
	if exists {
		return Portfolio{}, DuplicateName()
	}
	return p, nil
}
func (s *Service) Create(ctx context.Context, input Input) (Portfolio, error) {
	p, err := s.validate(ctx, "", input)
	if err != nil {
		return Portfolio{}, err
	}
	return s.repo.Create(ctx, p)
}
func (s *Service) Update(ctx context.Context, id string, input Input) (Portfolio, error) {
	old, err := s.repo.Find(ctx, id)
	if err != nil {
		return Portfolio{}, err
	}
	if old.CurrencyCode != input.CurrencyCode {
		return Portfolio{}, CurrencyChanged()
	}
	p, err := s.validate(ctx, id, input)
	if err != nil {
		return Portfolio{}, err
	}
	return s.repo.Update(ctx, p)
}
