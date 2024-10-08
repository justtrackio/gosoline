-- +goose Up
create table test_model1
(
    id         int unsigned auto_increment primary key,
    name       varchar(255) null,
    updated_at datetime     null,
    created_at datetime     null
);

create table test_model2
(
    id            int unsigned auto_increment primary key,
    name          varchar(255) null,
    foo           varchar(8)   null,
    change_author varchar(255),
    updated_at    timestamp    null,
    created_at    timestamp    null
);

create table test_model3
(
    id         int unsigned auto_increment primary key,
    name       text         null,
    foo        varchar(255) null,
    updated_at timestamp    null,
    created_at timestamp    null
);

/* used to test with an existing history table */
CREATE TABLE test_model2_history_entries
(
    change_history_action    VARCHAR(8) NOT NULL DEFAULT 'insert',
    change_history_revision  int,
    change_history_action_at timestamp  NULL     DEFAULT CURRENT_TIMESTAMP,
    change_history_author_id int,
    id                       int unsigned,
    updated_at               timestamp  NULL,
    created_at               timestamp  NULL,
    change_author            varchar(255),
    name                     varchar(255),
    address                  varchar(255),
    PRIMARY KEY (change_history_revision, id)
);

create table test_model3_history_entries
(
    change_history_action    VARCHAR(8) NOT NULL DEFAULT 'insert',
    change_history_revision  int,
    change_history_action_at timestamp  NULL     DEFAULT CURRENT_TIMESTAMP,
    id                       int unsigned,
    name                     varchar(255),
    updated_at               timestamp  null,
    created_at               timestamp  null,
    PRIMARY KEY (change_history_revision, id)
);
