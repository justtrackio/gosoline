-- +goose Up
-- +goose StatementBegin
create table users
(
    id        int auto_increment primary key,
    name      varchar(255) null,
    is_active bool         null
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table users;
-- +goose StatementEnd
