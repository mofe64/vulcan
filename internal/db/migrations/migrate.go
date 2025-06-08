package migrations

import (
	"context"
	"database/sql"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
)

// func RunMigrations(dbURL string, logger *zap.Logger) error {
// 	// A. Init source driver (reads from embed.FS)
// 	d, err := iofs.New(FS, ".")
// 	if err != nil {
// 		return err
// 	}

// 	// B. Init database driver
// 	db, err := sql.Open("pgx", dbURL)
// 	if err != nil {
// 		return err
// 	}
// 	driver, err := postgres.WithInstance(db, &postgres.Config{})
// 	if err != nil {
// 		return err
// 	}

// 	// C. Perform up migrations (idempotent)
// 	m, err := migrate.NewWithInstance("iofs", d, "postgres", driver)
// 	if err != nil {
// 		return err
// 	}

// 	err = m.Up()
// 	if err != nil && err != migrate.ErrNoChange {
// 		return err
// 	}
// 	if err == migrate.ErrNoChange {
// 		logger.Info("DB schema up-to-date")
// 	} else {
// 		logger.Info("DB schema migrated successfully")
// 	}
// 	return nil
// }

// RunMigrations runs all pending up-migrations.
// It returns ctx.Err() if the caller cancels or the timeout expires.
func RunMigrations(ctx context.Context, dbURL string, log *zap.Logger) error {
	// init source driver (reads from embed.FS)
	d, err := iofs.New(FS, ".")
	if err != nil {
		return err
	}

	// *context-aware* database connection.
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return err
	}
	// optional: shorten driver-level I/O timeouts as extra safety.
	db.SetConnMaxIdleTime(30 * time.Second)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	// start the migrator.
	m, err := migrate.NewWithInstance("iofs", d, "postgres", driver)
	if err != nil {
		return err
	}

	// run Up() in its own goroutine so we can watch ctx.
	done := make(chan error, 1)
	go func() { done <- m.Up() }()

	select {
	case <-ctx.Done():
		// caller timed out or sent SIGTERM.
		// try to cleanly close the migration to unblock DB driver.
		_, _ = m.Close()
		return ctx.Err()
	case err := <-done:
		if err != nil && err != migrate.ErrNoChange {
			return err
		}
		if err == migrate.ErrNoChange {
			log.Info("DB schema up-to-date")
		} else {
			log.Info("DB schema migrated successfully")
		}
		return nil
	}
}
