CREATE TABLE kv (
  key text PRIMARY KEY,
  value jsonb NOT NULL
);
INSERT INTO kv VALUES ('last_telegram_update', '0') ON CONFLICT (key) DO NOTHING;

CREATE TABLE channel (
  id text PRIMARY KEY DEFAULT md5(random()::text),
  jq text NOT NULL
);

CREATE TABLE subscription (
  channel text NOT NULL REFERENCES channel (id),
  chat_id numeric(15) NOT NULL,
  PRIMARY KEY (channel, chat_id)
);
