package main

import (
	database "RTF/database"
	routes "RTF/routes"
	"fmt"
	"log"
	"net/http"
	"os"
)

func serveStaticFiles() {
	http.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./frontend/favicon.svg")
	})
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./frontend/css"))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("./frontend/js"))))
}

func main() {

	err := database.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize the database: %v", err)
	}
	defer func() {
		if err := database.DBInstance.DB.Close(); err != nil {
			log.Fatal("Error closing the database:", err)
		}
	}()

	serveStaticFiles()

	routes.InitRoutes()

	portNumber := os.Getenv("PORT")
	if portNumber == "" {
		portNumber = "8080"
	}
	port := ":" + portNumber
	fmt.Println("Server started at http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
