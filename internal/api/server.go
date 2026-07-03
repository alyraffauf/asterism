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

	return http.ListenAndServe(addr, mux)
}
