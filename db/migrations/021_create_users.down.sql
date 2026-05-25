CREATE TABLE IF NOT EXISTS user_preferences
(
    user_id     VARCHAR                                                                       NOT NULL
        PRIMARY KEY,
    language    VARCHAR                  DEFAULT 'pt-BR'                                      NOT NULL,
    currency    VARCHAR                  DEFAULT 'BRL'                                        NOT NULL,
    date_create TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) NOT NULL,
    date_update TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) NOT NULL
);

ALTER TABLE IF EXISTS user_preferences
    OWNER TO silvioubaldino;

INSERT INTO user_preferences (user_id, language, currency, date_create, date_update)
SELECT id, language, currency, created_at, updated_at
FROM users
ON CONFLICT (user_id) DO NOTHING;

ALTER TABLE user_consents
    DROP CONSTRAINT IF EXISTS fk_user_consents_user;

ALTER TABLE user_devices
    DROP CONSTRAINT IF EXISTS fk_user_devices_user;

DROP TABLE IF EXISTS users;
