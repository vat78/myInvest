package valuation

import "context"

type Repository interface {
	Load(context.Context, string) (Snapshot, error)
	LoadAnalysis(context.Context, string, string) (AnalysisSnapshot, error)
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

func (s *Service) Analysis(ctx context.Context, pid, date string) (Analysis, error) {
	if err := ValidatePortfolioID(pid); err != nil {
		return Analysis{}, err
	}
	if err := ValidateDate(date); err != nil {
		return Analysis{}, err
	}
	snap, err := s.repo.LoadAnalysis(ctx, pid, date)
	if err != nil {
		return Analysis{}, err
	}
	return BuildAnalysis(snap, pid, date)
}
