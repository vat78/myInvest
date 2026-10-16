package benchmarks

import (
	"context"
	"sort"
)

type Repository interface {
	UpdateSources(context.Context, string, string) error
	List(context.Context) ([]Benchmark, error)
	Create(context.Context, Benchmark) (Benchmark, error)
	History(context.Context, string) ([]Point, error)
	Rewrite(context.Context, string, func([]Point) ([]Point, error)) error
}
type Service struct{ repo Repository }

func NewService(r Repository) *Service                           { return &Service{r} }
func (s *Service) List(ctx context.Context) ([]Benchmark, error) { return s.repo.List(ctx) }
func (s *Service) Create(ctx context.Context, b Benchmark) (Benchmark, error) {
	b, e := ValidateBenchmark(b)
	if e != nil {
		return b, e
	}
	return s.repo.Create(ctx, b)
}
func (s *Service) History(ctx context.Context, id string) ([]Point, error) {
	if !validID(id) {
		return nil, &FieldError{"benchmarkID", "Выберите бенчмарк"}
	}
	return s.repo.History(ctx, id)
}
func (s *Service) Save(ctx context.Context, p Point) error {
	p, e := ValidatePoint(p)
	if e != nil {
		return e
	}
	return s.repo.Rewrite(ctx, p.BenchmarkID, func(points []Point) ([]Point, error) {
		replaced := false
		for i := range points {
			if points[i].Date == p.Date {
				points[i] = p
				replaced = true
				break
			}
		}
		if !replaced {
			points = append(points, p)
		}
		sort.Slice(points, func(i, j int) bool { return points[i].Date < points[j].Date })
		return Recalculate(points)
	})
}

func (s *Service) Preview(ctx context.Context, p Point) (Point, error) {
	p, e := ValidatePoint(p)
	if e != nil {
		return Point{}, e
	}
	points, e := s.History(ctx, p.BenchmarkID)
	if e != nil {
		return Point{}, e
	}
	earlier := []Point{}
	for _, row := range points {
		if row.Date < p.Date {
			earlier = append(earlier, row)
		}
	}
	earlier = append(earlier, p)
	result, e := Recalculate(earlier)
	if e != nil {
		return Point{}, e
	}
	return result[len(result)-1], nil
}

func (s *Service) UpdateSources(ctx context.Context, id, sources string) error {
	if !validID(id) {
		return &FieldError{"benchmarkID", "Выберите бенчмарк"}
	}
	sources, err := ValidateSources(sources)
	if err != nil {
		return err
	}
	return s.repo.UpdateSources(ctx, id, sources)
}
