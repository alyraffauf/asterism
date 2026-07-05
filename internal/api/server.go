package api

import (
	"log/slog"
	"net/http"

	"github.com/alyraffauf/asterism/internal/store"
	"github.com/bluesky-social/indigo/atproto/identity"
)

type Server struct {
	Store     *store.Store
	Directory identity.Directory
	Logger    *slog.Logger
}

func (s *Server) Run(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.Health)

	mux.HandleFunc("GET /xrpc/blue.microcosm.links.getBacklinksCount", s.GetBacklinksCount)
	mux.HandleFunc("GET /xrpc/blue.microcosm.links.getBacklinkDids", s.GetBacklinkDids)
	mux.HandleFunc("GET /xrpc/blue.microcosm.links.getBacklinks", s.GetBacklinks)
	mux.HandleFunc("GET /xrpc/blue.microcosm.links.getManyToMany", s.GetManyToMany)
	mux.HandleFunc("GET /xrpc/blue.microcosm.links.getManyToManyCounts", s.GetManyToManyCounts)

	mux.HandleFunc("GET /xrpc/blue.microcosm.identity.resolveMiniDoc", s.GetMiniDidDoc)

	mux.HandleFunc("GET /xrpc/com.atproto.identity.resolveHandle", s.ResolveHandle)

	return http.ListenAndServe(addr, withCORS(mux))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}
