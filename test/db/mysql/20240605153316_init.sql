-- +goose Up
-- +goose StatementBegin
create table todo (
    id int auto_increment,
    name varchar(255) not null,
    data json,
    created_at timestamp null on update CURRENT_TIMESTAMP,
    constraint todo_pk primary key (id)
);
INSERT INTO todo (name)
VALUES ("my todo for today");
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
drop table `todo`;
-- +goose StatementEnd