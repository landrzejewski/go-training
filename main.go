package main

import "fmt"

func helloWorld() {
	fmt.Println("Hello World!")
}

// var currentYear = 2024
const CURRENT_YEAR int = 2024; 

func basics() {
	// Deklaracja zmiennej:

	// var nazwa_zmiennej [typ] = wartość
	// lub
	// nazwa_zmiennej := wartość

	var firstName string = "Jan"
	var lastName = "Kowalski"
	var age int
	age = 32

	email := "jan.kowalski@training.pl" // tylko w ramach funkcji

	/*
	var a, b, c int = 1, 2, 3
	var d, text = 4, "Hello"

	e, otherText := 5, "World"
	*/

	fmt.Println(firstName)
	fmt.Println(lastName)
	fmt.Println(age)
	fmt.Println(email)

	// Typy danych

	// Booleans

	// bool może przechowywać wartość true lub false
	var result bool // domyślnie zmienn bool  mają przypisaną wartość false 
	result = true
	fmt.Println(result)
	
	// Integers - przechowuje wartośąci całkowite

	// int - rozmar zależy od platformy 32bit/64bit, typ domyślny dla literałów całkowitych
	// int8 - 8bit/1byte
	// int16 - 16bit/2byte
	// int32 - 32bit/4byte
	// int64 - 64bit/8byte 

	// uint - rozmar zależy od platformy 32bit/64bit, brak wartości ujemnych
	// uint8 - 8bit/1byte, brak wartości ujemnych (0-255)
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
	fmt.Printf("Wartość z paddingiem %010.2f", 1100.232232); // padding symbol_wilekość paddingu_precyzja_formatowanie 

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

	// Instrukcja warunkowa if

	inputValue := 5

	if value % 2 == 0 { // wynaga wyrażenia zwracającego bool, nie zapisujemy nawiasów
		fmt.Printf("Value %v is even\n", inputValue)
	} else {
		fmt.Printf("Value %v is not even\n", inputValue)
	}

	// wyrażenia logiczne mogą być skracane, jeśli ich rezultat jest znany po rozwinięciu ich części
	// blok else jest opcjonalny
	// można dodać wiele bloków if else
	
	// Instrukcja switch

	switch inputValue {
	case 1:
		fmt.Println("First")
	case 2:
		fmt.Println("Second")
	case 3, 4, 5:
		fmt.Println("Small values")	
	default:
		fmt.Println("Other")
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

	colors := [3]string { "red", "blue", "yellow" }
	for idx, color := range colors {
		fmt.Printf("Color: %v has idex %v\n", color, idx)
	}

	for {
		fmt.Printf("Tick\n")
		break
	}
}

func fahrenheit() {
	for range 3 {
		var fahrenheit float64
		fmt.Print("Enter temperature in Fahrenheit: ")
		fmt.Scan(&fahrenheit)
		celsius := (fahrenheit - 32) / (9/5)
		fmt.Printf("Temperature in celsius: %.2f°\n", celsius)  
	}
}

func collections() {
	// Arrays
	// zmienna := [length/...]typ_danych{values}
	
	var numbers = [...]int{1, 2, 3}
	fmt.Println(numbers)
	fmt.Printf("3rd element: %d\n", numbers[2])
	numbers[2] = 4
	fmt.Printf("3rd element: %d\n", numbers[2])
	fmt.Printf("Numbers length: %d\n", len(numbers))

	var values = [100]int{1, 10:3, 99:100}
	fmt.Println(values)

	// values[101] = 10 // error - index out of bounds (na poziomie kompilacji)

	dimensions := [3][3]int{
		{1, 2, 3},
		{4, 5, 6},
	}
	fmt.Printf("Element: %d\n", dimensions[0][0])

	// Slices
	// zmienna := []typ_danych{values}

	numbersSlice := []int{1, 2, 3}
	fmt.Println(numbersSlice)
	fmt.Printf("Numbers slice length: %d\n", len(numbersSlice))
	fmt.Printf("Numbers slice capacity (internal array length): %d\n", cap(numbersSlice))

	messages := [5]string{}
	messagesSlice := messages[0:3]
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

	var newRatings = ratings;
	newRatings["a"] = 1.0

	for key, value := range ratings {
		fmt.Printf("%v: %f\n", key, value)
	}

}

func double(value int) int {
	value += 1
	return value * 2
}

func doubleWithPointer(valuePointer *int) int {
	*valuePointer += 1
	return *valuePointer * 2
}

func main() {
	value := 10
	otherValue := value; // kopia wartości, otherValue jest równe 10
	value = 0
	
	/*
	fmt.Printf("Value: %v\n", value)
	fmt.Printf("Value: %v\n", otherValue)
	*/

	/*
	result := double(otherValue) // kopia wartości
	fmt.Printf("Value: %v\n", value)
	fmt.Printf("Value: %v\n", otherValue)
	fmt.Printf("Result: %v\n", result)
	*/

	otherResult := doubleWithPointer(&otherValue) // przkazujemy wskaźnik na adres pamięci zawierającej wartość otherValue
	fmt.Printf("Value: %v\n", value)
	fmt.Printf("Value: %v\n", otherValue)
	fmt.Printf("Other result: %v\n", otherResult)

	// standardowo użycie operatora przypisania (=) powoduje utworzenie kopii wartości, wyjątkiem są maps i slices
	// używając wskaźników (*nazwa_typu) jest w stanie operować (odczyt/zapis) na wskazanym adresie pamięci 
	// użycie * przed zmienna wskaźnikową pozwala na dostanie się do wartości wskazywanej przez wskaźnik
	// użycie & pozwala na odczyt adresu pamięci przechowującej daną wartość  

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
}

