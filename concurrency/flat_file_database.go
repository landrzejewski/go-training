/*
Stwórsz bazę danych opartą o plik płaski przechowującą dane w postaci binarnej (https://gobyexample.com/reading-files).
Baza powinna umożliwiać wykonywanie następujących operacje: ADD, READ, UPDATE, DELETE na podstwie podango id rekordu.
W celu uzyskania lepszej wydajnoci, wprowadź indeksowanie pozycji rekordu w pliku oraz pamięć podręczną (wykorzystaj mapy).
Pomyśl o optymalnym sposobie usuwania rekordów i ponownym wykorzystaniem miejsca po usuniętym rekordzie.
*/

package concurrency

import (
	"fmt"
	"os"
	"training.pl/examples/utils"
)

type Database struct {
	file *os.File
}

func (db *Database) Close() error {
	return db.file.Close()
}

func (db *Database) Write(offset int64, input interface{}) (int, error) {
	buffer, err := utils.ToBytes(input)
	if err != nil {
		return 0, err
	}
	length, err := db.file.WriteAt(buffer, offset)
	if err != nil {
		return 0, err
	}
	return length, nil
}

func (db *Database) Read(offset int64, size int, output interface{}) error {
	buffer := make([]byte, size)
	_, err := db.file.ReadAt(buffer, offset)
	if err != nil {
		return err
	}
	err = utils.FromBytes(buffer, output)
	if err != nil {
		return err
	}
	return nil
}

func newDatabase(filePath string) (*Database, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	db := &Database{file}
	return db, nil
}

type User struct {
	Id        int
	FirstName string
	LastName  string
	IsActive  bool
}

func UsersDatabase() {
	db, _ := newDatabase("users.db")
	defer db.Close()
	length, _ := db.Write(0, &User{1, "Jan", "Kowalski", true})
	var user User
	db.Read(0, length, &user)
	fmt.Println(&user)
}
