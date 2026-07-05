package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alyraffauf/asterism/internal/store"
)

type getBacklinksResponse struct {
	Total   uint64         `json:"total"`
	Records []store.Record `json:"records"`
	Cursor  *string        `json:"cursor"`
}

func (s *Server) GetBacklinks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	subject := query.Get("subject")
	source := query.Get("source")
	dids := query["did"]

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

	reverse := query.Get("reverse") == "true"

	var after int64
	if raw := query.Get("cursor"); raw != "" {
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}
		after, err = strconv.ParseInt(string(decoded), 10, 64)
		if err != nil {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}
	}

	total, records, err := s.Store.ListBacklinks(r.Context(), subject, collection, path, dids, reverse, after, limit)
	if err != nil {
		s.Logger.Error("list backlinks", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var cursor *string
	if uint64(len(records)) == limit {
		last := records[len(records)-1]
		encoded := base64.StdEncoding.EncodeToString([]byte(strconv.FormatInt(last.ID, 10)))
		cursor = &encoded
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(getBacklinksResponse{Total: total, Records: records, Cursor: cursor})
}
