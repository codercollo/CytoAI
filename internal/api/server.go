package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes builds the chi router with the full middleware chain and all handler
// groups.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(s.requestID)
	r.Use(s.logging)
	r.Use(s.cors)

	// Public health endpoints — no auth.
	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)

	// Authenticated + rate-limited API surface (docs/spec.md §6).
	r.Route("/v1", func(r chi.Router) {
		r.Use(s.auth)
		r.Use(s.rateLimit)

		r.Post("/telematics", s.handleIngestTelematics)
		r.Post("/repayments", s.handleIngestRepayments)
		r.Post("/swaps", s.handleIngestSwaps)
		r.Post("/score", s.handleScoreCompute)
		r.Post("/score/rescore-all", s.handleRescoreAll)
		r.Get("/score/{rider_id}", s.handleScoreGet)
		r.Get("/portfolio", s.handlePortfolio)
		r.Get("/operator/stress-flags", s.handleOperatorStressFlags)

		r.Get("/riders", s.handleRiderList)
		r.Post("/riders", s.handleRiderCreate)
		r.Get("/riders/{rider_id}", s.handleRiderGet)
	})

	return r
}
