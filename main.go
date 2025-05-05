package main

import "fmt"

func main() {
	// fmt.Println("Hello World")
	functions()
}

const CurrentYear = 5

var currentMonth = 2025

func variables() {
	// var nazwaZmiennej [typ] = wartość
	// lub
	// nazwaZmiennej := wartość (tylko na poziomie funkcji)

	var firstName string = "Jan"
	var lastName = "Kowalski"
	var age int
	age = 32

	email := "jan@training.pl"

	fmt.Println(firstName + " " + lastName + ", " + email)
	fmt.Println(age)

	/*
		var a, b, c int = 1, 2, 3
		var d, text = 4, "Hello"

		e, otherText := 5, "World"
	*/
}


func basicTypes() {
	/*
			int - rozmiar zależy od platformy (32bit/64bit), typ domyślny dla literałów całkowitych
		    dodatkowo występują int8, int16, int32, int64

			uint - rozmiar zależy od platformy (32bit/64bit), tylko wartości dodatnie
			dodatkowo występują uint8, uint16, uint32, uint64

			float64, float32 - reprezentują wartości zmiennoprzecinkowe, domyślnie float64

			bool - przechowuje wartości true lub false

			string - przechowuje tekst zakodowany w utf8
	*/

	// Jeżeli zmienna nie zostanie zainicjalizowana wprost to będzie ona posiadała wartość tzw. zerową/domyślną
	var salary int
	var isActive bool
	var name string
	fmt.Println(salary)
	fmt.Println(isActive)
	fmt.Println(name)
}

func controlFlow() {
	fmt.Printf("Użytkownik %v %v ma wiek zdefiniowany jako %T\n", "Jan", "Kowalski", 32)
	fmt.Printf("Pensja użytkownika wynosi: %010.2f\n", 2300.23445)

	/*
		Operatory arytmetyczne
		+	dodawanie
		-	odejmowanie
		*	mnożenie
		/	dzielenie
		%	dzielenie modulo
		++	inkrementacja
		--	dekrementacja
	*/

	var value = 100
	var otherValue = 20.0
	var calculationResult = float64(value) * otherValue // typ musi być jawanie skonwertowany
	fmt.Printf("Result: %.2f\n", calculationResult)

	/*
		Operatory przypisania
		=    przypisanie
		+=, -=, *=, /=, %=, &=, |=, ^=, >>=, <<=  skrócony zapis x = x [operator] x
	*/

	/*
		Operatory porównania
		==   równość
		!=   nierówność
		>    większy
		<    mniejszy
		>=   większy/równy
		<=   mniejszy/równy
	*/

	/*
		Operatory logiczne
		&&   i
		||   lub
		!    zaprzeczenie
	*/

	/*
		Operatory bitowe
		&    i
		|    lub
		^    xor
		<<   przesunięcie bitów w lewo
		>>   przesunięcie bitów w prawo
	*/

	// Instrukcja warunkowa

	inputValue := 5

	if inputValue%2 == 0 { // wymaga wyrażenia zwracającego bool, nie zapisujemy nawiasów
		fmt.Printf("Value %v is even \n", inputValue)
	} else { // else musi wystąpić po nawiasie klamrowym
		fmt.Printf("Value %v is not even \n", inputValue)
	}

	// wyrażenia logiczne mogą być skracane, jeśli ich rezultat jest znany po rozwinięciu ich części
	// blok else jest opcjonalny
	// można dodać wiele bloków if else

	// Instrukcja switch

	switch inputValue {
	case 1:
		fmt.Println("One")
	case 2:
		fmt.Println("Two")
	case 3, 4, 5:
		fmt.Println("Greater than 2")
	default:
		{
			fmt.Printf("Unknown")
		}
	}

	switch {
	case inputValue <= 2:
		fmt.Println("Lower than 3")
	case inputValue > 3:
		fmt.Println("Greater than 2")
	}

	/*
		var otherValue any;

		switch otherValue.(type) {
		case bool:
			fmt.Println("Bool")
		case int:
			fmt.Println("Int")
		default:
			fmt.Println("Unknown")
		}
	*/

	// Pętle

	for i := 0; i < 10; i++ {
		if i == 5 {
			continue
		}
		fmt.Printf("Counter: %v\n", i)
		if i > 7 {
			break
		}
	}

	j := 0
	for j <= 10 {
		fmt.Printf("Counter: %v\n", j)
		j++
	}

	for x := range 5 {
		fmt.Printf("Counter: %v\n", x)
	}

	colors := [3]string{"red", "blue", "yellow"}
	for idx, color := range colors {
		fmt.Printf("Color: %v has index %v \n", color, idx)
	}

	for {
		fmt.Println("GO")
		break
	}
}

func functions() {
	fmt.Printf("Sum: 2 + 3 = %v\n", add(2, 3))

	divResult, errorMessage := div(10, 2)
	fmt.Println(divResult, errorMessage)

	sumAll(1, 2, 3, 4)
	values := []int{1, 2, 3, 4}
	sumAll(values...)

	forEach(values, func(value, _ int) {
		fmt.Println(value)
	})

	forEach(values, showElement)

	var firstGenerator = idGeneratorFactory()
	fmt.Println(firstGenerator())
	fmt.Println(firstGenerator())

	var secondGenerator = idGeneratorFactory()
	fmt.Println(secondGenerator())

	fmt.Println(firstGenerator())
}

/*
func add(value int, otherValue int) int {
	return value + otherValue
}
*/

func add(value, otherValue int) (sum int) { // ten sam typ dla parametrów wejściowych i nazwany typ zwracany
	sum = value + otherValue
	return // naked return, zwraca zadeklarowany rezultat - sum
}

func div(value float64, divident float64) (float64, string) { // zwracanie kilku rezultatów z funkcji
	if divident == 0 {
		return 0.0, "Division by zero"
	}
	return value / divident, ""
}

// rekurencja
func factorial(n int) int {
	if n == 0 {
		return 1
	}
	return n * factorial(n-1)
}

func forEach(numbers []int, task func(int, int)) {
	for idx, number := range numbers {
		task(number, idx)
	}
}

func showElement(value, idx int) {
	fmt.Printf("Value: %v (idx:%v)\n", value, idx)
}

func sumAll(values ...int) (sum int) {
	for _, value := range values {
		sum += value
	}
	return
}

func idGeneratorFactory() func() int {
	lastId := 0
	return func() int {
		lastId++
		return lastId
	}
}

func fahrenheitToCelsius() (clesius float64) {
	var fahrenheit float64
	fmt.Println("Enter fahrenheit: ")
	_, err := fmt.Scan(&fahrenheit)
	if err != nil {
		panic("Invalid input")
	}
	clesius = (fahrenheit - 32) * 5 / 9
	return
}