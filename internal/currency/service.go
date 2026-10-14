package currency

import "context"

type Repository interface {
	ListCurrencies(context.Context) ([]string, error)
	ListRates(context.Context) ([]Rate, error)
	AddCurrency(context.Context, string) error
	FindRate(context.Context, Pair) (string, bool, error)
	// SaveRates must persist both directions atomically.
	SaveRates(context.Context, Rate, Rate) error
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service                        { return &Service{repo: repo} }
func (s *Service) List(ctx context.Context) ([]string, error)    { return s.repo.ListCurrencies(ctx) }
func (s *Service) ListRates(ctx context.Context) ([]Rate, error) { return s.repo.ListRates(ctx) }

func (s *Service) Add(ctx context.Context, input string) (string, error) {
	code, err := Code(input)
	if err != nil {
		return "", err
	}
	return code, s.repo.AddCurrency(ctx, code)
}
func (s *Service) Find(ctx context.Context, date, base, quote string) (string, bool, error) {
	pair, err := NewPair(date, base, quote)
	if err != nil {
		return "", false, err
	}
	return s.repo.FindRate(ctx, pair)
}
func (s *Service) Save(ctx context.Context, date, base, quote, input string) error {
	pair, err := NewPair(date, base, quote)
	if err != nil {
		return err
	}
	value, err := Decimal(input)
	if err != nil {
		return err
	}
	inverse, err := Inverse(value)
	if err != nil {
		return err
	}
	return s.repo.SaveRates(ctx,
		Rate{Pair: pair, Value: value},
		Rate{Pair: Pair{Date: pair.Date, Base: pair.Quote, Quote: pair.Base}, Value: inverse})
}
