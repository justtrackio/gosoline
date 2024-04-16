create table todos
(
    id         int       auto_increment primary key,
    text       text      not null,
    due_date   timestamp not null,
    updated_at timestamp not null,
    created_at timestamp not null
);
