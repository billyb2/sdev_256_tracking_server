-- migrate:up
create table tokens (
  token string primary key not null,
  user_id int not null,
  foreign key(user_id) references users(user_id)

);

-- migrate:down
drop table tokens;
