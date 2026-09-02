// Package api implements the REST surface from docs/spec.md §6.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/codercollo/cytoai/internal/domain"
	"github.com/codercollo/cytoai/internal/risk"
)

// Scorer is the subset of *risk.Scorer the score handlers need.
type Scorer interface {
	Score(ctx context.Context, battery risk.BatteryFeatures, repayment risk.RepaymentFeatures) (*risk.Result, error)
}

// Store is the subset of *queries.Store the handlers need.
type Store interface {
	InsertTelemetryReadings(ctx context.Context, rows []domain.TelemetryReading) (int64, error)
	InsertRepaymentEvents(ctx context.Context, rows []domain.RepaymentEvent) (int64, error)
	InsertSwapEvents(ctx context.Context, events []domain.SwapEvent) (int64, error)
	InsertScore(ctx context.Context, s domain.Score) (int64, error)
	LatestScore(ctx context.Context, riderID string) (domain.Score, error)
	PortfolioScores(ctx context.Context, partnerID string, limit, offset int) ([]domain.Score, error)
	TelemetryReadingsByBattery(ctx context.Context, batteryID string) ([]domain.TelemetryReading, error)
	RepaymentEventsByLoan(ctx context.Context, loanID string) ([]domain.RepaymentEvent, error)
	LoanByRiderAndBattery(ctx context.Context, riderID, batteryID string) (domain.Loan, error)
	LatestLoanForRider(ctx context.Context, riderID string) (domain.Loan, error)
	SwapEventsByRider(ctx context.Context, riderID string) ([]domain.SwapEvent, error)
	SwapEventsByPartner(ctx context.Context, partnerID string) ([]domain.SwapEvent, error)
	ResolveBatteryID(ctx context.Context, partnerID, ref string) (string, error)
	ResolveLoanID(ctx context.Context, partnerID, ref string) (string, error)
	ResolveRiderID(ctx context.Context, partnerID, ref string) (string, error)
	ResolveBatteryIDByExternalRef(ctx context.Context, ref string) (string, error)
	RiderBelongsToPartner(ctx context.Context, partnerID, riderID string) (bool, error)
	RiderBatteryPairsForPartner(ctx context.Context, partnerID string) ([]domain.RiderBatteryPair, error)
	SwapNetworkRiderIDsForPartner(ctx context.Context, partnerID string) ([]string, error)
	ListRiders(ctx context.Context, partnerID string, limit, offset int) ([]domain.RiderSummary, error)
	RiderByID(ctx context.Context, partnerID, riderID string) (domain.RiderSummary, error)
	CreateRider(ctx context.Context, partnerID string, reg domain.RiderRegistration) (domain.RiderSummary, error)
}

// Authenticator is the subset of *auth.Authenticator middleware needs.
type Authenticator interface {
	Authenticate(ctx context.Context, providedKey string) (string, error)
}

// AnomalyDetector reports fraud/anomaly flags for raw ingested records. It is
// kept separate from Scorer because anomalies are a confidence signal, not a
// score input (spec.md §9: no autonomous decisions).
type AnomalyDetector interface {
	DetectTelemetryAnomalies(ctx context.Context, records []risk.TelemetryAnomalyRecord) ([]risk.AnomalyFlag, error)
	DetectRepaymentAnomalies(ctx context.Context, records []risk.RepaymentAnomalyRecord) ([]risk.AnomalyFlag, error)
	DetectSwapAnomalies(ctx context.Context, records []risk.SwapAnomalyRecord) ([]risk.AnomalyFlag, error)
}

// Config wires a Server's dependencies.
type Config struct {
	Store   Store
	Scorer  Scorer
	Auth    Authenticator
	Logger  *slog.Logger
	PingDB  func(ctx context.Context) error
	MLReady func(ctx context.Context) (*risk.ReadyStatus, error)
	// AnomalyDetector is optional; when nil, anomaly checks are skipped.
	AnomalyDetector AnomalyDetector
	// CORSAllowedOrigins is the browser-origin allowlist (preflight + responses).
	CORSAllowedOrigins []string
}

// Server holds the dependencies shared by all handlers.
type Server struct {
	cfg     Config
	logger  *slog.Logger
	limiter *rateLimiter
}

// NewServer builds a Server, applying defaults where needed.
func NewServer(cfg Config) *Server {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		cfg:     cfg,
		logger:  logger,
		limiter: newRateLimiter(5, 10), // MVP: 5 req/s, burst 10, per partner
	}
}

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyPartnerID
)

func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(ctxKeyRequestID).(string)
	return id
}

func partnerIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(ctxKeyPartnerID).(string)
	return id
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string, details any) {
	if details == nil {
		details = map[string]any{}
	}
	writeJSON(w, status, map[string]any{
		"error":   code,
		"message": msg,
		"details": details,
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
