package Forum

import (
    "database/sql"
    "errors"
    "fmt"
    models "forum/models"
    "time"
)

func StoreSession(sessionToken string, email string) error {
    tx, err := DBInstance.DB.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %v", err)
    }
    defer tx.Rollback()

    // Update user's online status
    _, err = tx.Exec("UPDATE users SET is_online = TRUE, last_seen = CURRENT_TIMESTAMP WHERE email = ?", email)
    if err != nil {
        return fmt.Errorf("failed to update user status: %v", err)
    }

    // Delete existing session
    _, err = tx.Exec("DELETE FROM sessions WHERE email = ?", email)
    if err != nil {
        return fmt.Errorf("failed to delete existing session: %v", err)
    }

    expirationTime := time.Now().Add(24 * time.Hour)
    _, err = tx.Exec(
        "INSERT INTO sessions (session_token, email, expires_at, created_at) VALUES (?, ?, ?, ?)",
        sessionToken, email, expirationTime, time.Now(),
    )
    if err != nil {
        return fmt.Errorf("failed to store session: %v", err)
    }

    return tx.Commit()
}

func GetUserBySession(sessionToken string) (*models.User, error) {
    var user models.User

    query := `
        SELECT 
            u.id, 
            u.username, 
            u.email, 
            u.first_name,
            u.last_name,
            u.age,
            u.gender,
            u.is_online,
            u.last_seen,
            u.created_at,
            COALESCE((SELECT COUNT(*) FROM posts WHERE user_id = u.id), 0) AS post_count,
            COALESCE((SELECT COUNT(*) FROM comments WHERE user_id = u.id), 0) AS comment_count
        FROM users u 
        JOIN sessions s ON u.email = s.email 
        WHERE s.session_token = ? AND s.expires_at > CURRENT_TIMESTAMP`

    row := DBInstance.DB.QueryRow(query, sessionToken)
    err := row.Scan(
        &user.ID, 
        &user.Username, 
        &user.Email, 
        &user.FirstName,
        &user.LastName,
        &user.Age,
        &user.Gender,
        &user.IsOnline,
        &user.LastSeen,
        &user.JoinDate,
        &user.PostCount,
        &user.CommentCount,
    )

    if err == sql.ErrNoRows {
        return nil, errors.New("no active session found")
    }
    if err != nil {
        return nil, err
    }

    return &user, nil
}

func EndSession(sessionToken string) error {
    tx, err := DBInstance.DB.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Set user offline
    _, err = tx.Exec(`
        UPDATE users 
        SET is_online = FALSE 
        WHERE email = (SELECT email FROM sessions WHERE session_token = ?)
    `, sessionToken)
    if err != nil {
        return err
    }

    // Remove session
    _, err = tx.Exec("DELETE FROM sessions WHERE session_token = ?", sessionToken)
    if err != nil {
        return err
    }

    return tx.Commit()
}

func CleanExpiredSessions() error {
    _, err := DBInstance.DB.Exec(`
        UPDATE users 
        SET is_online = FALSE 
        WHERE email IN (
            SELECT email 
            FROM sessions 
            WHERE expires_at <= CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        return err
    }

    _, err = DBInstance.DB.Exec("DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP")
    return err
}
