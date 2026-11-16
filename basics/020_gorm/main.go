// Command 020_gorm is module 020 of the Go course: GORM, the most widely used Go ORM.
//
// Module 019 wrote SQL by hand; GORM generates it from Go structs. This module ends by
// implementing module 019's repository interface on GORM, so the two are directly comparable.
//
// It needs the PostgreSQL from the repository's docker-compose.yml. The DSN environment variable
// overrides the default connection string (shared with module 019). If the database is
// unreachable the module says so and runs only the parts that need no database.
//
//	docker-compose up -d
//	go run ./basics/020_gorm
//	DSN='postgres://admin:admin@localhost:5433/users?sslmode=disable' go run ./basics/020_gorm
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db := connectOrExplain(ctx) // nil when PostgreSQL is unreachable: live parts are skipped
	if db != nil {
		if sqlDB, err := db.DB(); err == nil {
			defer sqlDB.Close()
		}
	}

	ormOrSQL()                      // what GORM gives, what it costs, when to use which
	modelsAndTags()                 // naming conventions and gorm tags, parsed without a database
	connectingAndMigrating(ctx, db) // gorm.Config, tuning the pool via db.DB(), AutoMigrate
	crud(ctx, db)                   // First vs Find, Updates skips zero values, soft deletes
	queryingAndPreload(ctx, db)     // chaining, the N+1 problem counted, Preload, Joins, Raw
	associations(ctx, db)           // nested create, many2many, the Association API
	transactionsAndHooks(ctx, db)   // db.Transaction, rollback on error and panic, BeforeCreate
	repositoryOnGorm(ctx, db)       // module 019's interface, a third implementation
}

// The connection settings mirror module 019: same default, same DSN override, same tuning.
const (
	defaultDSN  = "postgres://admin:admin@localhost:5432/users?sslmode=disable"
	pingTimeout = 2 * time.Second
)

func dsn() string {
	if v, ok := os.LookupEnv("DSN"); ok && v != "" {
		return v
	}
	return defaultDSN
}

// connectOrExplain opens GORM and verifies the pool, or explains why it could not and returns nil.
func connectOrExplain(ctx context.Context) *gorm.DB {
	db, err := connect(ctx)
	if err != nil {
		fmt.Println("NOTE: PostgreSQL is not reachable, so the live parts are skipped.")
		fmt.Println("      " + shorten(err))
		fmt.Println("      Start it with:  docker-compose up -d")
		fmt.Println("      DSN: " + dsn() + "  (set the DSN env var to point elsewhere)")
		fmt.Println()
		return nil
	}
	fmt.Println("PostgreSQL is reachable; running every topic against it.")
	fmt.Println()
	return db
}

func connect(ctx context.Context) (*gorm.DB, error) {
	// gorm.Open returns a *gorm.DB - a statement builder wrapping a *sql.DB, NOT a connection.
	// gorm.Config matters more than it looks:
	db, err := gorm.Open(postgres.Open(dsn()), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent), // default: slow queries+errors to stdout
		PrepareStmt:            true,                                  // cache prepared statements
		SkipDefaultTransaction: true,                                  // no implicit tx around every write
		// NamingStrategy overrides the pluralisation and casing rules.
	})
	if err != nil {
		return nil, err
	}
	// GORM does NOT tune the pool: reach the underlying *sql.DB and do it yourself (module 019).
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25) // module 019's tuning, unchanged
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// shorten puts a (multi-line) driver error on one short line, so the output stays readable.
func shorten(err error) string {
	if err == nil {
		return "<nil>"
	}
	line := strings.Join(strings.Fields(err.Error()), " ")
	if len(line) > 110 {
		line = line[:110] + "..."
	}
	return line
}

// ormOrSQL weighs GORM against hand-written database/sql.
func ormOrSQL() {
	fmt.Println("--- ORM or database/sql? ---")

	// GORM gives: no scanning code kept in sync with the SELECT list, schema migration from
	// structs, associations, hooks, soft deletes/timestamps/optimistic locking, and portability
	// across PostgreSQL, MySQL, SQLite and SQL Server.
	// It costs: IMPLICIT SQL (a harmless-looking chain can emit a Cartesian join - read it with
	// db.Debug()), the N+1 trap one forgotten Preload away (slow, never failing), reflection that
	// moves errors to run time (Where("nmae = ?", x) compiles), and another layer to learn.
	fmt.Println("database/sql: err := db.QueryRowContext(ctx,")
	fmt.Println("                  `select id, name, email from users where id = $1`, id).")
	fmt.Println("                  Scan(&u.ID, &u.Name, &u.Email)")
	fmt.Println("GORM:         err := db.WithContext(ctx).First(&u, id).Error")
	fmt.Println("the second is shorter; the first is the one whose cost you can predict")

	// database/sql when the SQL is the interesting part: reporting, analytics, window functions,
	// CTEs, precise control of the plan. GORM for boring CRUD over many similar entities.
	// The common middle ground: GORM for CRUD, db.Raw for the three queries that matter.
	// sqlc (type-safe Go generated from SQL) and sqlx (a thin database/sql extension) approach
	// that middle ground from the other side.
}

// modelsAndTags shows GORM's conventions by parsing the models - no database needed.
func modelsAndTags() {
	fmt.Println("\n--- models, tags and conventions ---")

	// Convention over configuration: the table is the snake_case PLURAL of the struct name,
	// columns are snake_case field names, a field named ID is the primary key (auto-increment
	// if an integer), CreatedAt/UpdatedAt are maintained, DeletedAt enables soft delete.
	s, err := schema.Parse(&userModel{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		fmt.Println("parse failed:", err)
		return
	}
	fmt.Printf("derived table name for userModel: %q, TableName() overrides it to %q\n",
		schema.NamingStrategy{}.TableName("userModel"), s.Table)
	fmt.Printf("primary key %q, columns %v\n", s.PrioritizedPrimaryField.DBName, s.DBNames)
	for _, ix := range s.ParseIndexes() {
		fmt.Println("index", strings.TrimSpace(ix.Name+" "+ix.Class))
	}
	// Orders, Tags (associations), Secret (gorm:"-") and internal (unexported - invisible to
	// every reflective library) have no column.

	// A tag is a STRING, so a typo is silent (module 007): no index, no error.
	t, _ := schema.Parse(&typoModel{}, &sync.Map{}, schema.NamingStrategy{})
	fmt.Printf("gorm:\"uniqeIndex\" (typo) -> table %q with %d indexes, and no complaint\n",
		t.Table, len(t.ParseIndexes()))

	// gorm.Model embeds ID uint, CreatedAt, UpdatedAt, DeletedAt - idiomatic, but it commits you
	// to soft deletes and a uint key, so these models declare the fields explicitly.
}

// userModel is the GORM model for module 019's user. Tag options (semicolon-separated):
// primaryKey autoIncrement size:N type:jsonb not null default:X uniqueIndex index check:...
// column:name embedded, and -> (read-only), <- (write-only), - (ignore).
type userModel struct {
	ID        int64          `gorm:"primaryKey;autoIncrement"`
	Name      string         `gorm:"size:100;not null"`
	Email     string         `gorm:"size:100;not null;uniqueIndex"`
	Nickname  *string        `gorm:"size:100"` // a pointer, so NULL is representable (module 019)
	CreatedAt time.Time      // maintained automatically
	UpdatedAt time.Time      // maintained automatically
	DeletedAt gorm.DeletedAt `gorm:"index"` // the presence of this field enables SOFT DELETE
	Orders    []orderModel   `gorm:"foreignKey:UserID"`
	Tags      []tagModel     `gorm:"many2many:m020_user_tags"` // the inverse of tagModel.Users
	Secret    string         `gorm:"-"`
	internal  string
}

// TableName overrides the derived "user_models". The m020_ prefix keeps this module's tables
// apart from module 019's m019_users in the same database.
func (userModel) TableName() string { return "m020_users" }

type orderModel struct {
	ID        int64 `gorm:"primaryKey;autoIncrement"`
	UserID    int64 `gorm:"not null;index"` // <Owner>ID: the inferred foreign key
	Item      string
	Amount    int64 `gorm:"not null;default:0;check:amount >= 0"`
	CreatedAt time.Time
}

func (orderModel) TableName() string { return "m020_orders" }

type tagModel struct {
	ID    int64       `gorm:"primaryKey;autoIncrement"`
	Label string      `gorm:"size:50;not null;uniqueIndex"`
	Users []userModel `gorm:"many2many:m020_user_tags"`
}

func (tagModel) TableName() string { return "m020_tags" }

type typoModel struct {
	ID    int64
	Email string `gorm:"uniqeIndex"`
}

// connectingAndMigrating shows AutoMigrate and what it deliberately will not do.
func connectingAndMigrating(ctx context.Context, db *gorm.DB) {
	fmt.Println("\n--- connecting and AutoMigrate ---")

	// See connect: gorm.Config, and db.DB() to tune the pool. db.WithContext(ctx) is how a
	// context reaches the query - easy to forget, and without it a cancelled request keeps querying.
	//	db := gorm.Open(postgres.Open(dsn())) // ERROR: assignment mismatch: 1 variable but gorm.Open returns 2 values
	if db == nil {
		fmt.Println("(skipped: no database)")
		return
	}

	// A clean slate keeps the module re-runnable. Schema changes run on a session with
	// PrepareStmt OFF: a statement cached before DropTable/AutoMigrate would make PostgreSQL fail
	// with "cached plan must not change result type" on the next run.
	migrate := db.Session(&gorm.Session{PrepareStmt: false, Context: ctx})
	_ = migrate.Migrator().DropTable("m020_user_tags", &orderModel{}, &tagModel{}, &userModel{})
	// AutoMigrate creates tables, adds missing columns, indexes and constraints.
	if err := migrate.AutoMigrate(&userModel{}, &orderModel{}, &tagModel{}); err != nil {
		fmt.Println("AutoMigrate failed:", err)
		return
	}
	fmt.Println("AutoMigrate created m020_users, m020_orders, m020_tags and the join table")

	migrator := db.Migrator()
	cols, _ := migrator.ColumnTypes(&userModel{})
	names := make([]string, 0, len(cols))
	for _, c := range cols {
		names = append(names, c.Name())
	}
	fmt.Println("m020_users columns:", names)
	fmt.Println("unique index on email:", migrator.HasIndex(&userModel{}, "Email"))
	sqlDB, _ := db.DB()
	st := sqlDB.Stats()
	fmt.Printf("underlying pool: maxOpen=%d open=%d idle=%d\n", st.MaxOpenConnections, st.OpenConnections, st.Idle)

	// AutoMigrate will NOT drop or rename a column, change a type destructively, or touch data
	// (backfills, splits). Great for development and the first deploy; production needs
	// versioned, reviewable migrations: golang-migrate, goose, atlas, dbmate.
}

// crud shows the basic operations and the traps, in the order people hit them.
func crud(ctx context.Context, db *gorm.DB) {
	fmt.Println("\n--- CRUD and soft deletes ---")
	if db == nil {
		fmt.Println("(skipped: no database)")
		return
	}
	tx := db.WithContext(ctx)

	// Every call returns a *gorm.DB carrying .Error and .RowsAffected - not a (value, error)
	// pair, so the error is easy to drop. Check .Error every time.
	u := userModel{Name: "Ada", Email: "ada@example.com"}
	if err := tx.Create(&u).Error; err != nil { // INSERT; fills ID, CreatedAt, UpdatedAt
		fmt.Println("create failed:", err)
		return
	}
	fmt.Printf("Create -> id=%d createdAt set=%t\n", u.ID, !u.CreatedAt.IsZero())
	_ = tx.Create(&userModel{Name: "Alan", Email: "alan@example.com"}).Error

	// Trap 1: First returns gorm.ErrRecordNotFound, Find does NOT - an empty Find is a success
	// with RowsAffected == 0. Mixing the two idioms up is the most common GORM bug.
	var missing userModel
	err := tx.First(&missing, "email = ?", "nobody@example.com").Error
	fmt.Println("First on nothing -> errors.Is(err, gorm.ErrRecordNotFound) =", errors.Is(err, gorm.ErrRecordNotFound))
	var none []userModel
	res := tx.Find(&none, "email = ?", "nobody@example.com")
	fmt.Printf("Find  on nothing -> err=%v rowsAffected=%d len=%d  <- NOT an error\n",
		res.Error, res.RowsAffected, len(none))

	// Trap 2: Updates with a STRUCT skips zero values - GORM cannot tell "set to zero" from
	// "not set" (module 001a's zero-value ambiguity). Use a map, or Select the columns.
	_ = tx.Model(&u).Updates(userModel{Name: "Ada Lovelace"}).Error
	fmt.Printf("Updates(struct{Name: \"Ada Lovelace\"}) -> %q\n", reload(tx, u.ID).Name)
	_ = tx.Model(&u).Updates(userModel{Name: ""}).Error
	fmt.Printf("Updates(struct{Name: \"\"})             -> %q  <- the zero value was SKIPPED\n", reload(tx, u.ID).Name)
	_ = tx.Model(&u).Updates(map[string]any{"name": ""}).Error
	fmt.Printf("Updates(map{\"name\": \"\"})              -> %q  <- a map says what it means\n", reload(tx, u.ID).Name)
	_ = tx.Model(&u).Updates(map[string]any{"name": "Ada"}).Error

	// Trap 3: Save writes EVERY column (INSERT if the key is zero), silently clobbering a
	// concurrent update. Prefer Updates.
	// Trap 4: Delete/Update with no condition is refused - a guard against DELETE with no WHERE.
	// db.Where("1=1").Delete(...) overrides it, which should feel appropriately uncomfortable.
	fmt.Println("Delete with no WHERE ->", shorten(tx.Delete(&userModel{}).Error))

	// Soft delete: with a DeletedAt field, Delete only sets the timestamp, and EVERY later query
	// silently adds WHERE deleted_at IS NULL. Unscoped() sees the deleted rows.
	_ = tx.Delete(&userModel{}, u.ID).Error
	var visible, withDeleted int64
	_ = tx.Model(&userModel{}).Where("id = ?", u.ID).Count(&visible).Error
	_ = tx.Unscoped().Model(&userModel{}).Where("id = ?", u.ID).Count(&withDeleted).Error
	fmt.Printf("after Delete: a normal query sees %d, Unscoped sees %d\n", visible, withDeleted)

	// The trap that surprises everyone: a uniqueIndex still covers soft-deleted rows. The fix is
	// a partial index (WHERE deleted_at IS NULL), which AutoMigrate will not create.
	reuse := tx.Create(&userModel{Name: "Ada Again", Email: "ada@example.com"}).Error
	fmt.Println("re-creating with the same email ->", shorten(reuse))

	_ = tx.Unscoped().Delete(&userModel{}, u.ID).Error // really removes the row
	_ = tx.Unscoped().Model(&userModel{}).Where("id = ?", u.ID).Count(&withDeleted).Error
	fmt.Println("after Unscoped().Delete, rows with that id:", withDeleted)
}

func reload(tx *gorm.DB, id int64) userModel {
	var u userModel
	_ = tx.First(&u, id).Error
	return u
}

// queryingAndPreload shows the query builder and the N+1 problem, measured.
func queryingAndPreload(ctx context.Context, db *gorm.DB) {
	fmt.Println("\n--- querying, Preload and the N+1 problem ---")
	if db == nil {
		fmt.Println("(skipped: no database)")
		return
	}
	tx := db.WithContext(ctx)

	for i, name := range []string{"One", "Two", "Three"} {
		buyer := userModel{Name: "Buyer " + name, Email: strings.ToLower(name) + "@example.com"}
		if err := tx.Create(&buyer).Error; err != nil {
			fmt.Println("seed failed:", err)
			return
		}
		for j := range 3 {
			_ = tx.Create(&orderModel{UserID: buyer.ID, Item: fmt.Sprintf("item-%d-%d", i+1, j+1),
				Amount: int64((i + 1) * (j + 1) * 100)}).Error
		}
	}

	// Each chain call returns a new *gorm.DB. But a value on which Where (etc.) was already
	// called ACCUMULATES conditions: build only on a fresh one - db, db.WithContext(ctx) or
	// db.Session(&gorm.Session{}). Also: Where(map[string]any{...}), Joins, Raw.
	var expensive []orderModel
	_ = tx.Where("amount > ?", 400).Order("amount desc").Limit(3).Find(&expensive).Error
	fmt.Printf("Where+Order+Limit -> %d orders, top amount=%d\n", len(expensive), expensive[0].Amount)

	var slim []userModel
	_ = tx.Select("id", "name").Where("email like ?", "%@example.com").Find(&slim).Error
	fmt.Printf("Select(\"id\", \"name\") -> %d users, first email %q (not loaded)\n", len(slim), slim[0].Email)

	// Count queries with a callback. Callbacks live in the connection's callback table, shared
	// by EVERY session derived from it - so remove it again afterwards.
	var queries int
	_ = tx.Callback().Query().After("gorm:query").Register("count_queries", func(*gorm.DB) { queries++ })

	// N+1: load users, then each user's orders in a loop - 1 + N queries. It passes tests with
	// three rows and collapses in production.
	var users []userModel
	queries = 0
	_ = tx.Find(&users).Error
	for i := range users {
		_ = tx.Where("user_id = ?", users[i].ID).Find(&users[i].Orders).Error
	}
	naive := queries

	// Preload: ONE extra query per association, using WHERE user_id IN (...) - the default choice.
	// Nested: Preload("Orders.Items"); conditional: Preload("Orders", "amount > ?", 100);
	// everything one level deep: Preload(clause.Associations).
	var preloaded []userModel
	queries = 0
	_ = tx.Preload("Orders").Find(&preloaded).Error
	withPreload := queries
	_ = tx.Callback().Query().Remove("count_queries")
	fmt.Printf("%d users and their orders: naive loop %d queries (1+N), Preload %d; same %d = %d orders\n",
		len(users), naive, withPreload, totalOrders(users), totalOrders(preloaded))
	// Nothing warns about N+1 at compile time: read the SQL (db.Debug()), count queries in a load test.

	// Joins issues one JOIN instead - good for belongs-to, wasteful for has-many (each parent
	// row repeats per child). Joins("Orders") does the LEFT JOIN by association name.
	var joined []userModel
	_ = tx.Joins("JOIN m020_orders ON m020_orders.user_id = m020_users.id").
		Where("m020_orders.amount > ?", 500).Distinct().Find(&joined).Error
	fmt.Println("Joins + Distinct -> users with an order over 500:", len(joined))

	// Raw, for when the chain fights you - here an aggregate.
	var report []struct {
		Name  string
		Total int64
	}
	_ = tx.Raw(`select u.name, coalesce(sum(o.amount), 0) as total
	            from m020_users u left join m020_orders o on o.user_id = u.id
	            where u.deleted_at is null
	            group by u.name order by total desc limit 3`).Scan(&report).Error
	for _, r := range report {
		fmt.Printf("db.Raw aggregate: %-12s total=%d\n", r.Name, r.Total)
	}
}

func totalOrders(users []userModel) int {
	total := 0
	for _, u := range users {
		total += len(u.Orders)
	}
	return total
}

// associations shows has-many and many2many relationships.
func associations(ctx context.Context, db *gorm.DB) {
	fmt.Println("\n--- associations ---")
	if db == nil {
		fmt.Println("(skipped: no database)")
		return
	}
	tx := db.WithContext(ctx)

	// GORM infers relationships from field types and names (the foreign key is <Owner>ID;
	// foreignKey/references override it): belongs to (Order{UserID; User}), has one
	// (User{Profile}), has many (User.Orders), many to many (User.Tags via a join table).

	// Creating a parent creates its children too (FullSaveAssociations extends that to
	// updates) - occasionally more writing than intended. Omit("Orders") turns it off.
	author := userModel{Name: "Composer", Email: "composer@example.com",
		Orders: []orderModel{{Item: "nested-a", Amount: 10}, {Item: "nested-b", Amount: 20}}}
	if err := tx.Create(&author).Error; err != nil {
		fmt.Println("create failed:", err)
		return
	}
	fmt.Printf("Create with nested Orders -> user id=%d, order ids=[%d %d]\n",
		author.ID, author.Orders[0].ID, author.Orders[1].ID)

	var loaded userModel
	_ = tx.Preload("Orders").First(&loaded, author.ID).Error
	fmt.Println("Preload(\"Orders\") ->", len(loaded.Orders), "orders")

	// many2many through the join table, and the Association API: Append, Delete, Replace,
	// Clear and Count change a relationship without loading the whole graph.
	tags := []tagModel{{Label: "vip"}, {Label: "beta"}}
	for i := range tags {
		_ = tx.Where(tagModel{Label: tags[i].Label}).FirstOrCreate(&tags[i]).Error
	}
	if err := tx.Model(&loaded).Association("Tags").Append(&tags); err != nil {
		fmt.Println("association append failed:", err)
	}
	fmt.Println("Association(\"Tags\").Append ->", tx.Model(&loaded).Association("Tags").Count(), "tags")

	var tagged tagModel
	_ = tx.Preload("Users").First(&tagged, "label = ?", "vip").Error
	fmt.Printf("from the other side: tag %q has %d user(s)\n", tagged.Label, len(tagged.Users))

	firstTag := tags[:1]
	_ = tx.Model(&loaded).Association("Tags").Replace(&firstTag)
	fmt.Println("Association.Replace ->", tx.Model(&loaded).Association("Tags").Count(), "tag(s)")
	_ = tx.Model(&loaded).Association("Tags").Clear()
	fmt.Println("Association.Clear   ->", tx.Model(&loaded).Association("Tags").Count(), "tag(s)")

	// Without `constraint:OnDelete:CASCADE` in the tag there is NO database-level foreign key:
	// associations are a Go-side convention and the database will happily hold orphans.
	// Deep object graphs are where ORMs surprise you: load one level at a time, read the SQL.
}

// transactionsAndHooks shows managed transactions and model hooks.
func transactionsAndHooks(ctx context.Context, db *gorm.DB) {
	fmt.Println("\n--- transactions and hooks ---")
	if db == nil {
		fmt.Println("(skipped: no database)")
		return
	}
	tx := db.WithContext(ctx)
	before := countUsers(ctx, db)

	// The MANAGED form commits on nil and rolls back on an error or a panic: no transaction can
	// leak, no defer tx.Rollback() to forget, nested calls become savepoints. (The manual
	// db.Begin/Commit/Rollback mirrors module 019.) Inside the closure use t, NEVER db: db is a
	// different connection - easier to get wrong here because both are in scope.
	err := tx.Transaction(func(t *gorm.DB) error {
		if err := t.Create(&userModel{Name: "Tx A", Email: "tx-a@example.com"}).Error; err != nil {
			return err
		}
		return t.Create(&userModel{Name: "Tx B", Email: "tx-b@example.com"}).Error
	})
	afterCommit := countUsers(ctx, db)
	fmt.Printf("committed: err=%v, rows %d -> %d\n", err, before, afterCommit)

	err = tx.Transaction(func(t *gorm.DB) error {
		if err := t.Create(&userModel{Name: "Tx C", Email: "tx-c@example.com"}).Error; err != nil {
			return err
		}
		return errors.New("deliberate failure after the first insert")
	})
	fmt.Printf("failed: err=%v, rows %d -> %d (rolled back)\n", err, afterCommit, countUsers(ctx, db))

	fmt.Println("a panic inside Transaction:", catchPanic(func() {
		_ = tx.Transaction(func(t *gorm.DB) error {
			_ = t.Create(&userModel{Name: "Tx D", Email: "tx-d@example.com"}).Error
			panic("boom")
		})
	}), "- rows still", countUsers(ctx, db))

	// Hooks: BeforeSave/Create/Update/Delete, AfterFind, ... are methods on the model.
	messy := userModel{Name: "Hooked", Email: "  MiXeD@Example.COM  "}
	if err := tx.Create(&messy).Error; err == nil {
		fmt.Printf("BeforeCreate normalised the email to %q\n", messy.Email)
	}
	invalid := userModel{Name: "", Email: "no-name@example.com"}
	fmt.Println("a hook returning an error aborts the write:", shorten(tx.Create(&invalid).Error))
}

// BeforeCreate normalises the email. Returning an error aborts the operation and rolls back.
// Hooks are fine for normalisation, derived fields and audit columns, but they run on EVERY save
// (bulk operations too), are invisible at the call site and tie the model to *gorm.DB. Keep them
// small and about the row: business logic in a hook is business logic nobody can find.
func (u *userModel) BeforeCreate(tx *gorm.DB) error {
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	if u.Name == "" {
		return errors.New("userModel: name must not be empty")
	}
	return nil
}

func countUsers(ctx context.Context, db *gorm.DB) int64 {
	var n int64
	_ = db.WithContext(ctx).Model(&userModel{}).Count(&n).Error
	return n
}

// catchPanic runs f and returns whatever it panicked with (a local copy of module 005's helper).
func catchPanic(f func()) (recovered any) {
	defer func() {
		if r := recover(); r != nil {
			recovered = r
		}
	}()
	f()
	return "did not panic"
}

// repositoryOnGorm implements module 019's repository interface a third time, on GORM.
func repositoryOnGorm(ctx context.Context, db *gorm.DB) {
	fmt.Println("\n--- the same repository, on GORM ---")
	if db == nil {
		fmt.Println("(skipped: no database - module 019 runs the in-memory implementation)")
		return
	}

	// Clean tables, so the output matches module 019's. One FRESH chain per statement: reusing
	// a chain after Unscoped()/Delete would carry its state into the next one.
	for _, model := range []any{&orderModel{}, &userModel{}} {
		_ = db.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(model).Error
	}

	// The storage engine is a swappable detail: the same contract, the same domain errors.
	exerciseRepository(ctx, &gormRepo{db: db}, "GORM (module 019's interface, third implementation)")

	// gorm.ErrRecordNotFound and the unique violation become domain errors at the boundary, so
	// gorm never escapes the repository - as database/sql never escaped module 019's.
	// Lines per method: database/sql ~15 (explicit SQL and scanning), GORM ~4 (generated SQL,
	// reflection), in-memory ~6 (a map and a mutex). Shorter versus predictable.
}

// gormRepo maps between the domain user and the GORM model.
type gormRepo struct{ db *gorm.DB }

var _ userRepository = (*gormRepo)(nil) // compile-time proof, as in module 019

func (r *gormRepo) Create(ctx context.Context, u *user) error {
	model := userModel{Name: u.Name, Email: u.Email}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return translateError(err)
	}
	u.ID = model.ID
	return nil
}

func (r *gormRepo) ByID(ctx context.Context, id int64) (*user, error) {
	var m userModel
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, translateError(err)
	}
	return &user{ID: m.ID, Name: m.Name, Email: m.Email}, nil
}

func (r *gormRepo) List(ctx context.Context) ([]user, error) {
	var models []userModel
	if err := r.db.WithContext(ctx).Order("id").Find(&models).Error; err != nil {
		return nil, translateError(err)
	}
	out := make([]user, len(models))
	for i, m := range models {
		out[i] = user{ID: m.ID, Name: m.Name, Email: m.Email}
	}
	return out, nil
}

func (r *gormRepo) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&userModel{}, id)
	if res.Error != nil {
		return translateError(res.Error)
	}
	if res.RowsAffected == 0 {
		return errNotFound // deleting nothing is no error to GORM - Find's trap again
	}
	return nil
}

// translateError keeps GORM from escaping the repository.
func translateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errNotFound
	}
	// A dependency-free stand-in; the real form is
	// var pgErr *pgconn.PgError; errors.As(err, &pgErr) && pgErr.Code == "23505".
	if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key") {
		return errDuplicate
	}
	return err
}

// The domain side below mirrors module 019 (user, userRepository, the domain errors and the
// contract exerciseRepository), copied so that this module is self-contained.

type user struct {
	ID    int64
	Name  string
	Email string
}

type userRepository interface {
	Create(ctx context.Context, u *user) error
	ByID(ctx context.Context, id int64) (*user, error)
	List(ctx context.Context) ([]user, error)
	Delete(ctx context.Context, id int64) error
}

var (
	errNotFound  = errors.New("not found")
	errDuplicate = errors.New("already exists")
)

// exerciseRepository is the CONTRACT, run identically against any implementation.
func exerciseRepository(ctx context.Context, repo userRepository, label string) {
	fmt.Println(label + ":")
	u := &user{Name: "Repo Ada", Email: "repo-ada@example.com"}
	if err := repo.Create(ctx, u); err != nil {
		fmt.Println("  Create:", err)
		return
	}
	fmt.Printf("  Create           -> id=%d\n", u.ID)
	dup := &user{Name: "Repo Dup", Email: "repo-ada@example.com"}
	fmt.Println("  Create duplicate -> errors.Is(err, errDuplicate) =",
		errors.Is(repo.Create(ctx, dup), errDuplicate))
	found, err := repo.ByID(ctx, u.ID)
	fmt.Printf("  %-16s -> %+v (err=%v)\n", fmt.Sprintf("ByID(%d)", u.ID), deref(found), err)
	_, err = repo.ByID(ctx, 999999)
	fmt.Println("  ByID(999999)     -> errors.Is(err, errNotFound) =", errors.Is(err, errNotFound),
		"<- NOT gorm.ErrRecordNotFound")
	list, err := repo.List(ctx)
	fmt.Printf("  List             -> %d user(s) (err=%v)\n", len(list), err)
	fmt.Printf("  %-16s -> %v\n", fmt.Sprintf("Delete(%d)", u.ID), repo.Delete(ctx, u.ID))
	fmt.Println("  Delete again     -> errors.Is(err, errNotFound) =",
		errors.Is(repo.Delete(ctx, u.ID), errNotFound))
}

func deref(u *user) user {
	if u == nil {
		return user{}
	}
	return *u
}
