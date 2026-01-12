CREATE TABLE IF NOT EXISTS user_consents
(
    id           UUID                                                                          NOT NULL
        PRIMARY KEY,
    user_id      VARCHAR                                                                       NOT NULL,
    term_version VARCHAR(50)                                                                   NOT NULL,
    agreed_at    TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) NOT NULL,
    ip_address   VARCHAR(45),
    user_agent   TEXT
);

ALTER TABLE IF EXISTS user_consents
    OWNER TO silvioubaldino;

