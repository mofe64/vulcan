package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "github.com/go-sql-driver/mysql" // mySQL driver.
	_ "github.com/lib/pq"              // postgreSQL driver.
	_ "github.com/mattn/go-sqlite3"    // SQLite driver.

	"github.com/mofe64/vulcan/config"
)

// InitDB: initializes the database connection based on the provided config.
func InitDB(cfg *config.VulkanConfig) (*sql.DB, error) {
	var driver, connStr string
	switch cfg.DBType {
	case "sqlite":
		driver = "sqlite3"
		connStr = cfg.DBConnection // e.g., "./data.sqlite"
	case "postgres":
		driver = "postgres"
		connStr = cfg.DBConnection // e.g., "postgres://user:pass@localhost/dbname"
	case "mysql":
		driver = "mysql"
		connStr = cfg.DBConnection // e.g., "user:pass@tcp(localhost:3306)/dbname"
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.DBType)
	}

	// open connection to the database.
	dbConn, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, err
	}

	// test the connection.
	if err := dbConn.Ping(); err != nil {
		return nil, err
	}

	// initialize the database schema by reading from a SQL file.
	if err := initSchema(dbConn, cfg.DBType); err != nil {
		return nil, err
	}

	return dbConn, nil
}

// GetVulkanDBPath checks for the existence of vulkan.db in an OS-appropriate directory.
// If the file doesn't exist, it creates an empty one and returns the full file path.
// GetVulkanDBPath is designed specifically for persistent storage (SQLite database)
// in a standardized, OS-specific, user data location (APPDATA, ~/Library, ~/.local/share).
func GetVulkanDBPath() (string, error) {
	var dataDir string

	switch runtime.GOOS {
	case "windows":
		// for Windows, use %APPDATA%\vulkan\data
		baseDir := os.Getenv("APPDATA")
		if baseDir == "" {
			var err error
			baseDir, err = os.UserHomeDir()
			if err != nil {
				return "", err
			}
		}
		dataDir = filepath.Join(baseDir, "vulkan", "data")
	case "darwin":
		// for macOS, use ~/Library/Application Support/vulkan/data
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataDir = filepath.Join(home, "Library", "Application Support", "vulkan", "data")
	default:
		// for Linux and other OSes, use $XDG_DATA_HOME/vulkan/data if available,
		// otherwise default to ~/.local/share/vulkan/data
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			dataDir = filepath.Join(home, ".local", "share", "vulkan", "data")
		} else {
			dataDir = filepath.Join(xdgData, "vulkan", "data")
		}
	}

	// ensure the data directory exists.
	// rwxr-xr-x (755) permissions for the directory.
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", err
	}

	// construct the full path for the SQLite file.
	dbPath := filepath.Join(dataDir, "vulkan.db")

	// check if the file exists.
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// file doesn't exist, so create it.
		file, err := os.Create(dbPath)
		if err != nil {
			return "", err
		}
		// close the file immediately; SQLite will manage it after opening.
		file.Close()
	}

	return dbPath, nil
}

// initSchema reads the schema from an external SQL file and executes it,
// then inserts a default user record.
// Todo: option to turn off default root user creation
func initSchema(db *sql.DB, dbType string) error {
	// choose file name based on the database type.
	var fileName string
	switch dbType {
	case "sqlite":
		fileName = "schema_sqlite.sql"
	case "postgres":
		fileName = "schema_postgres.sql"
	case "mysql":
		fileName = "schema_mysql.sql"
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Assume the SQL files are in a folder named "sql" located in the working directory.
	schemaPath := filepath.Join("sql", fileName)
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file (%s): %w", schemaPath, err)
	}
	schemaSQL := string(schemaBytes)

	// Execute the schema SQL. This script should include CREATE TABLE statements using
	// an "IF NOT EXISTS" clause so that running it repeatedly will have no effect if the tables exist.
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("failed to execute schema SQL: %w", err)
	}

	// Prepare the SQL statement for inserting a default user.
	// The SQL syntax differs slightly between databases.
	var insertUserSQL string
	switch dbType {
	case "sqlite":
		insertUserSQL = `
        INSERT OR IGNORE INTO users (id, firstname, lastname, email, password, role)
        VALUES (?, ?, ?, ?, ?, ?);`
	case "postgres":
		insertUserSQL = `
        INSERT INTO users (id, firstname, lastname, email, password, role)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (id) DO NOTHING;`
	case "mysql":
		insertUserSQL = `
        INSERT IGNORE INTO users (id, firstname, lastname, email, password, role)
        VALUES (?, ?, ?, ?, ?, ?);`
	}

	// insert the default user with id=1, firstname="root", lastname="root",
	// email="root@vulkan.local", password="root", and role="root".
	_, err = db.Exec(insertUserSQL, 1, "root", "root", "root@vulkan.local", "root", "root")
	if err != nil {
		return fmt.Errorf("failed to insert default user: %w", err)
	}

	return nil
}
