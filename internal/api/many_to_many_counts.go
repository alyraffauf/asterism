package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alyraffauf/asterism/internal/store"
)

type manyToManyCountsResponse struct {
	CountsByOtherSubject []store.OtherSubjectCount `json:"counts_by_other_subject"`
	Cursor               *string                   `json:"cursor"`
}

func (s *Server) GetManyToManyCounts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	subject := query.Get("subject")
	source := query.Get("source")
	linkDids := query["did"]
	otherSubjects := query["otherSubject"]

	collection, path, err := parseSource(source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pathToOther, err := normalizePath(query.Get("pathToOther"))
	if err != nil {
		http.Error(w, "pathToOther: "+err.Error(), http.StatusBadRequest)
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

	counts, err := s.Store.ManyToManyCounts(r.Context(), subject, collection, path, pathToOther, linkDids, otherSubjects, after, limit)
	if err != nil {
		s.Logger.Error("many to many counts", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var cursor *string
	if uint64(len(counts)) == limit {
		encoded := base64.StdEncoding.EncodeToString([]byte(counts[len(counts)-1].Subject))
		cursor = &encoded
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manyToManyCountsResponse{CountsByOtherSubject: counts, Cursor: cursor})
}
