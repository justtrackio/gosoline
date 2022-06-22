create table test_models
(
    id         int unsigned auto_increment
        primary key,
    name       varchar(255) null,
    updated_at timestamp    null,
    created_at timestamp    null
);

create table test_manies
(
    id         int unsigned auto_increment
        primary key,
    name       varchar(255) null,
    updated_at timestamp    null,
    created_at timestamp    null
);

create table test_many_to_manies
(
    id           int unsigned auto_increment
        primary key,
    test_many_id int unsigned,
    many_id      int unsigned,
    other_id     int unsigned,
    created_at   timestamp null,
    updated_at   timestamp null,
    CONSTRAINT test_many_to_manies_many FOREIGN KEY (many_id) REFERENCES test_manies (id),
    CONSTRAINT test_many_to_manies_other FOREIGN KEY (other_id) REFERENCES test_manies (id)
)