package desktop

import (
	"context"
	"errors"
	"fmt"
	"myinvest/internal/benchmarks"
)

type BenchmarkResult struct {
	Benchmark benchmarks.Benchmark `json:"benchmark"`
	Errors    map[string]string    `json:"errors"`
	Message   string               `json:"message"`
}
type BenchmarkPointResult struct {
	Errors  map[string]string `json:"errors"`
	Message string            `json:"message"`
}
type BenchmarksAPI struct {
	useService func(func(*benchmarks.Service) error) error
}

func NewAPI(fn func(func(*benchmarks.Service) error) error) *BenchmarksAPI { return &BenchmarksAPI{fn} }
func (a *BenchmarksAPI) ListBenchmarks() ([]benchmarks.Benchmark, error) {
	out := []benchmarks.Benchmark{}
	e := a.useService(func(s *benchmarks.Service) error { var e error; out, e = s.List(context.Background()); return e })
	if e != nil {
		return nil, fmt.Errorf("Не удалось загрузить бенчмарки. Повторите попытку")
	}
	return out, nil
}
func (a *BenchmarksAPI) BenchmarkHistory(id string) ([]benchmarks.Point, error) {
	out := []benchmarks.Point{}
	e := a.useService(func(s *benchmarks.Service) error { var e error; out, e = s.History(context.Background(), id); return e })
	if e != nil {
		return nil, fmt.Errorf("Не удалось загрузить историю бенчмарка. Повторите попытку")
	}
	return out, nil
}
func result(e error) (map[string]string, string) {
	out := map[string]string{}
	var f *benchmarks.FieldError
	if errors.As(e, &f) {
		out[f.Field] = f.Message
	} else if e != nil {
		return out, "Не удалось сохранить данные. Повторите попытку"
	}
	return out, ""
}
func (a *BenchmarksAPI) CreateBenchmark(b benchmarks.Benchmark) BenchmarkResult {
	out := BenchmarkResult{}
	e := a.useService(func(s *benchmarks.Service) error {
		var e error
		out.Benchmark, e = s.Create(context.Background(), b)
		return e
	})
	out.Errors, out.Message = result(e)
	return out
}
func (a *BenchmarksAPI) SaveBenchmarkPoint(p benchmarks.Point) BenchmarkPointResult {
	e := a.useService(func(s *benchmarks.Service) error { return s.Save(context.Background(), p) })
	fields, message := result(e)
	return BenchmarkPointResult{fields, message}
}

type BenchmarkPreviewResult struct {
	Point   benchmarks.Point  `json:"point"`
	Errors  map[string]string `json:"errors"`
	Message string            `json:"message"`
}

func (a *BenchmarksAPI) PreviewBenchmarkPoint(p benchmarks.Point) BenchmarkPreviewResult {
	out := BenchmarkPreviewResult{}
	e := a.useService(func(s *benchmarks.Service) error {
		var e error
		out.Point, e = s.Preview(context.Background(), p)
		return e
	})
	out.Errors, out.Message = result(e)
	return out
}

func (a *BenchmarksAPI) SaveBenchmarkSources(id, sources string) BenchmarkPointResult {
	err := a.useService(func(s *benchmarks.Service) error { return s.UpdateSources(context.Background(), id, sources) })
	fields, message := result(err)
	return BenchmarkPointResult{fields, message}
}
