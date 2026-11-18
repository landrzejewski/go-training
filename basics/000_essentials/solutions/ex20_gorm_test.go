package solutions

// ExampleDescribeModel needs no database. The other tests need the PostgreSQL from
// docker-compose.yml and are SKIPPED without it. They use testDSN and
// testProductStoreContract from ex19_database_sql_test.go, so copy that file too.

import (
	"errors"
	"fmt"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ExampleDescribeModel() {
	table, columns, unique, err := DescribeModel(&Article{})
	fmt.Println(table)
	fmt.Println(columns)
	fmt.Println(unique, err)
	// Output:
	// articles
	// [id title slug body published_at created_at updated_at]
	// [idx_articles_slug] <nil>
}

func TestGormProductStore(t *testing.T) {
	db := openTestGorm(t)
	if err := db.WithContext(t.Context()).Migrator().DropTable("ex20_products"); err != nil {
		t.Fatal(err)
	}
	store := NewGormProductStore(db)
	if err := store.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	testProductStoreContract(t, store)
}

func TestAuthorsWithNovels(t *testing.T) {
	db := openTestGorm(t)
	ctx := t.Context()
	if err := db.WithContext(ctx).Migrator().DropTable(&Novel{}, &Author{}); err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(ctx).AutoMigrate(&Author{}, &Novel{}); err != nil {
		t.Fatal(err)
	}

	herbert, err := CreateAuthorWithNovels(ctx, db, "Herbert", "Dune", "Children of Dune")
	if err != nil || herbert.ID == 0 || len(herbert.Novels) != 2 {
		t.Fatalf("CreateAuthorWithNovels = %+v, %v; want an author with 2 novels", herbert, err)
	}
	if _, err := CreateAuthorWithNovels(ctx, db, "Le Guin", "The Dispossessed", " "); !errors.Is(err, ErrEmptyTitle) {
		t.Errorf("a blank title: err = %v, want ErrEmptyTitle", err)
	}
	if _, err := CreateAuthorWithNovels(ctx, db, "Asimov", "Foundation"); err != nil {
		t.Fatal(err)
	}

	// Count the SELECT statements, to prove there is no N+1.
	queries := 0
	if err := db.Callback().Query().After("gorm:query").Register("ex20_count", func(*gorm.DB) { queries++ }); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove("ex20_count") })

	authors, err := AuthorsWithNovels(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	got := fmt.Sprint(summary(authors))
	want := "[Asimov:[Foundation] Herbert:[Children of Dune Dune]]" // Le Guin was rolled back
	if got != want {
		t.Errorf("AuthorsWithNovels = %s, want %s", got, want)
	}
	if queries != 2 {
		t.Errorf("AuthorsWithNovels ran %d queries, want 2 (one for authors, one Preload)", queries)
	}
}

func summary(authors []Author) []string {
	out := make([]string, len(authors))
	for i, a := range authors {
		titles := make([]string, len(a.Novels))
		for j, n := range a.Novels {
			titles[j] = n.Title
		}
		out[i] = fmt.Sprintf("%s:%v", a.Name, titles)
	}
	return out
}

// openTestGorm opens GORM with TranslateError, or SKIPS the test without a database.
func openTestGorm(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open(testDSN()), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Skipf("PostgreSQL is not reachable (docker-compose up -d): %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return db
}
