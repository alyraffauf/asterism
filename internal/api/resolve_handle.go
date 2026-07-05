package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (s *Server) ResolveHandle(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	rawHandle := query.Get("handle")
	if rawHandle == "" {
		http.Error(w, "handle is required", http.StatusBadRequest)
		return
	}

	handle, err := syntax.ParseHandle(rawHandle)
	if err != nil {
		http.Error(w, "invalid handle", http.StatusBadRequest)
		return
	}

	resolveCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	identity, err := s.Directory.LookupHandle(resolveCtx, handle)
	if err != nil {
		s.Logger.Error("resolve handle", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(atproto.IdentityResolveHandle_Output{
		Did: identity.DID.String(),
	})
}
