package main

import "fmt"

func main() {
	// fmt.Println("Hello World")
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