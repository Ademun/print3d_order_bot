create type order_status as enum ('active', 'closed');

create table orders
(
    id     int primary key generated always as identity,
    status order_status not null,
    print_type text not null default 'Неизвестный',
    client_name  text         not null,
    cost         real         not null,
    comments     text[] default '{}',
    contacts     text[] default '{}',
    links        text[] default '{}',
    created_at   timestamptz  not null,
    closed_at    timestamptz,
    folder_path  text
);

create table order_files
(
    name  text not null,
    checksum numeric not null,
    tg_file_id text,
    order_id   int  not null,
    foreign key (order_id) references orders (id) on delete cascade
);