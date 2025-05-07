package db

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"sync"
)

const metadataFileSuffix = ".metadata"

type Record struct {
	Id     int64
	Offset int64
	Length int64
}

type Database struct {
	file        *os.File
	lock		sync.Mutex
	records     []Record
	idGenerator IdGenerator
}

func (d *Database) Close() {
	err := d.file.Close()
	if err != nil {
		log.Fatalf("Error closing database")
	}
	d.saveState()
}

func (d *Database) saveState() {
	byte, err := ToBytes(d.records)
	if err != nil {
		panic("Error marshaling records state")
	}
	err = os.WriteFile(d.file.Name() + metadataFileSuffix, byte, 0644)
	if err != nil {
		panic("Error saving records state")
	}
}

func getLastId(records []Record) int64 {
	if len(records) == 0 {
		return 0
	}
	var lastId int64
	for _, record := range records {
		if record.Id > lastId {
			lastId = record.Id
		}
	}
	return lastId
}

func (d *Database) findRecordIndex(id int64) int {
	return slices.IndexFunc(d.records, func(record Record) bool { return record.Id == id })
}

func Db(filepath string) (*Database, error) {
	// Opening data file
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("Error opening database")
	}

	// Restoring db state / metadata
	var records []Record
	byte, err := os.ReadFile(filepath + metadataFileSuffix)
	if err != nil {
		records = make([]Record, 0)
	} else {
		err = FromBytes(byte, &records)
		if err != nil {
			panic("Invalid metadata file")
		}
	}

	idGenerator := &Sequence{getLastId(records)}

	db := &Database{file, sync.Mutex{}, records, idGenerator}

	return db, nil
}

func (d *Database) Insert(input interface{}) (*Record, error) {
	bytes, err := ToBytes(input)
	if err != nil {
		return nil, err
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	offset, err := d.file.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	length, err := d.file.WriteAt(bytes, offset)
	if err != nil {
		return nil, err
	}
	record := Record{d.idGenerator.next(), offset, int64(length)}
	d.records = append(d.records, record)
	d.saveState()
	return &record, nil
}

func (d *Database) FindById(id int64, output interface{}) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	index := d.findRecordIndex(id)
	if index == -1 {
		return errors.New("Record not found")
	}
	record := d.records[index]
	bytes := make([]byte, record.Length)
	_, err := d.file.ReadAt(bytes, record.Offset)
	if err != nil {
		return err
	}
	err = FromBytes(bytes, output)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) DeleteById(id int64) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	index := d.findRecordIndex(id)
	if index == -1 {
		return errors.New("Record not found")
	}
	d.records = append(d.records[:index], d.records[index + 1:]...)
	d.saveState()
	return nil
}

func (d *Database) UpdateById(id int64, input interface{}) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	index := d.findRecordIndex(id)
	if index == -1 {
		return errors.New("Record not found")
	}
	bytes, err := ToBytes(input)
	if err != nil {
		return err
	}
	offset, err := d.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	length, err := d.file.WriteAt(bytes, offset)
	if err != nil {
		return err
	}
	record := &d.records[index]
	record.Offset = offset
	record.Length = int64(length)
	d.saveState()
	return nil
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
	// user := User{"Jan", "Kowalski", 25, true}
	// record, _ := db.Insert(&user)
	// fmt.Println(record)

	// db.DeleteById(1)

	// db.UpdateById(1, &User{"Jan", "Nowak", 30, false})

	user := User{}
	err := db.FindById(1, &user)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(user)
}
