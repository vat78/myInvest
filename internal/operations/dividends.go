package operations

import "time"

// ValidateDividend accepts the net cash credited in the portfolio currency.
// Asset attribution is optional and does not imply a position or change quantity.
func ValidateDividend(in Input) (Input, error) {
	if in.Kind != "dividend" {
		return in, &FieldError{"kind", "Выберите дивиденды"}
	}
	d, err := time.Parse("2006-01-02", in.Date)
	if err != nil || d.Year() < 1 || d.Format("2006-01-02") != in.Date {
		return in, &FieldError{"date", "Введите существующую календарную дату YYYY-MM-DD"}
	}
	amount, err := Amount(in.Amount)
	if err != nil {
		return in, &FieldError{"amount", err.Error()}
	}
	return Input{Kind: "dividend", Date: in.Date, AssetID: in.AssetID, Amount: amount, PortfolioAmount: amount, Comment: in.Comment}, nil
}
