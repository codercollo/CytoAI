package domain

import "time"

// Rider mirrors the `riders` table (docs/spec.md §4).
type Rider struct {
	ID          string     `db:"id"`
	ExternalRef *string    `db:"external_ref"`
	PartnerID   string     `db:"partner_id"`
	OnboardedAt *time.Time `db:"onboarded_at"`
}

// Loan mirrors the `loans` table (docs/spec.md §4).
type Loan struct {
	ID                  string     `db:"id" json:"id,omitempty"`
	ExternalRef         *string    `db:"external_ref" json:"external_ref,omitempty"`
	RiderID             string     `db:"rider_id" json:"rider_id"`
	BatteryID           *string    `db:"battery_id" json:"battery_id,omitempty"`
	PrincipalKes        *float64   `db:"principal_kes" json:"principal_kes,omitempty"`
	BatteryValueKes     *float64   `db:"battery_value_kes" json:"battery_value_kes,omitempty"`
	TermMonths          *int       `db:"term_months" json:"term_months,omitempty"`
	DailyInstallmentKes *float64   `db:"daily_installment_kes" json:"daily_installment_kes,omitempty"`
	StartedAt           *time.Time `db:"started_at" json:"started_at,omitempty"`
}

// RepaymentEvent mirrors the `repayment_events` table (docs/spec.md §4).
type RepaymentEvent struct {
	ID            int64      `db:"id" json:"id,omitempty"`
	LoanID        string     `db:"loan_id" json:"loan_id"`
	DueDate       time.Time  `db:"due_date" json:"due_date"`
	PaidDate      *time.Time `db:"paid_date" json:"paid_date,omitempty"`
	AmountDueKes  *float64   `db:"amount_due_kes" json:"amount_due_kes,omitempty"`
	AmountPaidKes *float64   `db:"amount_paid_kes" json:"amount_paid_kes,omitempty"`
	Status        *string    `db:"status" json:"status,omitempty"`
}

// RiderBatteryPair is a distinct rider/battery pair that has at least one
// loan. It is used to enumerate everything a partner can (re)score.
type RiderBatteryPair struct {
	RiderID   string `json:"rider_id"`
	BatteryID string `json:"battery_id"`
}
