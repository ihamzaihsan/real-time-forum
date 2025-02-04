package handlers

import (
    "RTF/database"
    "encoding/json"
    "net/http"
)

func ServeCategories(w http.ResponseWriter, r *http.Request) {
    rows, err := database.DBInstance.DB.Query("SELECT name FROM categories")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var categories []string
    for rows.Next() {
        var name string
        if err := rows.Scan(&name); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        categories = append(categories, name)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(categories)
}
