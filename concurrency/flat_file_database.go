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
	"os"
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

type Database struct {
	file *os.File
}

func (db *Database) Close() error {
	return db.file.Close()
}

func (db *Database) writeBytes(bytes []byte, offset int64) (int, error) {
	length, err := db.file.WriteAt(bytes, offset)
	if err != nil {
		return 0, err
	}
	return length, nil
}

func (db *Database) readBytes(offset int64, size int) ([]byte, error) {
	buffer := make([]byte, size)
	_, err := db.file.ReadAt(buffer, offset)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func (db *Database) Read(offset int64, size int, output interface{}) error {
	buffer, err := db.readBytes(offset, size)
	if err != nil {
		return err
	}
	err = fromBytes(buffer, output)
	if err != nil {
		return err
	}
	return nil
}

func (db *Database) Write(offset int64, input interface{}) (int, error) {
	buffer, err := toBytes(input)
	if err != nil {
		return 0, err
	}
	var length, erro = db.writeBytes(buffer, offset)
	if erro != nil {
		return 0, erro
	}
	return length, nil
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
