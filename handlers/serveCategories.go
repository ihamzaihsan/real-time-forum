package handlers

import (
	"RTF/database"
	"net/http"
)

func ServeCategories(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	rows, err := database.DBInstance.DB.Query("SELECT name FROM categories ORDER BY id")
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	categories := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			serverError(w, err)
			return
		}
		categories = append(categories, name)
	}
	if err := rows.Err(); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, categories)
}
