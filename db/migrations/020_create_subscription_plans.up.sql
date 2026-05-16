CREATE TABLE subscription_plans (
  id             VARCHAR PRIMARY KEY,
  name           VARCHAR NOT NULL,
  price          DECIMAL(10,2) NOT NULL,
  currency       VARCHAR(3) NOT NULL DEFAULT 'BRL',
  frequency      INT NOT NULL DEFAULT 1,
  frequency_type VARCHAR NOT NULL DEFAULT 'months',
  is_active      BOOLEAN NOT NULL DEFAULT false,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
