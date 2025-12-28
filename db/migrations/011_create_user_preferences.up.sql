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

