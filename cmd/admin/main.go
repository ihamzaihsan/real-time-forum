// admin promotes an existing trusted account using local database access.
package main

import (
	"RTF/database"
	"flag"
	"fmt"
	"os"
)

func main() {
	path := flag.String("database", os.Getenv("DATABASE_PATH"), "existing database file")
	email := flag.String("email", "", "existing account email")
	flag.Parse()
	if *path == "" {
		*path = "Real-Time-Forum.db"
	}
	if *email == "" {
		fmt.Fprintln(os.Stderr, "-email is required")
		os.Exit(1)
	}
	if _, err := os.Stat(*path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	db, err := database.Open(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()
	result, err := db.Exec("UPDATE users SET role='admin' WHERE email=? COLLATE NOCASE", *email)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		fmt.Fprintln(os.Stderr, "Account not found")
		os.Exit(1)
	}
	fmt.Println("Promoted existing account. No password or public credentials were created.")
}
