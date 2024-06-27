/*
Napisz prostą bazę danych, która przechowuje dane w formie binarnej, korzystając z dostępu losowego. Poniżej
podano przykład odczytu danych z określonej pozycji w pliku. Zapis powinien być zaimplementowany na podobnej zasadzie.
Baza danych powinna umożliwiać następujące operacje: odczyt rekordu według identyfikatora, aktualizacja rekordu,
usunięcie rekordu, dodanie rekordu. Aby zoptymalizować wydajność bazy danych, wprowadź indeksowanie pozycji
rekordu i prostą pamięć podręczną za pomocą HashMap. Pomyśl o optymalnym sposobie usuwania rekordów i
ponownego wykorzystania tego obszaru pliku. Rekordy powinny mieć stałą długość poszczególnych pól, a ich
definicja powinna znajdować się w sekcji nagłówka pliku.
*/

package concurrency

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

const (
	idSize = 8
	firstNameSize = 128
	lastNameSize = 128
	ageSize = 8
	recordSize = idSize + firstNameSize + lastNameSize + ageSize
)

type userRecord struct {
	Id [idSize]byte
	FirstName [firstNameSize]byte
	LastName [lastNameSize]byte
	Age [ageSize]byte
}

func (u *userRecord) getId() int64 {
	var id int64
	binary.Read(bytes.NewReader(u.Id[:]), binary.LittleEndian, &id)
	return id
}

func (u *userRecord) getFirstName() string {
	return string(u.FirstName[:])
}

func (u *userRecord) getLastName() string {
	return string(u.LastName[:])
}

func (u *userRecord) getAge() int64 {
	var age int64
	binary.Read(bytes.NewReader(u.Age[:]), binary.LittleEndian, &age)
	return age
}

func newUserRecord(id int64, firstName, lastName string, age int64) *userRecord {
	var firstNameBytes [firstNameSize]byte 
	copy(firstNameBytes[:], firstName)

	var lastNameBytes [lastNameSize]byte 
	copy(lastNameBytes[:], lastName)

	return &userRecord{
		Id: toByteArray(id),
		FirstName: firstNameBytes,
		LastName: lastNameBytes,
		Age: toByteArray(age),
	}
}

type database struct {
	file *os.File
	mutex sync.Mutex
	index map[int64]int64
	cache map[int64]*userRecord
}

func (d *database) close() error {
	return d.file.Close()
}

func (d * database) write(record *userRecord, offset int64) error {
	buffer := new(bytes.Buffer)
	err := binary.Write(buffer, binary.LittleEndian, record)
	if err != nil {
		return err
	}
	_, err = d.file.WriteAt( buffer.Bytes(), offset)
	if err != nil {
		return err
	}
	return nil
}

func (d *database) read(offset int64) (*userRecord, error) {
	buffer := make([]byte, recordSize)
	_, err := d.file.ReadAt(buffer, offset)
	if err != nil {
		return nil, err
	}

	record := userRecord{}
	reader := bytes.NewReader(buffer)
	err = binary.Read(reader, binary.LittleEndian, &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (d *database) loadIndex() error {
	stats, err := d.file.Stat()
	if err != nil {
		return err
	}
	if stats.Size() == 0 {
		return nil
	}

	for offset := int64(0); offset < stats.Size(); offset += recordSize {
		record, err := d.read(offset)
		if err != nil {
			return err
		}
		id := record.getId()
		d.index[id] = offset
	}
	return nil
}

func (d *database) appendRecord(record *userRecord) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	offset, err := d.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	err = d.write(record, offset)
	if err != nil {
		return err
	}
	d.index[record.getId()] = offset
	return nil
}

func (d *database) readRecord(id int64) (*userRecord, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if record, found := d.cache[id]; found {
		fmt.Println("Reading from cache")
		return record, nil
	}

	offset, found := d.index[id]
	if !found {
		return nil, fmt.Errorf("Record not found")
	}

	record, err := d.read(offset)
	if err != nil {
		return nil, err
	}

	d.cache[id] = record

	return record, nil
}

func (d *database) UpdateRecord(record *userRecord) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	offset, found := d.index[record.getId()]
	if !found {
		return fmt.Errorf("record not found")
	}

	d.cache[record.getId()] = record

	return d.write(record, offset)
}

func newDatabase(filePath string) (*database, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	} 
	db := &database{
		file,
		sync.Mutex{},
		make(map[int64]int64),
		make(map[int64]*userRecord),
	}

	err = db.loadIndex() 
	if err != nil {
		return nil, err
	} 

	return db, nil
}

func toByteArray(i int64) (arr [8]byte) {
    binary.LittleEndian.PutUint64(arr[0:8], uint64(i))
    return
}

func Run() {
	db, err := newDatabase("users.db")
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	defer db.close()

	// db.appendRecord(newUserRecord(3, "Marek", "Nowak", 40))
	record, _ := db.readRecord(3)
	fmt.Println(record.getId(), record.getFirstName(), record.getLastName(), record.getAge())
	record, _ = db.readRecord(3)
	fmt.Println(record.getId(), record.getFirstName(), record.getLastName(), record.getAge())
	
	db.UpdateRecord(newUserRecord(3, "Michał", "Nowak", 40))
	record, _ = db.readRecord(3)
	fmt.Println(record.getId(), record.getFirstName(), record.getLastName(), record.getAge())
}
