CREATE TABLE administration_notifications (
  notification_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  source text NOT NULL,
  category text NOT NULL,
  severity text NOT NULL CHECK (severity IN ('info','warning','critical')),
  title text NOT NULL,
  summary text NOT NULL,
  dedupe_key text NOT NULL,
  state text NOT NULL DEFAULT 'active' CHECK (state IN ('active','resolved')),
  occurrence_count integer NOT NULL DEFAULT 1,
  first_occurred_at timestamptz NOT NULL DEFAULT now(),
  last_occurred_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz,
  evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
  UNIQUE (tenant_id, dedupe_key, state)
);
CREATE INDEX administration_notifications_tenant_time_idx ON administration_notifications (tenant_id, last_occurred_at DESC);

CREATE TABLE administration_notification_inbox_state (
  notification_id text NOT NULL REFERENCES administration_notifications(notification_id) ON DELETE CASCADE,
  subject text NOT NULL,
  read_at timestamptz,
  archived_at timestamptz,
  PRIMARY KEY (notification_id, subject)
);

CREATE TABLE administration_notification_deliveries (
  delivery_id text PRIMARY KEY,
  notification_id text NOT NULL REFERENCES administration_notifications(notification_id) ON DELETE CASCADE,
  recipient_email text NOT NULL,
  channel text NOT NULL DEFAULT 'email',
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sent','failed')),
  attempts integer NOT NULL DEFAULT 0,
  error_message text,
  sent_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE administration_smtp_settings (
  tenant_id text PRIMARY KEY,
  host text NOT NULL,
  port integer NOT NULL,
  username text,
  password_ciphertext bytea,
  use_starttls boolean NOT NULL DEFAULT true,
  use_ssl boolean NOT NULL DEFAULT false,
  from_email text NOT NULL,
  from_name text NOT NULL DEFAULT 'SignalOps',
  reply_to text,
  updated_by text NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
