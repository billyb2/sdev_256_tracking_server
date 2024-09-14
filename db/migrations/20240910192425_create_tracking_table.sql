-- migrate:up
create table tracking (
  id integer primary key not null,
  tracking_number text unique not null,
  status text,
  group_name text,
  status_last_updated datetime,
  user_id int not null,
  foreign key(user_id) references users(id)
);

-- migrate:down
drop table tracking;
