// Package currency defines the manual currency directory and its use cases.
package currency

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

var codePattern = regexp.MustCompile(`^[A-Za-z]{3}$`)
var decimalPattern = regexp.MustCompile(`^[0-9]+([.][0-9]+)?$`)

func Code(input string) (string, error) {
	code := strings.TrimSpace(input)
	if !codePattern.MatchString(code) {
		return "", fmt.Errorf("код валюты должен содержать три латинские буквы")
	}
	return strings.ToUpper(code), nil
}

func Decimal(input string) (string, error) {
	value := strings.ReplaceAll(strings.TrimSpace(input), ",", ".")
	if !decimalPattern.MatchString(value) {
		return "", fmt.Errorf("введите положительный курс: цифры и точка или запятая")
	}
	if strings.Trim(value, "0.") == "" {
		return "", fmt.Errorf("курс должен быть больше нуля")
	}
	parts := strings.SplitN(value, ".", 2)
	parts[0] = strings.TrimLeft(parts[0], "0")
	if parts[0] == "" {
		parts[0] = "0"
	}
	return strings.Join(parts, "."), nil
}

type Pair struct{ Date, Base, Quote string }
type Rate struct {
	Pair
	Value string
}

const inverseScale = 18

// Inverse computes 1/value with exact integer division, then rounds once to
// 18 decimal places, half up. The manually entered rate is never rounded.
func Inverse(input string) (string, error) {
	value, err := Decimal(input)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(value, ".", 2)
	digits, scale := parts[0], 0
	if len(parts) == 2 {
		digits += parts[1]
		scale = len(parts[1])
	}
	denominator, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		return "", fmt.Errorf("не удалось прочитать точный курс")
	}
	numerator := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale+inverseScale)), nil)
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	if remainder.Lsh(remainder, 1).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if quotient.Sign() == 0 {
		return "", fmt.Errorf("обратный курс округляется до нуля при точности 18 знаков; обе пары не сохранены")
	}
	text := quotient.String()
	if len(text) <= inverseScale {
		text = strings.Repeat("0", inverseScale+1-len(text)) + text
	}
	boundary := len(text) - inverseScale
	integer, fraction := text[:boundary], strings.TrimRight(text[boundary:], "0")
	if fraction == "" {
		return integer, nil
	}
	return integer + "." + fraction, nil
}

func NewPair(date, base, quote string) (Pair, error) {
	d, err := time.Parse("2006-01-02", date)
	if err != nil || d.Year() < 1 || d.Format("2006-01-02") != date {
		return Pair{}, fmt.Errorf("дата должна быть существующей календарной датой YYYY-MM-DD")
	}
	b, err := Code(base)
	if err != nil {
		return Pair{}, err
	}
	q, err := Code(quote)
	if err != nil {
		return Pair{}, err
	}
	if b == q {
		return Pair{}, fmt.Errorf("выберите две разные валюты")
	}
	return Pair{Date: date, Base: b, Quote: q}, nil
}
