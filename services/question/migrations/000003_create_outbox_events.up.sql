CREATE TABLE IF NOT EXISTS outbox_events (
    id uuid PRIMARY KEY,
    aggregate_type varchar(64) NOT NULL,
    aggregate_id varchar(255) NOT NULL,
    event_type varchar(128) NOT NULL,
    payload jsonb NOT NULL,
    status varchar(32) NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    next_retry_at timestamp NOT NULL DEFAULT now(),
    sent_at timestamp NULL,
    last_error text NULL,
    created_at timestamp NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_status_next_retry_created
ON outbox_events(status, next_retry_at, created_at);