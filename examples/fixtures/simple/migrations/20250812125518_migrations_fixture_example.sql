-- +goose Up
-- +goose StatementBegin
create table orm_fixture_examples
(
    id int unsigned auto_increment
        primary key,
    name varchar(255) null,
    updated_at timestamp null,
    created_at timestamp null
);

create table plain_fixture_example
(
    id int unsigned auto_increment
        primary key,
    name varchar(255) null,
    updated_at timestamp null,
    created_at timestamp null
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS orm_fixture_examples;
DROP TABLE IF EXISTS plain_fixture_example;
-- +goose StatementEnd
