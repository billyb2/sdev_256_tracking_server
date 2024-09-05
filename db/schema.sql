CREATE TABLE IF NOT EXISTS "schema_migrations" (version varchar(128) primary key);
CREATE TABLE users (
  id integer primary key not null,
  username text unique not null,
  company text not null,
  password_hash bytea not null,
  salt bytea not null
);
CREATE TABLE tokens (
  token string primary key not null,
  user_id int not null,
  foreign key(user_id) references users(user_id)

);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20240902001155'),
  ('20240905192308');
