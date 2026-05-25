package handlers

import "net/http"

func ServeCommentLike(w http.ResponseWriter, r *http.Request) { serveReaction(w, r, "comment") }
