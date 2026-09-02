package api

import (
	"context"
	"errors"
	"time"

	"github.com/codercollo/cytoai/internal/domain"
	"github.com/codercollo/cytoai/internal/risk"
)

type mockStore struct {
	insertTelemetry       func(context.Context, []domain.TelemetryReading) (int64, error)
	insertRepayment       func(context.Context, []domain.RepaymentEvent) (int64, error)
	insertSwaps           func(context.Context, []domain.SwapEvent) (int64, error)
	insertScore           func(context.Context, domain.Score) (int64, error)
	latestScore           func(context.Context, string) (domain.Score, error)
	portfolio             func(context.Context, string, int, int) ([]domain.Score, error)
	telemetryByBattery    func(context.Context, string) ([]domain.TelemetryReading, error)
	repaymentByLoan       func(context.Context, string) ([]domain.RepaymentEvent, error)
	loanByRiderAndBattery func(context.Context, string, string) (domain.Loan, error)
	latestLoanForRider    func(context.Context, string) (domain.Loan, error)
	swapEventsByRider     func(context.Context, string) ([]domain.SwapEvent, error)
	swapEventsByPartner   func(context.Context, string) ([]domain.SwapEvent, error)
	resolveBatteryID      func(context.Context, string, string) (string, error)
	resolveLoanID         func(context.Context, string, string) (string, error)
	resolveRiderID        func(context.Context, string, string) (string, error)
	resolveBatteryByRef   func(context.Context, string) (string, error)
	riderBatteryPairs     func(context.Context, string) ([]domain.RiderBatteryPair, error)
	swapNetworkRiderIDs   func(context.Context, string) ([]string, error)
	listRiders            func(context.Context, string, int, int) ([]domain.RiderSummary, error)
	riderByID             func(context.Context, string, string) (domain.RiderSummary, error)
	createRider           func(context.Context, string, domain.RiderRegistration) (domain.RiderSummary, error)
}

func (m *mockStore) InsertTelemetryReadings(ctx context.Context, rows []domain.TelemetryReading) (int64, error) {
	if m.insertTelemetry != nil {
		return m.insertTelemetry(ctx, rows)
	}
	return int64(len(rows)), nil
}

func (m *mockStore) InsertRepaymentEvents(ctx context.Context, rows []domain.RepaymentEvent) (int64, error) {
	if m.insertRepayment != nil {
		return m.insertRepayment(ctx, rows)
	}
	return int64(len(rows)), nil
}

func (m *mockStore) InsertSwapEvents(ctx context.Context, events []domain.SwapEvent) (int64, error) {
	if m.insertSwaps != nil {
		return m.insertSwaps(ctx, events)
	}
	return int64(len(events)), nil
}

func (m *mockStore) InsertScore(ctx context.Context, s domain.Score) (int64, error) {
	if m.insertScore != nil {
		return m.insertScore(ctx, s)
	}
	return 1, nil
}

func (m *mockStore) LatestScore(ctx context.Context, riderID string) (domain.Score, error) {
	if m.latestScore != nil {
		return m.latestScore(ctx, riderID)
	}
	return domain.Score{}, errors.New("not found")
}

func (m *mockStore) PortfolioScores(ctx context.Context, partnerID string, limit, offset int) ([]domain.Score, error) {
	if m.portfolio != nil {
		return m.portfolio(ctx, partnerID, limit, offset)
	}
	return nil, nil
}

func (m *mockStore) TelemetryReadingsByBattery(ctx context.Context, batteryID string) ([]domain.TelemetryReading, error) {
	if m.telemetryByBattery != nil {
		return m.telemetryByBattery(ctx, batteryID)
	}
	return nil, nil
}

func (m *mockStore) RepaymentEventsByLoan(ctx context.Context, loanID string) ([]domain.RepaymentEvent, error) {
	if m.repaymentByLoan != nil {
		return m.repaymentByLoan(ctx, loanID)
	}
	return nil, nil
}

func (m *mockStore) LoanByRiderAndBattery(ctx context.Context, riderID, batteryID string) (domain.Loan, error) {
	if m.loanByRiderAndBattery != nil {
		return m.loanByRiderAndBattery(ctx, riderID, batteryID)
	}
	return domain.Loan{}, errors.New("not found")
}

func (m *mockStore) LatestLoanForRider(ctx context.Context, riderID string) (domain.Loan, error) {
	if m.latestLoanForRider != nil {
		return m.latestLoanForRider(ctx, riderID)
	}
	return domain.Loan{}, errors.New("not found")
}

func (m *mockStore) SwapEventsByRider(ctx context.Context, riderID string) ([]domain.SwapEvent, error) {
	if m.swapEventsByRider != nil {
		return m.swapEventsByRider(ctx, riderID)
	}
	return nil, nil
}

func (m *mockStore) SwapEventsByPartner(ctx context.Context, partnerID string) ([]domain.SwapEvent, error) {
	if m.swapEventsByPartner != nil {
		return m.swapEventsByPartner(ctx, partnerID)
	}
	return nil, nil
}

func (m *mockStore) ResolveBatteryID(ctx context.Context, partnerID, ref string) (string, error) {
	if m.resolveBatteryID != nil {
		return m.resolveBatteryID(ctx, partnerID, ref)
	}
	return ref, nil
}

func (m *mockStore) ResolveLoanID(ctx context.Context, partnerID, ref string) (string, error) {
	if m.resolveLoanID != nil {
		return m.resolveLoanID(ctx, partnerID, ref)
	}
	return ref, nil
}

func (m *mockStore) ResolveRiderID(ctx context.Context, partnerID, ref string) (string, error) {
	if m.resolveRiderID != nil {
		return m.resolveRiderID(ctx, partnerID, ref)
	}
	return ref, nil
}

func (m *mockStore) ResolveBatteryIDByExternalRef(ctx context.Context, ref string) (string, error) {
	if m.resolveBatteryByRef != nil {
		return m.resolveBatteryByRef(ctx, ref)
	}
	return ref, nil
}

func (m *mockStore) RiderBelongsToPartner(ctx context.Context, partnerID, riderID string) (bool, error) {
	return true, nil
}

func (m *mockStore) RiderBatteryPairsForPartner(ctx context.Context, partnerID string) ([]domain.RiderBatteryPair, error) {
	if m.riderBatteryPairs != nil {
		return m.riderBatteryPairs(ctx, partnerID)
	}
	return nil, nil
}

func (m *mockStore) SwapNetworkRiderIDsForPartner(ctx context.Context, partnerID string) ([]string, error) {
	if m.swapNetworkRiderIDs != nil {
		return m.swapNetworkRiderIDs(ctx, partnerID)
	}
	return nil, nil
}

func (m *mockStore) ListRiders(ctx context.Context, partnerID string, limit, offset int) ([]domain.RiderSummary, error) {
	if m.listRiders != nil {
		return m.listRiders(ctx, partnerID, limit, offset)
	}
	return nil, nil
}

func (m *mockStore) RiderByID(ctx context.Context, partnerID, riderID string) (domain.RiderSummary, error) {
	if m.riderByID != nil {
		return m.riderByID(ctx, partnerID, riderID)
	}
	return domain.RiderSummary{}, errors.New("not found")
}

func (m *mockStore) CreateRider(ctx context.Context, partnerID string, reg domain.RiderRegistration) (domain.RiderSummary, error) {
	if m.createRider != nil {
		return m.createRider(ctx, partnerID, reg)
	}
	return domain.RiderSummary{}, nil
}

type mockScorer struct {
	score func(context.Context, risk.BatteryFeatures, risk.RepaymentFeatures) (*risk.Result, error)
}

func (m mockScorer) Score(ctx context.Context, b risk.BatteryFeatures, r risk.RepaymentFeatures) (*risk.Result, error) {
	if m.score != nil {
		return m.score(ctx, b, r)
	}
	return &risk.Result{CytoScore: 50}, nil
}

type mockAuth struct {
	authenticate func(context.Context, string) (string, error)
}

func (m mockAuth) Authenticate(ctx context.Context, key string) (string, error) {
	if m.authenticate != nil {
		return m.authenticate(ctx, key)
	}
	return "partner-1", nil
}

func newTestServer(store Store, scorer Scorer, authn Authenticator) *Server {
	return NewServer(Config{Store: store, Scorer: scorer, Auth: authn})
}

func fptr(v float64) *float64 { return &v }
func iptr(v int) *int         { return &v }
func sptr(v string) *string   { return &v }

func validScoreStore() *mockStore {
	return &mockStore{
		loanByRiderAndBattery: func(ctx context.Context, riderID, batteryID string) (domain.Loan, error) {
			return domain.Loan{ID: "loan-1", PrincipalKes: fptr(300000), BatteryValueKes: fptr(150000)}, nil
		},
		telemetryByBattery: func(ctx context.Context, batteryID string) ([]domain.TelemetryReading, error) {
			return []domain.TelemetryReading{{
				BatteryID:        batteryID,
				ReadingAt:        time.Now().UTC(),
				DepthOfDischarge: fptr(60),
				TemperatureC:     fptr(27),
				Voltage:          fptr(52),
				CycleCount:       iptr(5),
			}}, nil
		},
		repaymentByLoan: func(ctx context.Context, loanID string) ([]domain.RepaymentEvent, error) {
			return []domain.RepaymentEvent{{
				LoanID:  loanID,
				DueDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				Status:  sptr("on_time"),
			}}, nil
		},
		insertScore: func(ctx context.Context, s domain.Score) (int64, error) { return 1, nil },
	}
}
