package main

import "fmt"

func helloWorld() {
	fmt.Println("Hello World!")
}

// var currentYear = 2024
const CURRENT_YEAR int = 2024; 

func main() {
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

func main() {
	
}
