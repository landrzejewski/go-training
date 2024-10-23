/*
Stwórsz bazę danych opartą o plik płaski przechowującą dane w postaci binarnej (https://gobyexample.com/reading-files).
Baza powinna umożliwiać wykonywanie następujących operacje: ADD, READ, UPDATE, DELETE na podstwie podango id rekordu.
W celu uzyskania lepszej wydajnoci, wprowadź indeksowanie pozycji rekordu w pliku oraz pamięć podręczną (wykorzystaj mapy).
Pomyśl o optymalnym sposobie usuwania rekordów i ponownym wykorzystaniem miejsca po usuniętym rekordzie.
*/

package concurrency

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

func toBytes(input interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(input)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func fromBytes(data []byte, output interface{}) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(output)
	if err != nil {
		return err
	}
	return nil
}

type User struct {
	Id        int
	FirstName string
	LastName  string
	IsActive  bool
}

func UsersDatabase() {
	textBytes, _ := toBytes("Ala ma kota")
	var text string
	fromBytes(textBytes, &text)
	fmt.Println(text)

	valueBytes, _ := toBytes(22)
	var value int
	fromBytes(valueBytes, &value)
	fmt.Println(value)

	userBytes, _ := toBytes(&User{1, "Jan", "Kowalski", true})
	var user User
	fromBytes(userBytes, &user)
	fmt.Println(&user)
}
