-- +goose Up
create table items
(
    id         int unsigned auto_increment primary key,
    updated_at timestamp    null,
    created_at timestamp    null
);
