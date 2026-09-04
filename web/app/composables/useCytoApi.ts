export interface AnomalyFlag {
  entity_type: string
  entity_id: string
  rule_or_model: string
  severity: string
  reason: string
}

export interface Score {
  id?: number
  rider_id?: string
  battery_id?: string
  financing_model?: string
  bhi_context?: string
  battery_health_index: number
  repayment_risk_index: number
  cyto_score: number
  scored_at?: string
  model_version?: string
  battery_model_version?: string
  repayment_model_version?: string
  anomaly_flags?: AnomalyFlag[]
}

export interface ScoreRequest {
  rider_id: string
  battery_id?: string
}

export interface ScoreResponse extends Score {
  insufficient_data?: boolean
  reason?: string
}

export interface PortfolioResponse {
  scores: Score[]
  limit: number
  offset: number
}

export interface StressFlagsResponse {
  flags: AnomalyFlag[]
}

export interface RowError {
  row?: number
  field?: string
  value?: string
  error: string
}

export interface ChartPoint {
  t: string
  bhi: number
  rri: number
}

export interface RescoreSkipped {
  rider_id: string
  battery_id?: string
  reason: string
}

export interface RescoreResponse {
  scored: number
  skipped?: RescoreSkipped[] | null
}

export type RegistrationStatus = 'registered' | 'linked'

export interface BatteryInfo {
  id: string
  external_ref?: string
  manufacturer?: string
  rated_capacity_wh?: number
  commissioned_at?: string
}

export interface LoanInfo {
  id: string
  external_ref?: string
  principal_kes?: number
  battery_value_kes?: number
  term_months?: number
  daily_installment_kes?: number
  started_at?: string
  financing_model?: string
}

export interface ScoreSummary {
  id?: number
  battery_id?: string
  battery_health_index: number
  repayment_risk_index: number
  cyto_score: number
  scored_at?: string
  model_version?: string
}

export interface Factors {
  battery: {
    cycle_count: number
    avg_depth_of_discharge: number
    avg_temperature_c: number
    age_days: number
    charge_rate_variance: number
  }
  repayment: {
    on_time_ratio: number
    avg_days_late: number
    payment_cadence_proxy: number
    loan_to_battery_value_ratio: number
    tenure_days: number
    telemetry_cadence_proxy: number
    battery_stress_profile?: number
    financing_model?: string
  }
}

export interface RiderResponse {
  id: string
  external_ref?: string
  onboarded_at?: string
  registration_status: RegistrationStatus
  battery?: BatteryInfo
  loan?: LoanInfo
  latest_score: ScoreSummary | null
  factors?: Factors
  anomaly_flags?: AnomalyFlag[]
}

export interface RiderListResponse {
  riders: RiderResponse[]
  limit: number
  offset: number
}

export interface BatteryCreateRequest {
  external_ref?: string
  manufacturer?: string
  rated_capacity_wh?: number
  commissioned_at?: string
}

export interface LoanCreateRequest {
  external_ref?: string
  principal_kes: number
  battery_value_kes: number
  term_months?: number
  daily_installment_kes?: number
  started_at?: string
}

export interface RiderCreateRequest {
  external_ref: string
  battery?: BatteryCreateRequest
  loan?: LoanCreateRequest
}

export type ApiErrorKind = 'network' | 'auth' | 'validation' | 'server' | 'unknown'

export interface ApiError {
  kind: ApiErrorKind
  status: number
  code: string
  message: string
  details?: Record<string, any>
}

// classifyApiError normalizes any thrown fetch error into a stable shape and
// category. It is the single source of truth for how the app reacts to
// network/auth/validation/server failures (see useCytoRequest).
export function classifyApiError(e: unknown): ApiError {
  const err = (e ?? {}) as any
  const status = Number(err?.status ?? err?.statusCode ?? err?.response?.status ?? 0)
  const body = err?.data
  const code = body && typeof body === 'object' && 'error' in body ? String(body.error) : ''
  const backendMessage =
    body && typeof body === 'object' && 'message' in body ? String(body.message) : ''
  const details =
    body && typeof body === 'object' && 'details' in body && body.details
      ? (body.details as Record<string, any>)
      : undefined

  if (!status) {
    return { kind: 'network', status: 0, code: 'network_error', message: "Can't reach the server.", details }
  }
  if (status === 401 || status === 403) {
    return { kind: 'auth', status, code: code || 'unauthorized', message: 'Check your partner API key.', details }
  }
  if (status === 400 || status === 422 || status === 409) {
    return { kind: 'validation', status, code, message: backendMessage || err?.message || 'Invalid request.', details }
  }
  if (status >= 500) {
    return { kind: 'server', status, code, message: 'Something went wrong on our end.', details }
  }
  return { kind: 'unknown', status, code, message: backendMessage || err?.message || 'Unexpected error.', details }
}

export function useCytoApi() {
  // The browser calls same-origin /v1/* paths; Nitro proxies them to the Go
  // API (see web/server/routes/v1/[...].ts), so no API host is embedded here.
  const base = ''

  function apiKey(): string {
    if (import.meta.client) {
      return localStorage.getItem('cyto_api_key') || ''
    }
    return ''
  }

  function headers(extra: Record<string, string> = {}): Record<string, string> {
    const h: Record<string, string> = { ...extra }
    const key = apiKey()
    if (key) {
      h.Authorization = `Bearer ${key}`
    }
    return h
  }

  return {
    portfolio: (limit = 100, offset = 0) =>
      $fetch<PortfolioResponse>(`${base}/v1/portfolio?limit=${limit}&offset=${offset}`, {
        headers: headers(),
      }),

    riderScore: (riderId: string) =>
      $fetch<Score>(`${base}/v1/score/${riderId}`, { headers: headers() }),

    operatorStressFlags: () =>
      $fetch<StressFlagsResponse>(`${base}/v1/operator/stress-flags`, {
        headers: headers(),
      }),

    uploadTelematics: async (file: File) =>
      $fetch<{ inserted: number }>(`${base}/v1/telematics`, {
        method: 'POST',
        body: await file.text(),
        headers: headers({ 'Content-Type': 'text/csv' }),
      }),

    uploadRepayments: async (file: File) =>
      $fetch<{ inserted: number }>(`${base}/v1/repayments`, {
        method: 'POST',
        body: await file.text(),
        headers: headers({ 'Content-Type': 'text/csv' }),
      }),

    uploadSwaps: async (file: File) =>
      $fetch<{ inserted: number }>(`${base}/v1/swaps`, {
        method: 'POST',
        body: await file.text(),
        headers: headers({ 'Content-Type': 'text/csv' }),
      }),

    computeScore: (riderId: string, batteryId?: string) =>
      $fetch<ScoreResponse>(`${base}/v1/score`, {
        method: 'POST',
        body: JSON.stringify({
          rider_id: riderId,
          ...(batteryId ? { battery_id: batteryId } : {}),
        }),
        headers: headers({ 'Content-Type': 'application/json' }),
      }),

    rescoreAll: () =>
      $fetch<RescoreResponse>(`${base}/v1/score/rescore-all`, {
        method: 'POST',
        headers: headers(),
      }),

    riders: (limit = 100, offset = 0) =>
      $fetch<RiderListResponse>(`${base}/v1/riders?limit=${limit}&offset=${offset}`, {
        headers: headers(),
      }),

    rider: (riderId: string) =>
      $fetch<RiderResponse>(`${base}/v1/riders/${riderId}`, { headers: headers() }),

    createRider: (body: RiderCreateRequest) =>
      $fetch<RiderResponse>(`${base}/v1/riders`, {
        method: 'POST',
        body: JSON.stringify(body),
        headers: headers({ 'Content-Type': 'application/json' }),
      }),
  }
}
