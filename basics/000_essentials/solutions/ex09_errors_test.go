package solutions

import (
	"errors"
	"fmt"
	"strconv"
)

// ParseAge has a proper test in ex14_testing_test.go - writing it is exercise 14.2.
func ExampleParseAge() {
	fmt.Println(ParseAge("42"))
	_, err := ParseAge("abc")
	fmt.Println(err)
	fmt.Println(errors.Is(err, strconv.ErrSyntax)) // the wrapped strconv error is still there
	_, err = ParseAge("200")
	fmt.Println(err)
	fmt.Println(errors.Is(err, ErrAgeOutOfRange))
	// Output:
	// 42 <nil>
	// parse age "abc": strconv.Atoi: parsing "abc": invalid syntax
	// true
	// parse age 200: age out of range
	// true
}

func ExampleWithdraw() {
	fmt.Println(Withdraw(100, 30))
	_, err := Withdraw(100, 130)
	var fundsErr *InsufficientFundsError
	if errors.As(err, &fundsErr) { // finds the *InsufficientFundsError inside the wrap
		fmt.Printf("%v\nmissing: %.2f\n", err, fundsErr.Missing())
	}
	_, err = Withdraw(100, -5)
	fmt.Println(err, errors.As(err, &fundsErr))
	// Output:
	// 70 <nil>
	// withdraw: insufficient funds: balance 100.00, requested 130.00
	// missing: 30.00
	// withdraw: amount must be positive false
}

func ExamplePort() {
	for _, settings := range []map[string]string{
		{"port": "8080"},
		{},
		{"port": "http"},
		{"port": "70000"},
	} {
		p, err := Port(settings)
		if errors.Is(err, ErrMissingSetting) { // the caller decides: a missing port gets a default
			p, err = 80, nil
		}
		fmt.Println(p, err)
	}
	// Output:
	// 8080 <nil>
	// 80 <nil>
	// 0 config: setting "port": strconv.Atoi: parsing "http": invalid syntax
	// 0 config: port 70000 out of range 1-65535
}
