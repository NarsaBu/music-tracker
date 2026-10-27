-- How old a release may be and still count as news (FR-3.4); was SYNC_STALE_DAYS.
ALTER TABLE settings
    ADD COLUMN new_release_days int NOT NULL DEFAULT 90 CHECK (new_release_days BETWEEN 1 AND 3650);
