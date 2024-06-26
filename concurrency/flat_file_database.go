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
	id [idSize]byte
	firstName [firstNameSize]byte
	lastName [lastNameSize]byte
	age [ageSize]byte
}

type database struct {
	file *os.File
	mutex sync.Mutex
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

func (d *database) append(record *userRecord) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	offset, err := d.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	return d.write(record, offset)
}

func newDatabase(filePath string) (*database, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	} 
	return &database{
		file,
		sync.Mutex{},
	}, nil
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

	var firstNameBytes [firstNameSize]byte 
	copy(firstNameBytes[:], "Jan")

	var lastNameBytes [lastNameSize]byte 
	copy(lastNameBytes[:], "Nowak")

	record := userRecord{
		id: toByteArray(1),
		firstName: firstNameBytes,
		lastName: lastNameBytes,
		age: toByteArray(30),
	}

	err = db.append(&record)
	if err != nil {
		fmt.Println(err)
	}

}
