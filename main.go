package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	// "training.pl/examples/utils" // prefiks to nazwa pakietu - utils
	// . "training.pl/examples/utils" // import bez prefiksu
	// u "training.pl/examples/utils" // nistandardowy alias
	_ "training.pl/examples/utils" // ignorowanie nieużywanego importu
)

func main() {
	// utils.Add(1,2)
	// Add(1, 2)
	// u.Add(1,2)
}

func structs() {
	fmt.Println(person{"Jan", "Kowalski", 32})            // można nie podawać kluczy/nazw pól, ale wtedy ważna jest kolejność
	user := person{lastName: "Nowak", firstName: "Marek"} // niepodanie wartości skutkuje ustawieniem pola na wartość domyślną
	// otherUser := user // utworzenie kopii
	fmt.Println(&user)
	fmt.Println(newPerson("Jan", "Kowalski", 32))
	user.lastName = "Test"
	otherUser := &user // utworzenie referencji
	fmt.Println(otherUser.lastName)
	user.setAge(20)
	fmt.Println(user.description())

	// Singleton
	account := struct {
		number  string
		balance float64
	}{
		"00000000001",
		0.0,
	}
	fmt.Println(&account)

	monetaryAmountExercise()
}

// custom types / type aliases
type text string

func (t *text) print() {
	fmt.Println(*t)
}

func typeAliases() {
	var helloText text = "Hello"
	helloText.print()
}

func interfaces() {
	printArea(&rectangle{10, 20})
	printArea(&circle{100})
}

type display interface {
	getInfo() string
}

type shape interface {
	area() float64
	// getInfo() string
	// display
}

type rectangle struct {
	width, height float64
}

type circle struct {
	radius float64
}

func (r *rectangle) area() float64 {
	return r.height * r.width
}

func (c *circle) area() float64 {
	return math.Pi * c.radius * c.radius
}

func printArea(shape shape) {
	fmt.Printf("Area of %T is equal %.2f\n", shape, shape.area())
}

// Struct embedding
type address struct {
	street      string
	houseNumber int
}

func (a *address) description() string {
	return fmt.Sprintf("%v %d", a.street, a.houseNumber)
}

type user struct {
	name string
	// street string
	address
}

// func (u *user) description() string {
// 	return fmt.Sprintf("%v", u.name)
// }

func structEmbedding() {
	myUser := user{
		name: "Jan Kowalski",
		address: address{
			street:      "Dobra",
			houseNumber: 38,
		},
	}
	fmt.Println(myUser)
	fmt.Println(myUser.address.description())
	fmt.Println(myUser.description())
}

type person struct {
	firstName string
	lastName  string
	age       int
}

// w przypadku kiedy nie chcemy, aby wywołanie metody spowodowało utworzenie kopii struktory należy użyć wskaźnika/*
func (p *person) description() string {
	return fmt.Sprintf("{name: %v %v, age: %d}", p.firstName, p.lastName, p.age)
}

func (p *person) setAge(age int) {
	p.age = age
}

func newPerson(firstName, lastName string, age int) *person {
	return &person{firstName, lastName, age}
}

func handlingErrors() {
	result, err := divide(100.0, 2.0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}

	if error := readData("path"); error != nil {
		if errors.Is(error, ErrFileNotFound) {
			fmt.Println("File not found")
		} else if errors.Is(error, ErrRead) {
			fmt.Println("Read failed")
		}
	} else {
		fmt.Println("Done")
	}

	_, resultError := calculate(0)
	var errPointer *customError
	if errors.As(resultError, &errPointer) {
		fmt.Println(errPointer.code, errPointer.description)
	}

	// panic(err) // błąd krytyczny, przerwanie działania aplikacji
}

func divide(value, divident float64) (float64, error) {
	if divident == 0 {
		return 0, errors.New("division by zero")
	}
	return value / divident, nil
}

var ErrFileNotFound = fmt.Errorf("file not found")
var ErrRead = fmt.Errorf("file read failed")

func readData(path string) error {
	if path == "" {
		return ErrFileNotFound
	}
	// read
	// return fmt.Errorf("IO error", ErrRead)
	return nil
}

// Niestandardowa struktura reprezentująca błąd/sytuację wyjątkową

type customError struct {
	code        int
	description string
}

func (e *customError) Error() string {
	return fmt.Sprintf("%d: %s", e.code, e.description)
}

func calculate(value int) (int, error) {
	if value <= 0 {
		return -1, &customError{
			code:        1,
			description: "Value is too small",
		}
	}
	return value * 2, nil
}

func functions() {
	sum := sumAll(1, 2, 3, 4)
	fmt.Println(sum)
	values := []int{1, 2, 3, 4}
	sum = sumAll(values...)
	fmt.Println(sum)

	generator := idGenerator()
	fmt.Println(generator())
	fmt.Println(generator())
	generator2 := idGenerator()
	fmt.Println(generator2())
	fmt.Println(generator())

	forEach([]int{1, 2, 3, 4}, func(value, _ int) {
		fmt.Println(value)
	})
	forEach([]int{1, 2, 3, 4}, show)
}

func fahrenheit() {
	var fahrenheit float64
	fmt.Print("Enter temperature in Fahrenheit: ")
	fmt.Scan(&fahrenheit)
	// count, err := fmt.Scan(&fahrenheit)
	celsius := (fahrenheit - 32) / (9 / 5)
	fmt.Printf("Temperature in celsius: %.2f°\n", celsius)
}

func factorial(n int) int { // rekurencja
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

func show(value int, index int) {
	fmt.Printf("Value: %v\n", value)
}

func sumAll(values ...int) (sum int) { // zmienna ilość argumentów wejściowych, nazwany rezultat
	for _, value := range values {
		sum += value
	}
	return // zwaracamy wszystkie nazwane zmienne
}

func idGenerator() func() int {
	lastId := 0
	return func() int {
		lastId++
		return lastId
	}
}

func readingAndParsingStandardInput() {
	/*
		reader := bufio.NewReader(os.Stdin)
		text, readErr := reader.ReadString('\n')
		if readErr == nil {
			text = strings.TrimSuffix(text, "\n")
			text = strings.TrimSuffix(text, "\r")
			fmt.Println(text)
		}
	*/

	/*
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		text := scanner.Text()
		if err := scanner.Err(); err == nil {
			fmt.Println(text)
		}
	*/

	value := 3.1415
	formattedValue := fmt.Sprintf("%.2f", value)
	fmt.Println(formattedValue)

	parsedValue, _ := strconv.ParseFloat(formattedValue, 64)
	fmt.Println(parsedValue)
}

func pointers() {
	value := 10
	otherValue := value // kopia wartości, otherValue jest równe 10
	value = 0
	fmt.Printf("Value: %v\n", value)
	fmt.Printf("Other value: %v\n", otherValue)

	result := double(otherValue) // kopia wartości

	fmt.Printf("Value: %v\n", value)
	fmt.Printf("Other value: %v\n", otherValue)
	fmt.Printf("Result: %v\n", result)

	otherResult := doubleWithPointer(&otherValue) // przkazujemy wskaźnik na adres pamięci zawierającej wartość otherValue
	fmt.Printf("Value: %v\n", value)
	fmt.Printf("Value: %v\n", otherValue)
	fmt.Printf("Other result: %v\n", otherResult)

	// Dla tablic
	var arr = [...]int{1, 2, 3}
	//var otherArr = arr // kopia wartości
	var otherArrPointer = &arr // adres/wskazanie na adres oryginalnej tablicy w pamięci
	//otherArr[0] = 0
	otherArrPointer[0] = 0 // (*otherArrPointer)[0] = 0
	fmt.Println(arr)
	//fmt.Println(otherArr)
	fmt.Println(otherArrPointer)

	// Dla slices i maps nie trzeba używać wskaźników (dzialamy na referencji/widoku)
	var slice = []int{1, 2, 3}
	var otherSlice = slice
	otherSlice[0] = 0
	fmt.Println(slice)
	fmt.Println(otherSlice)

	originalMap := map[string]int{"foo": 1, "bar": 2}
	copiedMap := originalMap // reference copy
	copiedMap["foo"] = 42
	// Both maps now reflect the change
	fmt.Println("Original map:", originalMap)
	fmt.Println("Copied map:", copiedMap)
}

func double(value int) int {
	value += 1
	return value * 2
}

func doubleWithPointer(valuePointer *int) int {
	*valuePointer += 1
	return *valuePointer * 2
}

func collections() {
	// Arrays
	// zmienna := [length/...]typ_danych{values}

	var numbers = [...]int{1, 2, 3, 4}
	fmt.Println(numbers)
	fmt.Printf("3rd element: %d\n", numbers[2])
	numbers[2] = 4
	fmt.Printf("3rd element: %d\n", numbers[2])
	fmt.Printf("Numbers length: %d\n", len(numbers))

	// numbers[101] = 10 // error - index out of bounds (na poziomie kompilacji)

	var values = [100]int{1, 10: 3, 99: 100}
	fmt.Println(values)

	dimensions := [3][3]int{
		{1, 2, 3},
		{4, 5, 6},
	}
	fmt.Printf("Element: %d\n", dimensions[0][0])

	// Slices
	// zmienna := []typ_danych{values}

	numbersSlice := []int{1, 2, 3}
	fmt.Println(numbersSlice)
	fmt.Printf("3rd element: %d\n", numbersSlice[2])
	fmt.Printf("Numbers slice length: %d\n", len(numbersSlice))
	fmt.Printf("Numbers slice capacity (internal array length): %d\n", cap(numbersSlice))

	messages := [5]string{"1", "2", "3", "4", "5"}
	messagesSlice := messages[0:3]
	fmt.Println(messagesSlice)
	fmt.Printf("Messages slice length: %d\n", len(messagesSlice))
	fmt.Printf("Messages slice capacity (internal array length): %d\n", cap(messagesSlice))

	otherMessagesSlice := make([]string, 3, 5)
	fmt.Println(otherMessagesSlice)
	fmt.Printf("Messages slice length: %d\n", len(otherMessagesSlice))
	fmt.Printf("Messages slice capacity (internal array length): %d\n", cap(otherMessagesSlice))
	otherMessagesSlice[2] = "Hello"
	fmt.Printf("3rd element: %v\n", otherMessagesSlice[2])

	fmt.Println(otherMessagesSlice)
	// otherMessagesSlice[6] = "Hi" //error - index out of bounds (na poziomie wykonania)

	otherMessagesSlice = append(otherMessagesSlice, "a", "b", "c", "d")
	fmt.Println(otherMessagesSlice)
	otherMessagesSlice[6] = "Hi"
	fmt.Println(otherMessagesSlice)

	// łączenie slices
	// newSlice = append(slice1, slice2, ...)
	// kopiowanie slices
	// copy(destSlice, srcSlice)
	// porównywanie slices
	// slices.Equal(slice1, slice2)
	// sortowanie
	// slices.Sort(slice)
	// sortowanie z użyciem funkcji
	/*
			fruits := []string{"peach", "banana", "kiwi"}

		    lenCmp := func(a, b string) int {
		        return cmp.Compare(len(a), len(b))
		    }
		    slices.SortFunc(fruits, lenCmp)
		    fmt.Println(fruits)
	*/

	customSlice := otherMessagesSlice[2:5]
	fmt.Println(customSlice)

	// Maps
	// zmienna := map[typ_klucza]typ_wartości{key:value,...}
	// make(map[typ_klucza]typ_wartości)

	var ratings = make(map[string]float64)
	ratings["a"] = 50.0
	ratings["b"] = 10.0
	fmt.Println(ratings)
	fmt.Printf("Value for key: a is equal %.0f\n", ratings["a"])
	fmt.Printf("Value for key: c is equal %.0f\n", ratings["c"]) // dla nieistniejącego klucza zwraca wartość domyślną np. 0 dla float64

	value, exists := ratings["a"]
	if exists {
		println(value)
	}

	// delete(ratings, "b") // usunięcie wpisu pod kluczem
	// clear(ratings) // wyczyszczenie całej mapy
	// fmt.Println(ratings)

	// porównywanie maps
	// maps.Equal(map1, map2)

	var newRatings = ratings
	newRatings["a"] = 1.0

	for key, value := range ratings {
		fmt.Printf("%v: %f\n", key, value)
	}
}

// var currentYear = 2024
// const CURRENT_YEAR int = 2024

func basics() {
	fmt.Println("Hello World")

	// Deklaracja zmiennej:

	// var nazwa_zmiennej [typ] = wartość
	// lub
	// nazwa_zmiennej := wartość

	var firstName string = "Jan"
	var lastName = "Kowalski"
	var age int
	age = 32

	email := "jan@trainingf.pl"

	fmt.Println(firstName + " " + lastName + " " + email + " ")
	fmt.Println(age)

	/*
		var a, b, c int = 1, 2, 3
		var d, text = 4, "Hello"

		e, otherText := 5, "World"
	*/

	// Integers - przechowuje wartości całkowite

	// int - rozmar zależy od platformy 32bit/64bit, typ domyślny dla literałów całkowitych
	// int8  -  8bit/1byte
	// int16 - 16bit/2byte
	// int32 - 32bit/4byte
	// int64 - 64bit/8byte

	// uint   - rozmar zależy od platformy 32bit/64bit, brak wartości ujemnych
	// uint8  - 8bit/1byte, brak wartości ujemnych (0-255)
	// uint16 - 16bit/2byte, brak wartości ujemnych (0-65525)
	// uint32 - 32bit/4byte, brak wartości ujemnych
	// uint64 - 64bit/8byte, brak wartości ujemnych

	// Floats

	// float64 - 64bit, typ domyślny dla literałów zmiennoprzecinkowych
	// float32 - 32bit

	// String

	// string - przechowuje teskst zakodowany w utf-8

	// Formatowanie tekstu

	fmt.Print(1, 2, 3, "\n")

	fmt.Printf("Użytkownik %v %v, ma wiek zdefiniowany jako %T\n", firstName, lastName, age)
	fmt.Printf("Procent (escape) %%\n")
	fmt.Printf("Wartość z paddingiem %010.2f", 1100.232232) // padding symbol_wilekość paddingu_precyzja_formatowanie

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

	inputValue := 5

	if inputValue%2 == 0 { // wymaga wyrażenia zwracającego bool, nie zapisujemy nawiasów
		fmt.Printf("Value %v is even \n", inputValue)
	} else { // else musi wystąpic po nawiasie klamrowym
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
		fmt.Printf("Unknown")
	}

	switch {
	case inputValue <= 2:
		fmt.Println("Lower than 3")
	case inputValue > 3:
		fmt.Println("Greater than 2")
	}

	/*
		var otherOnputValue any;

		switch otherOnputValue.(type) {
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

	colors := [3]string{"red", "blue", "yello"}
	for idx, color := range colors {
		fmt.Printf("Color: %v has index %v \n", color, idx)
	}

	for {
		fmt.Println("GO")
		break
	}
}

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

func monetaryAmountExercise() {
	amount := newMonetaryAmount(100.0, "PLN")
	otherAmount := newMonetaryAmount(100.0, "PLN")
	amount.add(otherAmount)
	fmt.Println(amount)
}

// enums
type responseStatus int

const (
	ok = iota
	noContent
	notFound
)

var statusName = map[responseStatus]string{
	ok:        "Ok",
	noContent: "No content",
	notFound:  "Not found",
}

func task() responseStatus {
	// logika
	return ok
}

type response struct {
	body   string
	status responseStatus
}

func enums() {
	status := notFound

	switch status {
	case ok, noContent:
		fmt.Println("Success")
	case notFound:
		fmt.Printf("Failure: %v", statusName[responseStatus(status)])
	}
}
