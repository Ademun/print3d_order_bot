create type order_status as enum ('open', 'closed');

create table orders
(
    order_id     int primary key generated always as identity,
    order_status order_status not null,
    client_name  text         not null,
    comments     text[],
    contacts     text[],
    links        text[],
    created_at   timestamptz  not null,
    closed_at    timestamptz,
    folder_path  text
);

create table order_files
(
    file_name  text not null,
    tg_file_id text,
    order_id   int  not null,
    foreign key (order_id) references orders (order_id) on delete cascade
);