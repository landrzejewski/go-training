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
	"sync"
	"sync/atomic"
	"training.pl/examples/utils"
)

type IdGenerator interface {
	next() any
}

type Sequence struct {
	counter int64
}

func (g *Sequence) next() any {
	counter := atomic.AddInt64(&g.counter, 1)
	return counter
}

type Record struct {
	Id     any
	Offset int64
	Length int64
}

const stateFileExtension = ".state"

type Database struct {
	file        *os.File
	mutex       sync.RWMutex
	records     []Record
	idGenerator IdGenerator
}

func (db *Database) Close() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	buffer, err := utils.ToBytes(db.records)
	if err != nil {
		return err
	}

	err = os.WriteFile(db.file.Name()+stateFileExtension, buffer, 0666)
	if err != nil {
		return err
	}

	return db.file.Close()
}

func (db *Database) Write(offset int64, input interface{}) (*Record, error) {
	buffer, err := utils.ToBytes(input)
	if err != nil {
		return nil, err
	}
	db.mutex.Lock()
	defer db.mutex.Unlock()
	length, err := db.file.WriteAt(buffer, offset)
	if err != nil {
		return nil, err
	}
	record := Record{db.idGenerator.next(), offset, int64(length)}
	db.records = append(db.records, record)
	return &record, nil
}

func (db *Database) Read(offset int64, size int64, output interface{}) error {
	buffer := make([]byte, size)
	db.mutex.RLock()
	_, err := db.file.ReadAt(buffer, offset)
	db.mutex.RUnlock()
	if err != nil {
		return err
	}
	err = utils.FromBytes(buffer, output)
	if err != nil {
		return err
	}
	return nil
}

func Db(filePath string) (*Database, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	buffer, err := os.ReadFile(filePath + stateFileExtension)
	var records []Record
	if err != nil {
		records = make([]Record, 0)
	} else {
		err = utils.FromBytes(buffer, &records)
		if err != nil {
			return nil, err
		}
	}
	db := &Database{file, sync.RWMutex{}, records, &Sequence{0}}
	return db, nil
}

type User struct {
	FirstName string
	LastName  string
	IsActive  bool
}

func UsersDatabase() {
	db, _ := Db("users.db")
	defer db.Close()
	record1, _ := db.Write(0, &User{"Jan", "Kowalski", true})
	db.Write(record1.Offset+record1.Length, &User{"Jan2", "Kowalski2", true})
	var user User
	db.Read(db.records[1].Offset, db.records[1].Length, &user)
	fmt.Println(&user)
	db.Read(db.records[0].Offset, db.records[0].Length, &user)
	fmt.Println(&user)
}
