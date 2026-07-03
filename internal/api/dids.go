package api

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type backlinkDidsResponse struct {
	Total uint64 `json:"total"`
	LinkingDids []string `json:"linking_dids"`
	Cursor *string `json:"cursor"`
}

func (s *Server) GetBacklinkDids(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	subject := query.Get("subject")
	source := query.Get("source")

	collection, path, err := parseSource(source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}


	limit := uint64(100)
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || parsed == 0 || parsed > 1000 {
			http.Error(w, "limit must be a number between 1 and 1000", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	after := ""
	if raw := query.Get("cursor"); raw != "" {
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}
		after = string(decoded)
	}

	total, dids, err := s.Store.DistinctBacklinkDids(r.Context(), subject, collection, path, after, limit)
	if err != nil {
		log.Println("distinct backlink dids:", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var cursor *string
	if uint64(len(dids)) == limit {
		encoded := base64.StdEncoding.EncodeToString([]byte(dids[len(dids)-1]))
		cursor = &encoded
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backlinkDidsResponse{Total: total, LinkingDids: dids, Cursor: cursor})
}
