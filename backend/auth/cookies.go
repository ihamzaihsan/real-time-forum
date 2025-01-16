package cookies

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

func SetLoginCookie(w http.ResponseWriter, sessionToken string) {
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true, // Ensures the cookie is sent only over HTTPS
		SameSite: http.SameSiteStrictMode, // Prevents CSRF attacks
	}

	http.SetCookie(w, cookie)
}

func GenerateUUIDToken() string {
	return uuid.NewString()
}

// may be we will use them in the future.
func DeleteSessionCookie(w http.ResponseWriter) {
    cookie := &http.Cookie{
        Name:     "session_token",
        Value:    "",
        Path:     "/",
        Expires:  time.Now().Add(-1 * time.Hour), // Expire immediately
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    }
    http.SetCookie(w, cookie)
}

func GetSessionToken(r *http.Request) (string, error) {
    cookie, err := r.Cookie("session_token")
    if err != nil {
        return "", err
    }
    return cookie.Value, nil
}
