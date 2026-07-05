package database

import (
	"database/sql"
	"fmt"
)

func addColumn(tx *sql.Tx, table, name, definition string) error {
	rows, err := tx.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var id, required, key int
		var column, kind string
		var defaultValue interface{}
		if err := rows.Scan(&id, &column, &kind, &required, &defaultValue, &key); err != nil {
			rows.Close()
			return err
		}
		found = found || column == name
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if !found {
		_, err = tx.Exec("ALTER TABLE " + table + " ADD COLUMN " + name + " " + definition)
	}
	return err
}

func migrateExtensions(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, column := range []struct{ table, name, definition string }{
		{"users", "role", "TEXT NOT NULL DEFAULT 'member'"}, {"posts", "status", "TEXT NOT NULL DEFAULT 'published'"}, {"comments", "status", "TEXT NOT NULL DEFAULT 'published'"},
		{"moderation_actions", "target_kind", "TEXT NOT NULL DEFAULT 'post'"},
		{"moderation_actions", "target_id", "INTEGER"},
	} {
		if err := addColumn(tx, column.table, column.name, column.definition); err != nil {
			return err
		}
	}
	statements := []string{
		`CREATE TABLE IF NOT EXISTS oauth_identities (provider TEXT NOT NULL, subject TEXT NOT NULL, user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, PRIMARY KEY(provider,subject), UNIQUE(provider,user_id))`,
		`CREATE TABLE IF NOT EXISTS oauth_attempts (state TEXT PRIMARY KEY, expires_at DATETIME NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS oauth_pending (id TEXT PRIMARY KEY, provider TEXT NOT NULL, subject TEXT NOT NULL, email TEXT NOT NULL, expires_at DATETIME NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS forum_settings (name TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS moderator_requests (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), message TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending', reply TEXT NOT NULL DEFAULT '', created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS pending_moderator_request ON moderator_requests(user_id) WHERE status='pending'`,
		`CREATE INDEX IF NOT EXISTS posts_visibility ON posts(status,user_id,id)`,
		`CREATE INDEX IF NOT EXISTS comments_visibility ON comments(post_id,status,user_id)`,
		`DELETE FROM oauth_attempts WHERE julianday(expires_at)<=julianday('now')`,
		`DELETE FROM oauth_pending WHERE julianday(expires_at)<=julianday('now')`,
	}
	for _, query := range statements {
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("extension migration: %w", err)
		}
	}
	// Retain old report IDs and context while replacing the cascading post reference.
	var modern bool
	rows, err := tx.Query("PRAGMA table_info(reports)")
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, n, k int
		var name, kind string
		var value interface{}
		if err := rows.Scan(&id, &name, &kind, &n, &value, &k); err != nil {
			rows.Close()
			return err
		}
		modern = modern || name == "reply"
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if !modern {
		for _, q := range []string{
			`CREATE TABLE reports_new (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL REFERENCES users(id), post_id INTEGER REFERENCES posts(id) ON DELETE SET NULL, original_post_id INTEGER NOT NULL, title TEXT NOT NULL, reason TEXT NOT NULL, category TEXT NOT NULL DEFAULT 'other', reply TEXT NOT NULL DEFAULT '', answered BOOLEAN NOT NULL DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
			`INSERT INTO reports_new(id,user_id,post_id,original_post_id,title,reason,created_at) SELECT r.id,r.user_id,r.post_id,r.post_id,p.title,r.reason,r.created_at FROM reports r JOIN posts p ON p.id=r.post_id`,
			`DROP TABLE reports`, `ALTER TABLE reports_new RENAME TO reports`,
		} {
			if _, err := tx.Exec(q); err != nil {
				return err
			}
		}
	}
	if _, err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS unanswered_reports ON reports(user_id,original_post_id) WHERE answered=0`); err != nil {
		return err
	}
	return tx.Commit()
}
