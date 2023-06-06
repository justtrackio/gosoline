-- +goose Up
create table guard_policies
(
    id         varchar(255) not null
        primary key,
    policy     json         not null,
    updated_at datetime     not null,
    created_at datetime     not null
);