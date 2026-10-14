package quotes

import "testing"

func TestValidate(t *testing.T) {
	base := Quote{AssetID: "1", Date: "2026-10-14", CurrencyCode: " rub ", Price: "00012,123456789012345678900", Ticker: "forged"}
	q, err := Validate(base)
	if err != nil || q.Price != "12.123456789012345678900" || q.CurrencyCode != "RUB" || q.Ticker != "" {
		t.Fatalf("exact validation: %#v %v", q, err)
	}
	for _, tc := range []struct{ field, value string }{
		{"assetID", ""}, {"assetID", "0"}, {"assetID", "-1"}, {"assetID", "01"}, {"assetID", "9223372036854775808"},
		{"date", "2026-02-29"}, {"date", "0000-01-01"}, {"date", "2026-1-01"},
		{"currencyCode", "EURO"}, {"price", ""}, {"price", "0.00"}, {"price", "-1"}, {"price", "1e2"}, {"price", "1.2.3"}, {"price", ".1"}, {"price", "1."}, {"price", "1 000"},
	} {
		t.Run(tc.field+tc.value, func(t *testing.T) {
			in := base
			switch tc.field {
			case "assetID":
				in.AssetID = tc.value
			case "date":
				in.Date = tc.value
			case "currencyCode":
				in.CurrencyCode = tc.value
			case "price":
				in.Price = tc.value
			}
			_, err := Validate(in)
			f, ok := err.(*FieldError)
			if !ok || f.Field != tc.field {
				t.Fatalf("field error: %v", err)
			}
		})
	}
	for _, price := range []string{"0.1", "0.2", "999999999999999999999999999999.000001", "0.000000000000000000001"} {
		in := base
		in.Price = price
		in.Date = "2099-01-01"
		q, err := Validate(in)
		if err != nil || q.Price != price {
			t.Fatalf("price %s: %v %v", price, q, err)
		}
	}
}
