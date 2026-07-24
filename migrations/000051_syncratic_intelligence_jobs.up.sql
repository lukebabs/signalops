CREATE TABLE syncratic_intelligence_jobs (
  job_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  app_id TEXT NOT NULL DEFAULT 'marketops',
  use_case TEXT NOT NULL DEFAULT 'daily_market_surveillance',
  subject_symbol TEXT NOT NULL,
  session_date DATE NOT NULL,
  context_window_id TEXT NOT NULL REFERENCES syncratic_context_windows(context_window_id),
  evidence_digest TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','running','completed','retryable_failed','failed')),
  attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  max_attempts INTEGER NOT NULL DEFAULT 3 CHECK (max_attempts > 0),
  lease_expires_at TIMESTAMPTZ,
  ask_query_id TEXT,
  syncratic_insight_id TEXT REFERENCES syncratic_insights(syncratic_insight_id),
  error_code TEXT,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ,
  UNIQUE (tenant_id, app_id, use_case, subject_symbol, session_date, evidence_digest)
);

CREATE INDEX syncratic_intelligence_jobs_ready_idx
  ON syncratic_intelligence_jobs (status, created_at)
  WHERE status IN ('queued', 'retryable_failed');

CREATE TABLE syncratic_analyst_questions (
  analyst_question_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  context_window_id TEXT NOT NULL REFERENCES syncratic_context_windows(context_window_id),
  syncratic_insight_id TEXT REFERENCES syncratic_insights(syncratic_insight_id),
  question TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','running','completed','failed')),
  answer TEXT,
  claim_categories JSONB NOT NULL DEFAULT '[]'::jsonb,
  citation_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
  ask_query_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE INDEX syncratic_analyst_questions_context_idx
  ON syncratic_analyst_questions (tenant_id, context_window_id, created_at DESC);
