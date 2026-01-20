-- +goose Up
-- +goose StatementBegin
create table todo (
    id int auto_increment,
    name varchar(255) not null,
    data json,
    `index` int not null,
    created_at timestamp null on update CURRENT_TIMESTAMP,
    constraint todo_pk primary key (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table `todo`;
-- +goose StatementEnd
