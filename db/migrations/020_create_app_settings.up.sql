CREATE TABLE app_settings (
  key        VARCHAR PRIMARY KEY,
  value      VARCHAR NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO app_settings (key, value) VALUES ('plus_price', '9.90');
