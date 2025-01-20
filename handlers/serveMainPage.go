package handlers

import (
	"net/http"
	"path/filepath"
)


func ServeMainPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("frontend", "index.html"))
}
