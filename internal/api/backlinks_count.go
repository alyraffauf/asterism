package api

import (
	"encoding/json"
	"log"
	"net/http"
)

type backlinksCountResponse struct {
	Total uint64 `json:"total"`
}

func (s *Server) GetBacklinksCount(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	subject := query.Get("subject")
	source := query.Get("source")

	collection, path, err := parseSource(source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	total, err := s.Store.CountBacklinks(r.Context(), subject, collection, path)
	if err != nil {
		log.Println("count backlinks:", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backlinksCountResponse{Total: total})
}
