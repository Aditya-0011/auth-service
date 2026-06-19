/*
    OutCodes:
        -1: Success

        0: Record exists

        1: Record not found

        2: Db Error
*/

create extension if not exists pgcrypto;

create schema if not exists account;

create table if not exists account.creds (
    id int primary key,
    email varchar(100) not null unique,
    password bytea not null,
    key text not null unique,
	updatedAt timestamptz default now()
);

create or replace function account.generate_key(out key text)
as $$
begin
    key := 'pk_' || encode(gen_random_bytes(24), 'hex');
end;
$$ language plpgsql security definer;

create or replace function account.create(
    in p_email varchar, 
    in p_password bytea, 
    out id int,
    out outcode smallint)
as $$
begin
    id := 1;
    insert into account.creds(id, email, password, key) values (id, p_email, p_password, account.generate_key());
    outcode := -1;
exception   
    when unique_violation then
        id := 0;
        outcode := 0;
    when others then
        id := 0;
        outcode := 2;
end;
$$ language plpgsql security definer;

create or replace function account.rotate_key(in p_id int, out outcode smallint)
as $$
declare
    rows_affected int;
begin
    update account.creds
        set key = account.generate_key(), updatedAt = now()
        where id = p_id;

    get diagnostics rows_affected = row_count;

    if rows_affected = 0 then
        outcode := 1;
    else
        outcode := -1;
    end if;
exception
    when others then
        outcode := 2;
end;
$$ language plpgsql security definer;

create user auth_service with password 'test@12345';
revoke all on database infra from auth_service;
grant connect on database infra to auth_service;
grant usage on schema public to auth_service;
grant usage on schema account to auth_service;
grant select on table account.creds to auth_service;
grant execute on function 
    account.create (varchar, bytea),
    account.rotate_key(int)
to auth_service;
revoke insert, update, delete on account.creds from auth_service;