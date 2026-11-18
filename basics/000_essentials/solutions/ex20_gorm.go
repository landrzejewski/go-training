package solutions

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// --- 20.1 A model and its tags ---

// Article is a GORM model. From it GORM derives the table "articles" with the columns id,
// title, slug, body, published_at, created_at and updated_at, and a unique index on slug.
type Article struct {
	ID          int64      // a field named ID is the primary key
	Title       string     `gorm:"size:200;not null"`
	Slug        string     `gorm:"size:200;not null;uniqueIndex"`
	Body        string     // no tag: a text column named body
	PublishedAt *time.Time // a pointer, because the column may be NULL (a draft)
	WordCount   int        `gorm:"-"` // computed when needed, never stored
	CreatedAt   time.Time  // filled in by GORM
	UpdatedAt   time.Time  // filled in by GORM
}

// DescribeModel reports what GORM derives from a model, without any database: the table
// name, the column names in field order, and the sorted names of the unique indexes.
func DescribeModel(model any) (table string, columns, uniqueIndexes []string, err error) {
	s, err := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		return "", nil, nil, err
	}
	for _, ix := range s.ParseIndexes() {
		if ix.Class == "UNIQUE" {
			uniqueIndexes = append(uniqueIndexes, ix.Name)
		}
	}
	slices.Sort(uniqueIndexes)
	return s.Table, s.DBNames, uniqueIndexes, nil
}

// --- 20.2 The product store on GORM ---

// productRecord is the GORM model behind GormProductStore. The domain type Product stays free
// of gorm tags, and GORM never leaks out of the store.
type productRecord struct {
	ID    int64  `gorm:"primaryKey"`
	Name  string `gorm:"not null;uniqueIndex"`
	Price int64  `gorm:"not null"`
}

func (productRecord) TableName() string { return "ex20_products" }

// GormProductStore is ProductStore (19.1) on GORM. The *gorm.DB must be opened with
// TranslateError: true, so that a used name comes back as gorm.ErrDuplicatedKey.
type GormProductStore struct{ db *gorm.DB }

func NewGormProductStore(db *gorm.DB) *GormProductStore { return &GormProductStore{db: db} }

// Migrate creates the table, or adds what is missing to it.
func (s *GormProductStore) Migrate(ctx context.Context) error {
	return s.db.WithContext(ctx).AutoMigrate(&productRecord{})
}

func (s *GormProductStore) Add(ctx context.Context, p Product) (Product, error) {
	rec := productRecord{Name: p.Name, Price: p.Price}
	err := s.db.WithContext(ctx).Create(&rec).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return Product{}, fmt.Errorf("add %q: %w", p.Name, ErrProductExists)
	}
	if err != nil {
		return Product{}, fmt.Errorf("add %q: %w", p.Name, err)
	}
	return Product{ID: rec.ID, Name: rec.Name, Price: rec.Price}, nil
}

func (s *GormProductStore) ByID(ctx context.Context, id int64) (Product, error) {
	var rec productRecord
	err := s.db.WithContext(ctx).First(&rec, id).Error // First, not Find: no row is an error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Product{}, fmt.Errorf("product %d: %w", id, ErrProductNotFound)
	}
	if err != nil {
		return Product{}, fmt.Errorf("product %d: %w", id, err)
	}
	return Product{ID: rec.ID, Name: rec.Name, Price: rec.Price}, nil
}

func (s *GormProductStore) Cheaper(ctx context.Context, maxPrice int64) ([]Product, error) {
	var recs []productRecord
	err := s.db.WithContext(ctx).Where("price <= ?", maxPrice).Order("price, id").Find(&recs).Error
	if err != nil {
		return nil, fmt.Errorf("cheaper than %d: %w", maxPrice, err)
	}
	out := make([]Product, len(recs))
	for i, r := range recs {
		out[i] = Product{ID: r.ID, Name: r.Name, Price: r.Price}
	}
	return out, nil
}

// --- 20.3 Associations, a transaction and Preload ---

// Author has many Novels: Novel.AuthorID is the foreign key GORM infers.
type Author struct {
	ID     int64
	Name   string `gorm:"size:100;not null;uniqueIndex"`
	Novels []Novel
}

func (Author) TableName() string { return "ex20_authors" }

// Novel belongs to an Author.
type Novel struct {
	ID       int64
	AuthorID int64  `gorm:"not null;index"`
	Title    string `gorm:"size:200;not null"`
}

func (Novel) TableName() string { return "ex20_novels" }

// ErrEmptyTitle is returned by CreateAuthorWithNovels for a blank title.
var ErrEmptyTitle = errors.New("empty title")

// CreateAuthorWithNovels stores an author and their novels in ONE transaction. If any title
// is blank, nothing at all is stored - not even the author, who was inserted first.
func CreateAuthorWithNovels(ctx context.Context, db *gorm.DB, name string, titles ...string) (Author, error) {
	author := Author{Name: name}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Inside the transaction use tx, never db: db is another connection.
		if err := tx.Create(&author).Error; err != nil {
			return err
		}
		for _, title := range titles {
			if strings.TrimSpace(title) == "" {
				return fmt.Errorf("author %q: %w", name, ErrEmptyTitle) // an error -> rollback
			}
			novel := Novel{AuthorID: author.ID, Title: title}
			if err := tx.Create(&novel).Error; err != nil {
				return err
			}
			author.Novels = append(author.Novels, novel)
		}
		return nil // nil -> commit
	})
	if err != nil {
		return Author{}, err
	}
	return author, nil
}

// AuthorsWithNovels returns every author sorted by name, each with their novels sorted by
// title. It needs two queries however many authors there are: one for the authors and one,
// from Preload, for all their novels - no N+1.
func AuthorsWithNovels(ctx context.Context, db *gorm.DB) ([]Author, error) {
	var authors []Author
	err := db.WithContext(ctx).
		Preload("Novels", func(tx *gorm.DB) *gorm.DB { return tx.Order("title") }).
		Order("name").
		Find(&authors).Error
	return authors, err
}
