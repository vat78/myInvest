// Package portfolio defines portfolio metadata and application use cases.
package portfolio

import (
	"errors"
	"strings"
	"time"
)

type Portfolio struct{ ID, Name, NameKey, OpenedOn, CurrencyCode string }
type Input struct{ Name, OpenedOn, CurrencyCode string }
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Message }

var ErrNotFound = errors.New("Портфель не найден")

func DuplicateName() error {
	return &FieldError{"name", "Портфель с таким названием уже существует"}
}
func CurrencyChanged() error {
	return &FieldError{"currencyCode", "Валюту портфеля нельзя изменять"}
}
func Validate(input Input) (Portfolio, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Portfolio{}, &FieldError{"name", "Введите название портфеля"}
	}
	date, err := time.Parse("2006-01-02", input.OpenedOn)
	if err != nil || date.Year() < 1 || date.Format("2006-01-02") != input.OpenedOn {
		return Portfolio{}, &FieldError{"openedOn", "Введите существующую календарную дату YYYY-MM-DD"}
	}
	if input.CurrencyCode == "" {
		return Portfolio{}, &FieldError{"currencyCode", "Выберите валюту"}
	}
	return Portfolio{Name: name, NameKey: strings.ToLower(name), OpenedOn: input.OpenedOn, CurrencyCode: input.CurrencyCode}, nil
}
