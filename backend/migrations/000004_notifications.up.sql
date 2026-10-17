-- Notification outbox (ADR-4, docs/08-notifications.md). The rss channel also backs the web feed.
CREATE TABLE notifications (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    release_group_mbid uuid NOT NULL REFERENCES release_groups (mbid),
    artist_mbid        uuid NOT NULL REFERENCES artists (mbid),   -- "found via"
    credit_role        credit_role NOT NULL,
    channel            notification_channel NOT NULL,
    status             notification_status  NOT NULL DEFAULT 'pending',
    attempts           int NOT NULL DEFAULT 0,
    next_attempt_at    timestamptz NOT NULL DEFAULT now(),
    last_error         text,
    created_at         timestamptz NOT NULL DEFAULT now(),
    sent_at            timestamptz,
    UNIQUE (release_group_mbid, channel)                           -- FR-5.8
);
CREATE INDEX ON notifications (channel, status, next_attempt_at);
CREATE INDEX ON notifications (channel, created_at DESC, id DESC);
