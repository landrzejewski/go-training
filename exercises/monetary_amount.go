package exercises

import "fmt"

// Struktura monetaryAmount, która opisuje wartosci walutowe (zawiera kwotę i walutę)
// Struktura powinna umożliwiać dodawanie i odejmowanie innych wartości walutowych (zaimplementuj metody add, subtract),
// jeżeli waluta jest inna to zwracamy err
// Dodaj funkcję konstruktora

type monetaryAmount struct {
	value    float64
	currency string
}

var CurrencyMismatch = fmt.Errorf("currnency mismatch")

func newMonetaryAmount(value float64, currency string) *monetaryAmount {
	return &monetaryAmount{value, currency}
}

func (ma *monetaryAmount) add(monetaryAmount *monetaryAmount) error {
	if ma.currency != monetaryAmount.currency {
		return CurrencyMismatch
	}
	ma.value += monetaryAmount.value
	return nil
}

func (ma *monetaryAmount) subtract(monetaryAmount *monetaryAmount) error {
	if ma.currency != monetaryAmount.currency {
		return CurrencyMismatch
	}
	ma.value -= monetaryAmount.value
	return nil
}

/*func (ma monetaryAmount) addImmutable(amount *monetaryAmount) (*monetaryAmount, error) {
	if ma.currency != amount.currency {
		return nil, errors.New("incompatible currency")
	}
	ma.value += amount.value
	return &ma, nil
}*/

func MonetaryAmountExercise() {
	amount := newMonetaryAmount(100.0, "PLN")
	otherAmount := newMonetaryAmount(100.0, "PLN")
	amount.add(otherAmount)
	fmt.Println(amount)
}
