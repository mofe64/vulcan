package dbconfig

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mofe64/vulcan/internal/logs"
)

// SetUpAndConnectToDatabase sets up a connection to the database
func SetUpAndConnectToMySQLDatabase(username string, password string, host string, port string, dbName string) *sql.DB {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Connect to database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", username, password, host, port, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logs.ErrorLog.Fatalf("Error connecting to database: %v", err)
	}
	err = db.PingContext(ctx)
	if err != nil {
		logs.ErrorLog.Fatalf("Error pinging database: %v", err)
	}
	logs.InfoLog.Println("Connected to database")
	return db
}
