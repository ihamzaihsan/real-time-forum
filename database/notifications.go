package database

import "database/sql"

// SQLite triggers make alerts atomic with the associated change, including writes
// from moderation. Existing history is deliberately not backfilled.
func migrateNotifications(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, query := range []string{
		`CREATE TABLE IF NOT EXISTS notifications (id INTEGER PRIMARY KEY AUTOINCREMENT, recipient_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, actor_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE, comment_id INTEGER REFERENCES comments(id) ON DELETE CASCADE, kind TEXT NOT NULL, is_read BOOLEAN NOT NULL DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE INDEX IF NOT EXISTS notifications_inbox ON notifications(recipient_id,is_read,id DESC)`,
		`CREATE TRIGGER IF NOT EXISTS notify_post_reaction_insert AFTER INSERT ON likes WHEN NEW.post_id IS NOT NULL BEGIN
 INSERT INTO notifications(recipient_id,actor_id,post_id,kind) SELECT user_id,NEW.user_id,id,CASE WHEN NEW.is_like THEN 'like' ELSE 'dislike' END FROM posts WHERE id=NEW.post_id AND user_id!=NEW.user_id AND status='published'; END`,
		`CREATE TRIGGER IF NOT EXISTS notify_post_reaction_change AFTER UPDATE OF is_like ON likes WHEN NEW.post_id IS NOT NULL AND OLD.is_like!=NEW.is_like BEGIN
 INSERT INTO notifications(recipient_id,actor_id,post_id,kind) SELECT user_id,NEW.user_id,id,CASE WHEN NEW.is_like THEN 'like' ELSE 'dislike' END FROM posts WHERE id=NEW.post_id AND user_id!=NEW.user_id AND status='published'; END`,
		`CREATE TRIGGER IF NOT EXISTS notify_comment_insert AFTER INSERT ON comments WHEN NEW.status='published' BEGIN
 INSERT INTO notifications(recipient_id,actor_id,post_id,comment_id,kind) SELECT user_id,NEW.user_id,id,NEW.id,'comment' FROM posts WHERE id=NEW.post_id AND user_id!=NEW.user_id AND status='published'; END`,
		`CREATE TRIGGER IF NOT EXISTS notify_comment_approval AFTER UPDATE OF status ON comments WHEN OLD.status!='published' AND NEW.status='published' BEGIN
 INSERT INTO notifications(recipient_id,actor_id,post_id,comment_id,kind) SELECT user_id,NEW.user_id,id,NEW.id,'comment' FROM posts WHERE id=NEW.post_id AND user_id!=NEW.user_id AND status='published'; END`,
		`CREATE TRIGGER IF NOT EXISTS hide_comment_notifications AFTER UPDATE OF status ON comments WHEN NEW.status!='published' BEGIN DELETE FROM notifications WHERE comment_id=NEW.id; END`,
		`CREATE TRIGGER IF NOT EXISTS hide_post_notifications AFTER UPDATE OF status ON posts WHEN NEW.status!='published' BEGIN DELETE FROM notifications WHERE post_id=NEW.id; END`,
	} {
		if _, err := tx.Exec(query); err != nil {
			return err
		}
	}
	return tx.Commit()
}
