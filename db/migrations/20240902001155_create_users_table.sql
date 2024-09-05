-- migrate:up
create table users (
  id integer primary key not null,
  username text unique not null,
  company text not null,
  password_hash bytea not null,
  salt bytea not null
);

-- migrate:down

drop table users;
