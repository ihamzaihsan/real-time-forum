package handlers

import (
	"net/http"
	"path/filepath"
)

func serveShell(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("frontend", "index.html"))
}
