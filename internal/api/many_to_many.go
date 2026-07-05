package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alyraffauf/asterism/internal/store"
)

type manyToManyResponse struct {
	Total  uint64                 `json:"total"`
	Items  []store.ManyToManyItem `json:"items"`
	Cursor *string                `json:"cursor"`
}

func (s *Server) GetManyToMany(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	subject := query.Get("subject")
	source := query.Get("source")
	linkDids := query["linkDid"]
	otherSubjects := query["otherSubject"]

	collection, path, err := parseSource(source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pathToOther, err := normalizePath(query.Get("pathToOther"))
	if err != nil {
		http.Error(w, "path_to_other: "+err.Error(), http.StatusBadRequest)
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

	total, items, err := s.Store.ManyToMany(r.Context(), subject, collection, path, pathToOther, linkDids, otherSubjects, after, limit)
	if err != nil {
		s.Logger.Error("many to many", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var cursor *string
	if uint64(len(items)) == limit {
		last := items[len(items)-1]
		encoded := base64.StdEncoding.EncodeToString([]byte(strconv.FormatInt(last.LinkRecord.ID, 10)))
		cursor = &encoded
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manyToManyResponse{Total: total, Items: items, Cursor: cursor})
}
