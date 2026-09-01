package api

import "net/http"

type dependency struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type mlDependency struct {
	Status               string `json:"status"`
	BatteryModelLoaded   bool   `json:"battery_model_loaded"`
	RepaymentModelLoaded bool   `json:"repayment_model_loaded"`
	Error                string `json:"error,omitempty"`
}

type readyResponse struct {
	Status    string       `json:"status"`
	Postgres  dependency   `json:"postgres"`
	MLSidecar mlDependency `json:"ml_sidecar"`
}

// handleHealthz is liveness: always 200 if the process is responding.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReadyz reports per-dependency readiness (Postgres + ML sidecar),
// mirroring the Flask side's /readyz pattern of returning component status
// rather than a single boolean.
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := readyResponse{Status: "ready"}

	if s.cfg.PingDB != nil {
		if err := s.cfg.PingDB(ctx); err != nil {
			resp.Status = "not_ready"
			resp.Postgres = dependency{Status: "unavailable", Error: err.Error()}
		} else {
			resp.Postgres = dependency{Status: "ok"}
		}
	} else {
		resp.Status = "not_ready"
		resp.Postgres = dependency{Status: "unavailable", Error: "postgres ping not configured"}
	}

	if s.cfg.MLReady != nil {
		st, err := s.cfg.MLReady(ctx)
		if err != nil {
			resp.Status = "not_ready"
			resp.MLSidecar = mlDependency{Status: "unavailable", Error: err.Error()}
		} else {
			resp.MLSidecar = mlDependency{
				Status:               "ok",
				BatteryModelLoaded:   st.BatteryModelLoaded,
				RepaymentModelLoaded: st.RepaymentModelLoaded,
			}
			if !st.BatteryModelLoaded || !st.RepaymentModelLoaded {
				resp.Status = "not_ready"
			}
		}
	} else {
		resp.Status = "not_ready"
		resp.MLSidecar = mlDependency{Status: "unavailable", Error: "ml sidecar not configured"}
	}

	code := http.StatusOK
	if resp.Status != "ready" {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, resp)
}
