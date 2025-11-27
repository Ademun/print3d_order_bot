pragma foreign_keys = on;

create table orders
(
    order_id     integer primary key,
    order_status integer not null,
    client_name  text    not null,
    created_at   text    not null,
    closed_at    text,
    folder_path  text    not null
);

create table order_files
(
    file_name  text not null,
    tg_file_id text not null,
    order_id integer not null,
    foreign key (order_id) references orders(order_id) on delete cascade
);