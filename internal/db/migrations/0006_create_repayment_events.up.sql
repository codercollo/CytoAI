CREATE TABLE repayment_events (
    id BIGSERIAL PRIMARY KEY,
    loan_id UUID NOT NULL REFERENCES loans(id),
    due_date DATE NOT NULL,
    paid_date DATE,
    amount_due_kes NUMERIC,
    amount_paid_kes NUMERIC,
    status TEXT CHECK (status IN ('on_time','late','missed','partial'))
);
