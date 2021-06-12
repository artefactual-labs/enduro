package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/prometheus/client_golang/prometheus"
)

// Connect returns the database handler which is safe for concurrent access.
func Connect(ds string) (db *sql.DB, err error) {
	config, err := mysqldriver.ParseDSN(ds)
	if err != nil {
		return nil, fmt.Errorf("error parsing dsn: %w (%s)", err, ds)
	}
	config.Collation = "utf8mb4_unicode_ci"
	config.Loc = time.UTC
	config.ParseTime = true
	config.MultiStatements = true
	config.Params = map[string]string{
		"time_zone": "UTC",
	}

	conn, err := mysqldriver.NewConnector(config)
	if err != nil {
		return nil, fmt.Errorf("error creating connector: %w", err)
	}

	db = sql.OpenDB(conn)

	// Set reasonable sizes on the built-in pool.
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(30)
	db.SetConnMaxLifetime(time.Minute)

	// Register Prometheus collector.
	c := prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "src",
			Subsystem: "mysql_app",
			Name:      "open_connections",
			Help:      "Number of open connections to MySQL DB, as reported by mysql.DB.Stats()",
		},
		func() float64 {
			s := db.Stats()
			return float64(s.OpenConnections)
		},
	)
	prometheus.MustRegister(c)

	// Database migration.
	m, err := newMigrate(db)
	if err != nil {
		return nil, fmt.Errorf("error creating migrate object: %v", err)
	}
	if err := doMigrate(m); err != nil {
		return nil, fmt.Errorf("error during database migration: %v", err)
	}

	return db, nil
}

func newMigrate(db *sql.DB) (*migrate.Migrate, error) {
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return nil, fmt.Errorf("error creating migrate driver: %w", err)
	}

	sourceInstance, err := sourceDriver()
	if err != nil {
		return nil, fmt.Errorf("error creating source driver: %v", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceInstance, "mysql", driver)
	if err != nil {
		return nil, fmt.Errorf("error creating migrate instance: %w", err)
	}

	// Wait up to five minutes is another process is already on it.
	m.LockTimeout = 5 * time.Minute

	return m, nil
}

func doMigrate(m *migrate.Migrate) error {
	err := m.Up()
	if err == nil || err == migrate.ErrNoChange {
		return nil
	}

	if os.IsNotExist(err) {
		_, dirty, verr := m.Version()
		if verr != nil {
			return verr
		}
		if dirty {
			return err
		}
		return nil
	}

	return err
}
