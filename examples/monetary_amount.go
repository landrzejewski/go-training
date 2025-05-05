package exercises

import "fmt"

type monetaryAmount struct {
	value    float64
	currency string
}

var currencyMismatch = fmt.Errorf("currnency mismatch")

func newMonetaryAmount(value float64, currency string) *monetaryAmount {
	return &monetaryAmount{value, currency}
}

/*func (ma *monetaryAmount) add(monetaryAmount *monetaryAmount) error {
	if ma.currency != monetaryAmount.currency {
		return currencyMismatch
	}
	ma.value += monetaryAmount.value
	return nil
}

func (ma *monetaryAmount) subtract(monetaryAmount *monetaryAmount) error {
	if ma.currency != monetaryAmount.currency {
		return currencyMismatch
	}
	ma.value -= monetaryAmount.value
	return nil
}*/

func (ma *monetaryAmount) add(amount *monetaryAmount) error {
	return apply(ma, amount, func(monetaryAmount, otherMonetaryAmount *monetaryAmount) {
		monetaryAmount.value += otherMonetaryAmount.value
	})
}

func (ma *monetaryAmount) subtract(amount *monetaryAmount) error {
	return apply(ma, amount, func(monetaryAmount, otherMonetaryAmount *monetaryAmount) {
		monetaryAmount.value -= otherMonetaryAmount.value
	})
}

func apply(monetaryAmount, otherMonetaryAmount *monetaryAmount, operator func(monetaryAmount, otherMonetaryAmount *monetaryAmount)) error {
	if monetaryAmount.currency != otherMonetaryAmount.currency {
		return currencyMismatch
	}
	operator(monetaryAmount, otherMonetaryAmount)
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
	if amount.add(otherAmount) != nil {
		fmt.Println("Currency mismatch")
		return
	}
	fmt.Println(amount)
}
