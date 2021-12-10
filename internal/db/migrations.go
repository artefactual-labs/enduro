package db

import (
	"embed"

	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var fs embed.FS

func sourceDriver() (source.Driver, error) {
	return iofs.New(fs, "migrations")
}
