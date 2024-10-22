/*
Stwórsz bazę danych opartą o plik płaski przechowującą dane w postaci binarnej (https://gobyexample.com/reading-files).
Baza powinna umożliwiać wykonywanie następujących operacje: ADD, READ, UPDATE, DELETE na podstwie podango id rekordu.
W celu uzyskania lepszej wydajnoći, wprowadź indeksowanie pozycji rekordu w pliku oraz pamięć podręczną (wykorzystaj mapy).
Pomyśl o optymalnym sposobie usuwania rekordów i ponownym wykorzystaniem miejsca po usuniętym rekordzie.
Załóż, że każdy rekord ma stałą długość (ilośc bajtów)
*/

package concurrency

import (
	"bytes"
	"encoding/binary"
)

const (
	idSize        = 8
	firstNameSize = 128
	lastNameSize  = 128
	ageSize       = 8
	recordSize    = idSize + firstNameSize + lastNameSize + ageSize
)

type userRecord struct {
	Id        [idSize]byte
	FirstName [firstNameSize]byte
	LastName  [lastNameSize]byte
	Age       [ageSize]byte
}

func (u *userRecord) getId() int64 {
	return fromBytes(u.Id[:])
}

func (u *userRecord) getFirstName() string {
	return string(u.FirstName[:])
}

func (u *userRecord) getLastName() string {
	return string(u.LastName[:])
}

func (u *userRecord) getAge() int64 {
	return fromBytes(u.Age[:])
}

func newUserRecord(id int64, firstName, lastName string, age int64) *userRecord {
	var firstNameBytes [firstNameSize]byte
	copy(firstNameBytes[:], firstName)

	var lastNameBytes [lastNameSize]byte
	copy(lastNameBytes[:], lastName)

	return &userRecord{
		Id:        int64ToBytes(id),
		FirstName: firstNameBytes,
		LastName:  lastNameBytes,
		Age:       int64ToBytes(age),
	}
}

func int64ToBytes(value int64) (arr [8]byte) {
	binary.LittleEndian.PutUint64(arr[0:8], uint64(value))
	return
}

func fromBytes[T int64](data []byte) (value T) {
	err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &value)
	if err != nil {
		panic("Reading bytes failed")
	}
	return
}

func UsersDatabase() {
}
