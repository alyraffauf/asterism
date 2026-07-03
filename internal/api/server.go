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

	return http.ListenAndServe(addr, mux)
}
