CREATE TABLE IF NOT EXISTS "schema_migrations" (version varchar(128) primary key);
CREATE TABLE users (
  id serial primary key,
  username text unique not null,
  company text not null,
  password_hash text not null,
  salt bytea not null
);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20240902001155');
