package main

import (
	"cmp"
	"fmt"
	"slices"

	// "training.pl/go/common"
	// . "training.pl/go/common" // import bez prefiksu/namespace
	// c "training.pl/go/common" // import z aliasowaniem
	// _ "training.pl/go/common" // ignorowanie błędu kompilacji (przydatne kiedy chcemy załadować moduł, który np. wykonuje jakiś kod)
)

func main() {
	defer close()
	defer func() {
		fmt.Println("Other close")
	}()

	// pointers()

	// common.Add(1, 3)
	// Add(1, 3)
	// c.Add(1, 3)

	// fmt.Println("Input value:")
	// var input string
	// _, err := fmt.Scanln(&input)
	// if err != nil {
	// 	fmt.Printf("Error")
	// }

	structs()
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

	divResult, errorMessage := div(10, 0)
	fmt.Println(divResult, errorMessage)

	sumAll(1, 2, 3, 4, 5)
	values := []int{1, 2, 3, 4}
	sumAll(values...) // spread

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

func sumAll(values ...int) (sum int) { // varargs - zmienna liczba argumentów
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

func collections() {
	// Tablice (Arrays)
	// nazwaZmiennej := [length/...]typDanych{values}, indeksy liczone od 0 do n - 1

	numbers := [...]int{1, 2, 3, 4}
	fmt.Println(numbers)
    // numbers[10] = 10 // błąd kompilacji, indeks poza zakresem
	fmt.Printf("3rd element: %d\n", numbers[2])
	numbers[0] = 0
	fmt.Println("Numbers length:", len(numbers))

	var values = [100]int{1, 10: 3, 99: 100}
	fmt.Println(values)

	dimensions := [3][3]int{
		{1, 2, 3},
		{3, 4, 5},
		{3, 4, 5},
	}
	fmt.Println(dimensions)
	fmt.Println(dimensions[0][0])

	// Slices
	// zmienna := []typDanych{values}

	numbersSlice := []int{1, 2, 3}
	
	fmt.Println(numbersSlice)
	fmt.Printf("3rd element: %d\n", numbersSlice[2])
	fmt.Printf("Numbers slice length: %d\n", len(numbersSlice))
	fmt.Printf("Numbers slice capacity (internal array length): %d\n", cap(numbersSlice))

	messages := [5]string{"1", "2", "3", "4", "5"}
	messagesSlice := messages[0:3]
	messagesSlice[0] = "0"
	fmt.Println(messagesSlice)
	fmt.Println(messages)
	fmt.Printf("Messages slice length: %d\n", len(messagesSlice))
	fmt.Printf("Messages slice capacity (internal array length): %d\n", cap(messagesSlice))

	otherMessagesSlice := make([]string, 3, 300)
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

	
	fruits := []string{"peach", "banana", "kiwi"}

	lenCmp := func(a, b string) int {
		return cmp.Compare(len(a), len(b))
	}
		
	slices.SortFunc(fruits, lenCmp)
	slices.SortFunc(fruits, fruitComparator)
	fmt.Println(fruits)
	

	customSlice := otherMessagesSlice[2:5]
	fmt.Println(customSlice)

	// Maps
	// zmienna := map[typKlucza]typWartości{key:value,...}
	// make(map[typKlucza]typWartości)

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
	newRatings["a"] = 99.0

	for key, value := range ratings {
		fmt.Printf("%v: %f\n", key, value)
	}
}

func fruitComparator(a, b string) int {
	return cmp.Compare(len(a), len(b))
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
	fmt.Printf("Other value: %v\n", otherValue)
	fmt.Printf("Other result: %v\n", otherResult)

	// Dla tablic
	var arr = [...]int{1, 2, 3}
	// var otherArr = arr // kopia wartości

	var otherArrPointer = &arr // adres/wskazanie na adres oryginalnej tablicy w pamięci
	otherArrPointer[0] = 0     // (*otherArrPointer)[0] = 0

	fmt.Println(arr)
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

	var age = 10
	var agePointer = &age
	// var abc = &agePointer
	fmt.Println("Age:", age)
	fmt.Println("Age:", *agePointer)
	var doubleResult = doubleWithPointer(agePointer)
	fmt.Println("Result:", doubleResult)
	fmt.Println("Age:", age)
	fmt.Println("Age:", *agePointer)
}

func double(value int) int {
	value += 1
	return value * 2
}

func doubleWithPointer(valuePointer *int) int {
	*valuePointer += 1
	return *valuePointer * 2
}

func structs() {
	var myPath path = "/Users/Jan"
	myPath.show()

	fmt.Println(person{"Jan", "Kowalski", 40})                  // bez podawania kluczy, kolejność ma znacznie
	fmt.Println(person{lastName: "Kowalski", firstName: "Jan"}) // z podawaniem kluczy, kolejność nie ma znacznia, można nie podawać wszytskich elementów

	user := person{"Jan", "Kowalski", 40}
	// otherUser := user // utworzenie kopii
	otherUser := &user // wskaźnik na oryginał
	fmt.Println(*otherUser)

	user.lastName = "Nowak"
	user.show()

	admin := createPerson("Anna", "Nowak", 20)
	admin.show()

	account := struct {
		number  string
		balance float64
	}{
		"0000001",
		0.0,
	}
	fmt.Println(account)

	myRect := rect{10.0, 10.0}
	showShape(&myRect)

	//address := address{"Dobra", 38}
	//address.show()
	myClient := client{
		"Jan Kowalski",
		address{"Dobra", 38},
	}
	myClient.show()
	myClient.address.show()
}

// alias typu
type text = string

// utworzenie nowego typu na bazie istniejącego
type path string

func (p *path) show() {
	fmt.Println(*p)
}

type person struct {
	firstName string
	lastName  string
	age       int
}

// działanie na wskaźniku pozwala na uniknąć tworzenia kopii struktury
func (p *person) show() {
	fmt.Println(*p)
}

func createPerson(fistName, lastName string, age int) *person {
	return &person{fistName, lastName, age}
}

type display interface {
	show()
}

type shape interface {
	area() float64
	// show()
	display
}

type rect struct {
	width, height float64
}

func (r *rect) area() float64 {
	return r.width * r.height
}

func (r *rect) show() {
	fmt.Println("rect", r.width, r.height)
}

func showShape(shape shape) {
	shape.show()
}

// type nesting
/*type client struct {
	name string
	address address
}*/

// struct embedding
type client struct {
	name string
	address
}

type address struct {
	street      string
	houseNumber int
}

func (c *client) show() {
	fmt.Println("Client:", c.name)
}

func (a *address) show() {
	fmt.Println("Address:", a.street, a.houseNumber)
}

func close() {
	fmt.Println("close()")
}
