create type order_status as enum ('active', 'closed');

create table orders
(
    order_id     int primary key generated always as identity,
    order_status order_status not null,
    client_name  text         not null,
    cost         float4       not null,
    comments     text[] default '{}',
    contacts     text[] default '{}',
    links        text[] default '{}',
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