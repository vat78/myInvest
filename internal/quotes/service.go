package quotes

import "context"

type Repository interface {
	List(context.Context) ([]Quote, error)
	Save(context.Context, Quote) error
}
type Service struct{ repo Repository }

func NewService(repo Repository) *Service                    { return &Service{repo} }
func (s *Service) List(ctx context.Context) ([]Quote, error) { return s.repo.List(ctx) }
func (s *Service) Save(ctx context.Context, q Quote) error {
	q, err := Validate(q)
	if err != nil {
		return err
	}
	return s.repo.Save(ctx, q)
}
