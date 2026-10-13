package operations

import (
	"errors"
	"reflect"
	"testing"
)

func TestAssetTypeCatalogAndValidation(t *testing.T) {
	want := []AssetType{{"commodities", "Commodities"}, {"bonds", "Bonds"}, {"money_market_funds", "Money Market Funds"}, {"stocks", "Stocks"}, {"real_estate", "Real Estate"}, {"cryptocurrencies", "Cryptocurrencies"}, {"altcoins", "Altcoins"}}
	if !reflect.DeepEqual(AssetTypes(), want) {
		t.Fatal(AssetTypes())
	}
	for _, v := range want {
		if err := ValidateAssetType(v.Code); err != nil {
			t.Fatal(err)
		}
	}
	for _, code := range []string{"", "unknown", "Stocks", " stocks "} {
		var field *FieldError
		if err := ValidateAssetType(code); !errors.As(err, &field) || field.Field != "typeCode" {
			t.Fatal(code, err)
		}
	}
	catalog := AssetTypes()
	catalog[0].Code = "changed"
	if !reflect.DeepEqual(AssetTypes(), want) {
		t.Fatal("catalog mutated")
	}
}
