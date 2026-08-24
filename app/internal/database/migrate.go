package database

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// The .sql files are compiled into the binary, so the container and the
// `migrate` command always carry the exact schema that the code expects.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

func newMigrator(cfg Config) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("load embedded migrations: %w", err)
	}
	// the pgx/v5 migrate driver expects a pgx:// scheme rather than postgres://
	m, err := migrate.NewWithSourceInstance("iofs", src, "pgx5://"+cfg.dsnBody())
	if err != nil {
		return nil, fmt.Errorf("init migrator: %w", err)
	}
	return m, nil
}

// MigrateUp applies every pending migration. It is safe to call from several
// instances at once: the driver takes a Postgres advisory lock first.
func MigrateUp(cfg Config) error {
	m, err := newMigrator(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// MigrateDown rolls back the given number of migrations, or all of them when
// steps is 0.
func MigrateDown(cfg Config, steps int) error {
	m, err := newMigrator(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if steps == 0 {
		err = m.Down()
	} else {
		err = m.Steps(-steps)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("roll back migrations: %w", err)
	}
	return nil
}

// MigrateVersion reports the schema version and whether the last run left it
// dirty (a migration failed part way and needs a manual force).
func MigrateVersion(cfg Config) (uint, bool, error) {
	m, err := newMigrator(cfg)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read schema version: %w", err)
	}
	return version, dirty, nil
}

// MigrateForce clears the dirty flag and pins the schema at version. Only use
// it after checking by hand what the failed migration actually applied.
func MigrateForce(cfg Config, version int) error {
	m, err := newMigrator(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("force schema version %d: %w", version, err)
	}
	return nil
}
