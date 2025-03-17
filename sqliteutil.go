package sqliteutil

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
)

// OpenSqlite opens a sqlite database with [non-deadlocking settings](https://stackoverflow.com/questions/35804884/sqlite-concurrent-writing-performance)
// without running any migrations.
func OpenSqlite(path string) (*sql.DB, error) {
	if path != ":memory:" {
		os.MkdirAll(filepath.Dir(path), 0777)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, wrapErr(err)
	}

	// see this stackoverflow post for information on why the following
	// lines exist: https://stackoverflow.com/questions/35804884/sqlite-concurrent-writing-performance
	db.SetMaxOpenConns(1)
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return nil, wrapErr(err)
	}

	return db, nil
}

func wrapErr(err error) error {
	return fmt.Errorf("open sqlite: %w", err)
}

// OpenAndMigrateSqlite opens a sqlite database (and creates it if it does not exist) at a given
// path, it runs migrations using [atlas](https://atlasgo.io) if available.
//
// Do note that you must have a sqlite database driver initialized for the database to open.
//
//	import (
//		"github.com/LQR471814/sqliteutil"
//
//		_ "modernc.org/sqlite"
//	)
//
//	func main() {
//		sqliteutil.OpenAndMigrateSqlite("create table ...", "path/to/some.db")
//	}
func OpenAndMigrateSqlite(schema, path string) (*sql.DB, error) {
	// create a db if it does not already exist
	db, err := OpenSqlite(path)
	if err != nil {
		return nil, wrapErr(err)
	}
	err = db.Close()
	if err != nil {
		return nil, wrapErr(err)
	}

	_, err = exec.LookPath("atlas")
	if os.IsNotExist(err) {
		return db, wrapErr(fmt.Errorf(
			"could not find 'atlas' executable on path, is it installed? skipping migrations...",
		))
	}

	err = os.WriteFile("temp_migration_schema.sql", []byte(schema), 0666)
	if err != nil {
		return nil, wrapErr(err)
	}
	defer func() {
		err = os.Remove("temp_migration_schema.sql")
		if err != nil {
			slog.Warn("could not delete temp_migration_schema.sql", "err", err)
		}
	}()

	dbUrl := url.URL{
		Scheme: "sqlite",
		Path:   path,
	}
	cmd := exec.Command(
		"atlas", "schema", "apply",
		"--url", dbUrl.String(),
		"--to", "file://temp_migration_schema.sql",
		"--dev-url", "sqlite://file?mode=memory",
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return nil, wrapErr(err)
	}

	return OpenSqlite(path)
}
