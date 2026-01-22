CREATE TABLE IF NOT EXISTS user_devices
(
    id              UUID                                                                          NOT NULL
        PRIMARY KEY,
    user_id         VARCHAR                                                                       NOT NULL,
    expo_push_token VARCHAR(255)                                                                  NOT NULL,
    platform        VARCHAR(20)                                                                   NOT NULL,
    date_create     TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) NOT NULL,
    date_update     TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) NOT NULL,
    last_seen_at    TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_devices_token ON user_devices (expo_push_token);
CREATE INDEX IF NOT EXISTS idx_user_devices_user_id ON user_devices (user_id);

ALTER TABLE IF EXISTS user_devices
    OWNER TO silvioubaldino;
