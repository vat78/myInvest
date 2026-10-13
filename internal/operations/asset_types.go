package operations

import "errors"

var ErrAssetNotFound = errors.New("Актив не найден")

type AssetType struct{ Code, Name string }

// AssetTypes returns a fresh catalog in the user-facing order.
func AssetTypes() []AssetType {
	return []AssetType{
		{"commodities", "Commodities"},
		{"bonds", "Bonds"},
		{"money_market_funds", "Money Market Funds"},
		{"stocks", "Stocks"},
		{"real_estate", "Real Estate"},
		{"cryptocurrencies", "Cryptocurrencies"},
		{"altcoins", "Altcoins"},
	}
}
func ValidateAssetType(code string) error {
	if code == "" {
		return &FieldError{"typeCode", "Выберите тип актива"}
	}
	for _, v := range AssetTypes() {
		if v.Code == code {
			return nil
		}
	}
	return &FieldError{"typeCode", "Недопустимый тип актива"}
}
