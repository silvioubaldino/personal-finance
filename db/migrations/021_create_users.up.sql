CREATE TABLE IF NOT EXISTS users
(
    id         VARCHAR                                                                       NOT NULL
        PRIMARY KEY,
    language   VARCHAR                  DEFAULT 'pt-BR'                                      NOT NULL,
    currency   VARCHAR                  DEFAULT 'BRL'                                        NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) NOT NULL
);

ALTER TABLE IF EXISTS users
    OWNER TO silvioubaldino;

-- Backfill from user_preferences (1:1, drops afterwards)
INSERT INTO users (id, language, currency, created_at, updated_at)
SELECT user_id, language, currency, date_create, date_update
FROM user_preferences
ON CONFLICT (id) DO NOTHING;

-- Defensive backfill: any user_id referenced elsewhere but missing prefs
INSERT INTO users (id)
SELECT DISTINCT user_id FROM user_consents
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id)
SELECT DISTINCT user_id FROM user_devices
ON CONFLICT (id) DO NOTHING;

-- Cascade FKs on user-meta tables (financial data stays manual until full migration)
ALTER TABLE user_consents
    ADD CONSTRAINT fk_user_consents_user
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE user_devices
    ADD CONSTRAINT fk_user_devices_user
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

DROP TABLE user_preferences;
