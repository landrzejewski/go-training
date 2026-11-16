package db

import (
	"errors"
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T, path string) *Database {
	t.Helper()
	db, err := Open(path, &Sequence{})
	if err != nil {
		t.Fatalf("Open(%q): %v", path, err)
	}
	return db
}

func TestCRUD(t *testing.T) {
	db := openTestDB(t, filepath.Join(t.TempDir(), "test.db"))
	defer db.Close()

	user := User{"Jan", "Kowalski", 25, true}
	record, err := db.Create(&user)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if record.ID != 1 {
		t.Errorf("first id = %d, want 1", record.ID)
	}

	var loaded User
	if _, err := db.Read(record.ID, &loaded); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if loaded != user {
		t.Errorf("Read = %+v, want %+v", loaded, user)
	}

	user.IsActive = false
	updated, err := db.Update(record.ID, &user)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Offset <= record.Offset {
		t.Errorf("Update should append: offset %d, previous %d", updated.Offset, record.Offset)
	}
	// Decode into a FRESH value: gob omits zero-valued fields, and decoding into
	// an already populated struct leaves such fields at their previous value -
	// reusing `loaded` here would still show IsActive == true.
	var reloaded User
	if _, err := db.Read(record.ID, &reloaded); err != nil {
		t.Fatalf("Read after Update: %v", err)
	}
	if reloaded.IsActive {
		t.Error("Read after Update returned the old value")
	}

	if err := db.Delete(record.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := db.Read(record.ID, &reloaded); !errors.Is(err, ErrNotFound) {
		t.Errorf("Read after Delete: err = %v, want ErrNotFound", err)
	}
}

func TestNotFound(t *testing.T) {
	db := openTestDB(t, filepath.Join(t.TempDir(), "test.db"))
	defer db.Close()

	var user User
	if _, err := db.Read(42, &user); !errors.Is(err, ErrNotFound) {
		t.Errorf("Read: err = %v, want ErrNotFound", err)
	}
	if _, err := db.Update(42, &user); !errors.Is(err, ErrNotFound) {
		t.Errorf("Update: err = %v, want ErrNotFound", err)
	}
	if err := db.Delete(42); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete: err = %v, want ErrNotFound", err)
	}
}

func TestReopenReloadsIndexAndContinuesIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	db := openTestDB(t, path)
	first, err := db.Create(&User{FirstName: "Anna"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	second, err := db.Create(&User{FirstName: "Piotr"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	db = openTestDB(t, path)
	defer db.Close()

	var user User
	if _, err := db.Read(first.ID, &user); err != nil {
		t.Fatalf("Read after reopen: %v", err)
	}
	if user.FirstName != "Anna" {
		t.Errorf("Read after reopen = %q, want Anna", user.FirstName)
	}

	third, err := db.Create(&User{FirstName: "Ewa"})
	if err != nil {
		t.Fatalf("Create after reopen: %v", err)
	}
	if third.ID != second.ID+1 {
		t.Errorf("id after reopen = %d, want %d", third.ID, second.ID+1)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	db := openTestDB(t, filepath.Join(t.TempDir(), "test.db"))
	if err := db.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
