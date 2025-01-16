package handlers

import (
	"net/http"
	"path/filepath"
)

// serveMainPage handles serving the main HTML page for the SPA
func serveMainPage(w http.ResponseWriter, r *http.Request) {
	// Serve the index.html as the main page for SPA
	http.ServeFile(w, r, filepath.Join("frontend", "index.html"))

}
