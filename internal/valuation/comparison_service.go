package valuation

import (
	"context"
	"myinvest/internal/benchmarks"
)

// ComparisonSnapshot is a consistent read of valuations, external flows and indices.
// Flow amounts use portfolio currency; conversion is performed at the flow date.
type ComparisonSnapshot struct {
	Snapshot
	Benchmarks []ComparisonBenchmark
	Flows      []ComparisonFlow
}
type ComparisonBenchmark struct {
	Benchmark benchmarks.Benchmark
	Points    []benchmarks.Point
}
type ComparisonFlow struct{ PortfolioID, Date, Kind, Amount string }

type ComparisonRepository interface {
	LoadComparison(context.Context, string) (ComparisonSnapshot, error)
}

type ComparisonService struct{ repo ComparisonRepository }

func NewComparisonService(r ComparisonRepository) *ComparisonService {
	return &ComparisonService{repo: r}
}
func (s *ComparisonService) Compare(ctx context.Context, inflationID, date string) (ReturnComparison, error) {
	if err := ValidateDate(date); err != nil {
		return ReturnComparison{}, err
	}
	if err := ValidatePortfolioID(inflationID); err != nil {
		return ReturnComparison{}, err
	}
	snapshot, err := s.repo.LoadComparison(ctx, date)
	if err != nil {
		return ReturnComparison{}, err
	}
	return BuildReturnComparison(snapshot, inflationID, date)
}
