package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type backlinkDidsResponse struct {
	Total uint64 `json:"total"`
	LinkingDids []string `json:"linking_dids"`
}

func (s *Server) GetBacklinkDids(w http.ResponseWriter, r *http.Request) {
	subject := r.URL.Query().Get("subject")
	source := r.URL.Query().Get("source")

	collection, path, err := parseSource(source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit := uint64(100)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || parsed == 0 || parsed > 1000 {
			http.Error(w, "limit must be a number between 1 and 1000", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	total, dids, err := s.Store.DistinctBacklinkDids(r.Context(), subject, collection, path, limit)
	if err != nil {
		log.Println("distinct backlink dids:", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backlinkDidsResponse{Total: total, LinkingDids: dids})
}
