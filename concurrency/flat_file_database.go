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
	Length int
}

type Database struct {
	file        *os.File
	mutex       sync.RWMutex
	records     []Record
	idGenerator IdGenerator
}

func (db *Database) Close() error {
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
	record := Record{db.idGenerator.next(), offset, length}
	db.records = append(db.records, record)
	return &record, nil
}

func (db *Database) Read(offset int64, size int, output interface{}) error {
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
	db := &Database{file, sync.RWMutex{}, make([]Record, 0), &Sequence{0}}
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
	record, _ := db.Write(0, &User{"Jan", "Kowalski", true})
	var user User
	db.Read(record.Offset, record.Length, &user)
	fmt.Println(record.Id, &user)
}
