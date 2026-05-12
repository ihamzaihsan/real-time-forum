package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type DataBase struct {
	DB *sql.DB
}

var DBInstance DataBase

func InitDB() error {
	path := os.Getenv("DATABASE_PATH")
	if path == "" {
		path = "Real-Time-Forum.db"
	}
	db, err := Open(path)
	if err != nil {
		return err
	}
	DBInstance.DB = db
	return nil
}

// Open configures every pooled connection through the driver DSN.
func Open(path string) (*sql.DB, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(abs)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	dsn := (&url.URL{Scheme: "file", Path: uriPath}).String() + "?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL&_txlock=immediate"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	for _, initialize := range []func(*sql.DB) error{(*sql.DB).Ping, CreateTables, AddDefaultCategories, migrate} {
		if err := initialize(db); err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	// A nullable composite UNIQUE constraint does not prevent duplicate reactions.
	// Keep the newest record before adding partial unique indexes to existing DBs.
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	columns, err := tx.Query(`PRAGMA table_info(messages)`)
	if err != nil {
		return err
	}
	hasClientID := false
	for columns.Next() {
		var cid, notNull, primaryKey int
		var name, dataType string
		var defaultValue interface{}
		if err := columns.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			columns.Close()
			return err
		}
		if name == "client_id" {
			hasClientID = true
		}
	}
	err = columns.Err()
	columns.Close()
	if err != nil {
		return err
	}
	if !hasClientID {
		if _, err := tx.Exec(`ALTER TABLE messages ADD COLUMN client_id TEXT`); err != nil {
			return err
		}
	}
	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS messages_client_unique ON messages(sender_id, client_id) WHERE client_id IS NOT NULL`,
		`DELETE FROM likes WHERE post_id IS NOT NULL AND id NOT IN (SELECT MAX(id) FROM likes WHERE post_id IS NOT NULL GROUP BY user_id, post_id)`,
		`DELETE FROM likes WHERE comment_id IS NOT NULL AND id NOT IN (SELECT MAX(id) FROM likes WHERE comment_id IS NOT NULL GROUP BY user_id, comment_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS likes_post_unique ON likes(user_id, post_id) WHERE post_id IS NOT NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS likes_comment_unique ON likes(user_id, comment_id) WHERE comment_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS messages_conversation ON messages(sender_id, receiver_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS posts_created ON posts(created_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS comments_post ON comments(post_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS reports (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE, reason TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, UNIQUE(user_id, post_id))`,
		`UPDATE users SET is_online = FALSE`,
		`CREATE TABLE IF NOT EXISTS moderation_actions (id INTEGER PRIMARY KEY AUTOINCREMENT, moderator_id INTEGER NOT NULL REFERENCES users(id), post_id INTEGER NOT NULL, action TEXT NOT NULL, reason TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("migrate database: %w", err)
		}
	}
	return tx.Commit()
}

func CreateTables(db *sql.DB) error {

	createUsersTable := `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    age INTEGER NOT NULL,
    gender TEXT NOT NULL,
    is_online BOOLEAN DEFAULT FALSE,
    last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

	if _, err := db.Exec(createUsersTable); err != nil {
		return fmt.Errorf("failed to create users table: %v", err)
	}

	createCategoriesTable := `
    CREATE TABLE IF NOT EXISTS categories (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL UNIQUE
    );`

	if _, err := db.Exec(createCategoriesTable); err != nil {
		return fmt.Errorf("failed to create categories table: %v", err)
	}

	createPostsTable := `
    CREATE TABLE IF NOT EXISTS posts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        content TEXT NOT NULL,
        image_path TEXT,
        user_id INTEGER NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
    );`

	if _, err := db.Exec(createPostsTable); err != nil {
		return fmt.Errorf("failed to create posts table: %v", err)
	}

	createPostCategoriesTable := `
        CREATE TABLE IF NOT EXISTS post_categories (
            post_id INTEGER,
            category_id INTEGER,
            PRIMARY KEY (post_id, category_id),
            FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE,
            FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE CASCADE
        );`

	if _, err := db.Exec(createPostCategoriesTable); err != nil {
		return fmt.Errorf("failed to create post_categories table: %v", err)
	}

	createCommentsTable := `
    CREATE TABLE IF NOT EXISTS comments (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        content TEXT NOT NULL,
        user_id INTEGER NOT NULL,
        post_id INTEGER NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE
    );`

	if _, err := db.Exec(createCommentsTable); err != nil {
		return fmt.Errorf("failed to create comments table: %v", err)
	}

	createLikesTable := `
    CREATE TABLE IF NOT EXISTS likes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    post_id INTEGER,
    comment_id INTEGER,
    is_like BOOLEAN NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE,
    FOREIGN KEY (comment_id) REFERENCES comments (id) ON DELETE CASCADE,
    CONSTRAINT unique_like UNIQUE (user_id, post_id, comment_id)
  );`

	if _, err := db.Exec(createLikesTable); err != nil {
		return fmt.Errorf("failed to create likes table: %v", err)
	}

	createMessagesTable := `
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sender_id INTEGER NOT NULL,
    receiver_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_read BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (sender_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (receiver_id) REFERENCES users (id) ON DELETE CASCADE
);`

	if _, err := db.Exec(createMessagesTable); err != nil {
		return fmt.Errorf("failed to create messages table: %v", err)
	}

	createSessionTable := `
    CREATE TABLE IF NOT EXISTS sessions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        session_token TEXT NOT NULL UNIQUE,
        email TEXT NOT NULL,
        expires_at DATETIME NOT NULL,
        FOREIGN KEY (email) REFERENCES users (email) ON DELETE CASCADE
    );`

	if _, err := db.Exec(createSessionTable); err != nil {
		return fmt.Errorf("failed to create sessions table: %v", err)
	}

	return nil
}

func AddDefaultCategories(db *sql.DB) error {

	categories := []string{"science", "technology", "art", "sport", "games"}

	stmt, err := db.Prepare("INSERT OR IGNORE INTO categories (name) VALUES (?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, category := range categories {
		_, err := stmt.Exec(category)
		if err != nil {

			return err
		}
	}
	return nil
}
