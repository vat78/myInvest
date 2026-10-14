package valuation

import "context"

type AllocationRepository interface {
	LoadAllocation(context.Context, string, string) (AllocationSnapshot, error)
	SaveAllocation(context.Context, string, AllocationPlan) error
	DeleteAllocation(context.Context, string, string, string) error
}
type AllocationService struct{ repo AllocationRepository }

func NewAllocationService(r AllocationRepository) *AllocationService { return &AllocationService{r} }
func (s *AllocationService) Load(ctx context.Context, pid, date string) (Allocation, error) {
	if err := ValidatePortfolioID(pid); err != nil {
		return Allocation{}, err
	}
	if err := ValidateDate(date); err != nil {
		return Allocation{}, err
	}
	snap, err := s.repo.LoadAllocation(ctx, pid, date)
	if err != nil {
		return Allocation{}, err
	}
	return BuildAllocation(snap, pid, date)
}
func (s *AllocationService) Save(ctx context.Context, pid string, p AllocationPlan) error {
	if err := ValidatePortfolioID(pid); err != nil {
		return err
	}
	p, err := ValidateAllocation(p)
	if err != nil {
		return err
	}
	return s.repo.SaveAllocation(ctx, pid, p)
}
func (s *AllocationService) Delete(ctx context.Context, pid, generation, revision string) error {
	if err := ValidatePortfolioID(pid); err != nil {
		return err
	}
	if generation == "" || ValidatePortfolioID(revision) != nil {
		return ErrAllocationConflict
	}
	return s.repo.DeleteAllocation(ctx, pid, generation, revision)
}
