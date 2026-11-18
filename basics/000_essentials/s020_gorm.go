package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================================
// 020 - GORM
// ============================================================================================
//
// What you build: the users table of 019 once more, now with GORM - the most widely used Go
// ORM (object-relational mapper). An ORM writes the SQL for you, from Go structs.
//
//	1. models            a struct and its tags describe a table; AutoMigrate creates it
//	2. CRUD and traps    create, read, update, delete - and three mistakes everybody makes
//	3. the N+1 problem   loading related rows: one query per row, or two queries in total
//	4. transactions      db.Transaction commits or rolls back for you
//	5. a repository      UserRepository a third time; the output matches 019
//
// The same lookup both ways:
//
//	database/sql:  db.QueryRowContext(ctx, `select id, name, email from users where id = $1`, id).
//	                   Scan(&u.ID, &u.Name, &u.Email)
//	GORM:          db.WithContext(ctx).First(&u, id)
//
// GORM saves the scanning code and the boring CRUD. The price: the SQL is hidden (db.Debug()
// prints it), a typo such as Where("nmae = ?") still compiles, and the N+1 problem (step 3) is
// one forgotten Preload away. A common choice: GORM for plain CRUD, and db.Raw or database/sql
// for the queries that matter. Like 019, every step after the first needs PostgreSQL.

// gormBasics runs the steps of section 020.
func gormBasics() {
	fmt.Println("--- 020 GORM ---")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := gormModelsDemo(ctx) // 1. models - nil when PostgreSQL is not reachable
	if db != nil {
		if sqlDB, err := db.DB(); err == nil {
			defer sqlDB.Close()
		}
	}
	gormCRUDDemo(ctx, db)        // 2. CRUD and traps
	gormNPlusOneDemo(ctx, db)    // 3. the N+1 problem
	gormTransactionDemo(ctx, db) // 4. transactions
	gormRepositoryDemo(ctx, db)  // 5. a repository

	// More: basics/020_gorm
}

// --------------------------------------------------------------------------------------------
// Step 1 - models
// --------------------------------------------------------------------------------------------

// gormModelsDemo is step 1: connect, then let AutoMigrate create the tables from the models.
func gormModelsDemo(ctx context.Context) *gorm.DB {
	step("1. models: structs describe the tables, AutoMigrate creates them")
	db, err := openGorm()
	if err != nil {
		printNoDatabase(err) // from 019
		return nil
	}
	if err := resetGormTables(ctx, db); err != nil {
		show("AutoMigrate failed: %s", oneLine(err))
		return nil
	}
	show("AutoMigrate created the tables %s and %s", userRow{}.TableName(), orderRow{}.TableName())
	return db
}

// userRow is the GORM MODEL behind the users table: GORM reads the struct to know the table.
// A field named ID is the primary key, CreatedAt and UpdatedAt are filled in automatically,
// and a DeletedAt field turns Delete into a SOFT delete (step 2). Tags fine-tune the columns.
// A tag is just a string (007), so a typo such as "uniqeIndex" is silently ignored.
type userRow struct {
	ID        int64          `gorm:"primaryKey"`
	Name      string         `gorm:"size:100;not null"`
	Email     string         `gorm:"size:100;not null;uniqueIndex"`
	CreatedAt time.Time      // set on Create
	UpdatedAt time.Time      // set on every update
	DeletedAt gorm.DeletedAt `gorm:"index"`             // soft delete
	Orders    []orderRow     `gorm:"foreignKey:UserID"` // "has many": orderRow.UserID points here
}

// TableName overrides the name GORM would derive: "user_rows" (snake_case, plural).
func (userRow) TableName() string { return "e020_users" }

// orderRow is one order of a user (step 3).
type orderRow struct {
	ID     int64 `gorm:"primaryKey"`
	UserID int64 `gorm:"not null;index"`
	Item   string
}

func (orderRow) TableName() string { return "e020_orders" }

// openGorm opens GORM on PostgreSQL. Unlike sql.Open, gorm.Open connects and pings at once, so
// a database that is down is an error right here.
func openGorm() (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn()), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent), // logger.Info prints every SQL statement
		TranslateError: true,                                  // a unique violation -> gorm.ErrDuplicatedKey
	})
	if err != nil {
		return nil, err
	}
	// GORM sits on database/sql: db.DB() is the *sql.DB, to tune the pool (019) and to close it.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

// resetGormTables drops and recreates the tables, so that every run prints the same output.
// AutoMigrate creates tables and adds missing columns and indexes - fine for development. It
// never drops or renames anything, so production uses versioned migrations (goose,
// golang-migrate, atlas).
func resetGormTables(ctx context.Context, db *gorm.DB) error {
	tx := db.WithContext(ctx)
	if err := tx.Migrator().DropTable(&orderRow{}, &userRow{}); err != nil {
		return err
	}
	return tx.AutoMigrate(&userRow{}, &orderRow{})
}

// --------------------------------------------------------------------------------------------
// Step 2 - CRUD and traps
// --------------------------------------------------------------------------------------------

// gormCRUDDemo is step 2: create, read, update and delete - with the traps in the order people
// hit them.
func gormCRUDDemo(ctx context.Context, db *gorm.DB) {
	step("2. CRUD and traps: .Error, First vs Find, Updates, soft delete")
	if db == nil {
		show("(skipped: no database)")
		return
	}
	// WithContext is how a context reaches the query. It also gives a fresh "session" that is
	// safe to reuse for many statements.
	tx := db.WithContext(ctx)

	// Every method returns *gorm.DB, and the error is in its .Error field - not a second result,
	// so it is easy to forget. End every chain with .Error.
	ada := userRow{Name: "Ada", Email: "ada@example.com"}
	err := tx.Create(&ada).Error // INSERT; fills in ID, CreatedAt and UpdatedAt
	show("Create -> id=%d createdAt set=%t err=%v", ada.ID, !ada.CreatedAt.IsZero(), err)

	// Trap 1: First wants a row and returns gorm.ErrRecordNotFound; Find fills a slice, and an
	// empty result is NOT an error. Mixing the two up is the most common GORM bug.
	var one userRow
	err = tx.First(&one, "email = ?", "nobody@example.com").Error
	show("First, no rows -> errors.Is(err, gorm.ErrRecordNotFound): %t", errors.Is(err, gorm.ErrRecordNotFound))
	var many []userRow
	res := tx.Where("email = ?", "nobody@example.com").Find(&many)
	show("Find,  no rows -> err=%v RowsAffected=%d", res.Error, res.RowsAffected)

	// Trap 2: Updates with a STRUCT skips zero values - GORM cannot tell "set to empty" from
	// "not set" (001). A map says exactly which columns to write.
	_ = tx.Model(&ada).Updates(userRow{Name: ""}).Error
	show("Updates(userRow{Name: \"\"})  -> name is still %q", nameOf(tx, ada.ID))
	_ = tx.Model(&ada).Updates(map[string]any{"name": ""}).Error
	show("Updates(map{\"name\": \"\"})    -> name is now %q", nameOf(tx, ada.ID))

	// Trap 3: soft delete. With a DeletedAt field, Delete only fills in deleted_at, and EVERY
	// later query adds "WHERE deleted_at IS NULL" by itself. Unscoped() sees the deleted rows.
	// The unique index still covers them, so ada@example.com cannot be used again.
	_ = tx.Delete(&ada).Error
	var visible, all int64
	_ = tx.Model(&userRow{}).Count(&visible).Error
	_ = tx.Unscoped().Model(&userRow{}).Count(&all).Error
	show("after Delete: Count=%d, Unscoped().Count=%d", visible, all)
}

func nameOf(tx *gorm.DB, id int64) string {
	var u userRow
	_ = tx.First(&u, id).Error
	return u.Name
}

// --------------------------------------------------------------------------------------------
// Step 3 - the N+1 problem
// --------------------------------------------------------------------------------------------

// gormNPlusOneDemo is step 3. Loading users AND their orders naively takes one query for the
// users plus one query PER USER - "N+1" queries. It is fast with 3 users in a test and slow
// with 10 000 in production, and it never fails. Preload needs two queries, whatever N is.
func gormNPlusOneDemo(ctx context.Context, db *gorm.DB) {
	step("3. the N+1 problem: a query per user, or Preload")
	if db == nil {
		show("(skipped: no database)")
		return
	}
	tx := db.WithContext(ctx)
	for _, name := range []string{"Alan", "Grace", "Linus"} {
		// Creating a user with Orders inserts the orders too, with UserID filled in.
		u := userRow{Name: name, Email: strings.ToLower(name) + "@example.com",
			Orders: []orderRow{{Item: "book"}, {Item: "pen"}}}
		if err := tx.Create(&u).Error; err != nil {
			show("create failed: %s", oneLine(err))
			return
		}
	}

	// Count the SELECTs with a callback. Callbacks belong to db and every session made from
	// it, so this one is removed again at the end.
	queries := 0
	_ = db.Callback().Query().After("gorm:query").Register("count_queries", func(*gorm.DB) { queries++ })
	defer db.Callback().Query().Remove("count_queries")

	// The naive way: the users, then the orders of each user in a loop.
	var users []userRow
	_ = tx.Find(&users).Error
	for i := range users {
		_ = tx.Where("user_id = ?", users[i].ID).Find(&users[i].Orders).Error
	}
	loop := queries

	// Preload loads the orders of ALL the users in one more query: WHERE user_id IN (...).
	queries = 0
	var preloaded []userRow
	_ = tx.Preload("Orders").Find(&preloaded).Error
	show("%d users with their orders: a loop needs %d queries (1+N), Preload needs %d",
		len(preloaded), loop, queries)

	// When the chain gets in the way, write the SQL yourself and Scan it into a struct.
	var stats struct{ Users, Orders int64 }
	_ = tx.Raw(`select count(distinct u.id) as users, count(o.id) as orders
	            from e020_users u join e020_orders o on o.user_id = u.id
	            where u.deleted_at is null`).Scan(&stats).Error
	show("db.Raw -> %+v", stats)
}

// --------------------------------------------------------------------------------------------
// Step 4 - transactions
// --------------------------------------------------------------------------------------------

// gormTransactionDemo is step 4: the managed transaction. db.Transaction commits when the
// function returns nil, and rolls back when it returns an error or panics - there is no
// Rollback to forget (compare 019, step 4).
func gormTransactionDemo(ctx context.Context, db *gorm.DB) {
	step("4. transactions: db.Transaction commits or rolls back for you")
	if db == nil {
		show("(skipped: no database)")
		return
	}
	tx := db.WithContext(ctx)
	var before, after int64
	_ = tx.Model(&userRow{}).Count(&before).Error

	// Inside, use the t it gives you, NEVER db: db is another connection, outside the
	// transaction.
	err := tx.Transaction(func(t *gorm.DB) error {
		if err := t.Create(&userRow{Name: "Tx", Email: "tx@example.com"}).Error; err != nil {
			return err
		}
		return t.Create(&userRow{Name: "Tx copy", Email: "tx@example.com"}).Error // a duplicate
	})
	_ = tx.Model(&userRow{}).Count(&after).Error
	show("Transaction with a duplicate -> errors.Is(err, gorm.ErrDuplicatedKey)=%t, users %d -> %d",
		errors.Is(err, gorm.ErrDuplicatedKey), before, after)
}

// --------------------------------------------------------------------------------------------
// Step 5 - a repository
// --------------------------------------------------------------------------------------------

// gormRepositoryDemo is step 5: UserRepository a third time, run through the contract of 019.
func gormRepositoryDemo(ctx context.Context, db *gorm.DB) {
	step("5. a repository: the contract of 019, on GORM")
	if db == nil {
		show("(skipped: no database - 019 ran the same contract in memory)")
		return
	}
	if err := resetGormTables(ctx, db); err != nil {
		show("setup failed: %s", oneLine(err))
		return
	}
	exerciseRepo(ctx, NewGormUserRepo(db), "GormUserRepo (the same output as 019)")
}

// GormUserRepo is UserRepository on GORM. It converts between the domain type User and the
// model userRow, so GORM never leaks out of the repository.
type GormUserRepo struct{ db *gorm.DB }

// The compiler checks that GormUserRepo satisfies UserRepository (008).
var _ UserRepository = (*GormUserRepo)(nil)

func NewGormUserRepo(db *gorm.DB) *GormUserRepo { return &GormUserRepo{db: db} }

func (r *GormUserRepo) Create(ctx context.Context, u *User) error {
	row := userRow{Name: u.Name, Email: u.Email}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return translateGormError(err)
	}
	u.ID = row.ID
	return nil
}

func (r *GormUserRepo) ByID(ctx context.Context, id int64) (User, error) {
	var row userRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return User{}, translateGormError(err)
	}
	return User{ID: row.ID, Name: row.Name, Email: row.Email}, nil
}

func (r *GormUserRepo) List(ctx context.Context) ([]User, error) {
	var rows []userRow
	if err := r.db.WithContext(ctx).Order("id").Find(&rows).Error; err != nil {
		return nil, translateGormError(err)
	}
	users := make([]User, len(rows))
	for i, row := range rows {
		users[i] = User{ID: row.ID, Name: row.Name, Email: row.Email}
	}
	return users, nil
}

func (r *GormUserRepo) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&userRow{}, id)
	if res.Error != nil {
		return translateGormError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound // like Find, deleting nothing is no error to GORM
	}
	return nil
}

// translateGormError turns GORM's errors into the domain errors of 018.
func translateGormError(err error) error {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey): // needs TranslateError: true in gorm.Config
		return ErrDuplicate
	}
	return err
}
