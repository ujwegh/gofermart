package repository

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/ujwegh/gophermart/internal/app/config"
	"github.com/ujwegh/gophermart/migrations"
)

type DBStorage struct {
	DbConn *sqlx.DB
}

func NewDBStorage(cfg config.AppConfig) *DBStorage {
	db := Open(cfg.DatabaseDSN)
	// Migrate the database
	err := MigrateFS(db, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	return &DBStorage{DbConn: db}
}
