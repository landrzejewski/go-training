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


}