package repository

import (
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/ujwegh/gophermart/internal/app/config"
	"github.com/ujwegh/gophermart/migrations"
	"io/fs"
)

type DBStorage struct {
	DBConn *sqlx.DB
}

func open(dataSourceName string) *sqlx.DB {
	db, err := sqlx.Open("pgx", dataSourceName)
	db.SetMaxOpenConns(10)
	if err != nil {
		panic(err)
	}
	return db
}

func Migrate(db *sqlx.DB, dir string) error {
	err := goose.SetDialect("postgres")
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	err = goose.Up(db.DB, dir)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

func MigrateFS(db *sqlx.DB, migrationsFS fs.FS, dir string) error {
	if dir == "" {
		dir = "."
	}
	goose.SetBaseFS(migrationsFS)
	defer func() {
		goose.SetBaseFS(nil)
	}()
	return Migrate(db, dir)
}

func NewDBStorage(cfg config.AppConfig) *DBStorage {
	db := open(cfg.DatabaseURI)
	// Migrate the database
	err := MigrateFS(db, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	return &DBStorage{DBConn: db}
}
