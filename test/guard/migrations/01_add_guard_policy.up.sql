create table guard_policies
(
	id varchar(255) not null,
	description varchar(255) not null,
	effect varchar(255) not null,
	updated_at datetime not null,
	created_at datetime not null,
	constraint guard_policies_pk
		primary key (id)
);

create table guard_subjects
(
	id VARCHAR(255) not null,
	name VARCHAR(255) not null,
	constraint table_name_pk
		primary key (id, name)
);

create table guard_resources
(
	id VARCHAR(255) not null,
	name VARCHAR(255) not null,
	constraint table_name_pk
		primary key (id, name)
);

create table guard_actions
(
	id VARCHAR(255) not null,
	name VARCHAR(255) not null,
	constraint table_name_pk
		primary key (id, name)
);
