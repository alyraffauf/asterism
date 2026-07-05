package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type getMiniDidDocResponse struct {
	Did        string `json:"did"`
	Handle     string `json:"handle"`
	Pds        string `json:"pds"`
	SigningKey string `json:"signing_key"`
}

func (s *Server) GetMiniDidDoc(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	identifier := query.Get("identifier")
	if identifier == "" {
		http.Error(w, "identifier is required", http.StatusBadRequest)
		return
	}

	atID, err := syntax.ParseAtIdentifier(identifier)
	if err != nil {
		http.Error(w, "invalid identifier", http.StatusBadRequest)
		return
	}

	resolveCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	identity, err := s.Directory.Lookup(resolveCtx, atID)
	if err != nil {
		s.Logger.Error("resolve mini doc", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var signingKey string
	if key, ok := identity.Keys["atproto"]; ok {
		signingKey = key.PublicKeyMultibase
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(getMiniDidDocResponse{
		Did:        identity.DID.String(),
		Handle:     identity.Handle.String(),
		Pds:        identity.PDSEndpoint(),
		SigningKey: signingKey,
	})
}
