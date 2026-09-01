package domain

import "time"

// RiderRegistration is the transactional input for creating a rider plus an
// optional battery/loan relationship (docs/spec.md §4). The API layer builds
// it from the request; the persistence layer consumes it atomically.
//
// Battery and Loan are expected to be provided together (or both nil): a loan
// without a battery, or a battery without a loan, cannot be scored and is
// rejected before it reaches this struct.
type RiderRegistration struct {
	ExternalRef *string
	Battery     *BatteryRegistration
	Loan        *LoanRegistration
}

// BatteryRegistration is the subset of the `batteries` table a partner can
// supply when registering a rider.
type BatteryRegistration struct {
	ExternalRef     *string
	Manufacturer    *string
	RatedCapacityWh *float64
	CommissionedAt  *time.Time
}

// LoanRegistration is the subset of the `loans` table a partner can supply
// when registering a rider. PrincipalKes and BatteryValueKes are required by
// the scoring engine and must be set for a linked (scoreable) registration.
type LoanRegistration struct {
	ExternalRef         *string
	PrincipalKes        *float64
	BatteryValueKes     *float64
	TermMonths          *int
	DailyInstallmentKes *float64
	StartedAt           *time.Time
}

// RiderSummary is the API-facing view of a rider plus its primary (most
// recent) battery/loan relationship and its latest score, when they exist.
// Battery/Loan/Score are pointers so a "never scored" or "identity only"
// rider can be represented without fake zero values.
type RiderSummary struct {
	Rider   Rider
	Battery *Battery
	Loan    *Loan
	Score   *Score
}
