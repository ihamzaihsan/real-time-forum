package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
)

// Serve the static files (HTML, CSS, JS)
func serveStaticFiles() {
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./frontend/assets"))))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./frontend/css"))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("./frontend/js"))))
}
// Serve the main HTML page (the entry point of the SPA)
func serveMainPage(w http.ResponseWriter, r *http.Request) {
	// Serve the index.html as the main page
	http.ServeFile(w, r, filepath.Join("frontend", "index.html"))

}

// Main function to start the web server
func main() {
	// Serve static files (CSS, JS, assets like images)
	serveStaticFiles()

	// Route the main page (entry point of the SPA)
	http.HandleFunc("/", serveMainPage)
    http.HandleFunc("/register", serveMainPage)

	// Start the server
	port := ":8080"
	fmt.Println("Server started at http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
