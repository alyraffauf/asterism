package api

import (
	"net/http"

	"github.com/alyraffauf/asterism/internal/store"
)

type Server struct {
	Store *store.Store
}

func (s *Server) Run(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /xrpc/blue.microcosm.links.getBacklinksCount", s.GetBacklinksCount)
	mux.HandleFunc("GET /xrpc/blue.microcosm.links.getBacklinkDids", s.GetBacklinkDids)
	mux.HandleFunc("GET /xrpc/blue.microcosm.links.getBacklinks", s.GetBacklinks)
	mux.HandleFunc("GET /xrpc/blue.microcosm.links.getManyToMany", s.GetManyToMany)
	mux.HandleFunc("GET /xrpc/blue.microcosm.links.getManyToManyCounts", s.GetManyToManyCounts)

	return http.ListenAndServe(addr, withCORS(mux))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}
