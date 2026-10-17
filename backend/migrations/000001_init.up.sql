-- Base types and the singleton settings row (docs/05-data-model.md).
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE release_primary_type AS ENUM ('album', 'single', 'ep', 'broadcast', 'other');
CREATE TYPE credit_role          AS ENUM ('main', 'collaboration', 'featured');
CREATE TYPE notification_channel AS ENUM ('email', 'rss');
CREATE TYPE notification_status  AS ENUM ('pending', 'sent', 'failed');
CREATE TYPE digest_frequency     AS ENUM ('daily', 'weekly');
CREATE TYPE import_item_status   AS ENUM ('pending', 'matched', 'needs_review', 'not_found', 'skipped', 'failed');

CREATE TABLE settings (
    id                       smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    primary_types            release_primary_type[] NOT NULL DEFAULT '{album,ep,single}',
    excluded_secondary_types text[] NOT NULL DEFAULT
        '{Compilation,DJ-mix,Interview,Spokenword,Audiobook,"Audio drama",Demo,"Field recording"}',
    include_featured         boolean NOT NULL DEFAULT false,
    notify_announcements     boolean NOT NULL DEFAULT true,
    email_enabled            boolean NOT NULL DEFAULT false,
    email_to                 text,
    digest_frequency         digest_frequency NOT NULL DEFAULT 'daily',
    digest_weekday           smallint CHECK (digest_weekday BETWEEN 0 AND 6),
    digest_time              time NOT NULL DEFAULT '09:00',
    timezone                 text NOT NULL DEFAULT 'UTC',
    last_digest_at           timestamptz,
    next_digest_at           timestamptz,
    rss_enabled              boolean NOT NULL DEFAULT true,
    -- 32 random bytes, base64url without padding
    feed_token               text NOT NULL DEFAULT rtrim(translate(encode(gen_random_bytes(32), 'base64'), '+/', '-_'), '='),
    CHECK (digest_frequency = 'daily' OR digest_weekday IS NOT NULL)
);

INSERT INTO settings (id) VALUES (1);
