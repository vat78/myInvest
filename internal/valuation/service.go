package valuation

import "context"

type Repository interface {
	Load(context.Context, string) (Snapshot, error)
}
type Service struct{ repo Repository }

func NewService(r Repository) *Service { return &Service{r} }
func (s *Service) Overview(ctx context.Context, date string) (Overview, error) {
	if err := ValidateDate(date); err != nil {
		return Overview{}, err
	}
	snap, err := s.repo.Load(ctx, date)
	if err != nil {
		return Overview{}, err
	}
	return Build(snap, date)
}
