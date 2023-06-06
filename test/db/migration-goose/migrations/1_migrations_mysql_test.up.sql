-- +goose Up
create table mysql_test_models
(
  id int unsigned auto_increment
    primary key,
  name varchar(255) null,
  updated_at timestamp null,
  created_at timestamp null
);

create table mysql_plain_writer_test
(
  id int unsigned auto_increment
    primary key,
  name varchar(255) null,
  updated_at timestamp null,
  created_at timestamp null
);