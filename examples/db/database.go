package db

import (
	"log"
	"os"
)

type Record struct {
	Id     int64
	Offset int64
	Length int64
}

type Database struct {
	file        *os.File
	records     []Record
	idGenerator IdGenerator
}

func (d *Database) Close() {
	err := d.file.Close()
	if err != nil {
		log.Fatalf("Error closing database")
	}
}

func Db(filepath string) (*Database, error) {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("Error opening database")
	}

	records := make([]Record, 0)

	idGenerator := &Sequence{}

	db := &Database{file, records, idGenerator}

	return db, nil
}

type User struct {
	FirstName string
	LastName  string
	Age       int16
	IsActive  bool
}

func DatabaseExample() {
	db, _ := Db("users.db")
	defer db.Close()
}
