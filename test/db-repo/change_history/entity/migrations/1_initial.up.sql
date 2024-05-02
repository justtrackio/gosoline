create table items (
    id int auto_increment primary key,
    change_history_author_id int,
    action varchar(10),
    name varchar(10),
    created_at datetime,
    updated_at datetime
);
