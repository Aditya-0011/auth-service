/*
    OutCodes:
        -1: Success

        0: Record exists

        1: Record not found

        2: Db Error
*/

create schema if not exists account;

create table if not exists account.creds (
    id int primary key,
    email varchar(100) not null unique,
    password varchar(100) not null,
    updatedAt timestamptz default now()
);

create or replace function account.create(in p_email varchar, in p_password varchar, out id int, out outcode smallint)
as $$
begin
    id := 1;
    insert into account.creds(id, email, password) values (id, p_email, p_password);
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

create user auth_service with password 'test@12345'
revoke all on database infra from auth_service;
grant connect on database infra to auth_service;
grant usage on schema public to auth_service;
grant usage on schema account to auth_service;
grant select on table account.creds to auth_service;
grant execute on function account.create (varchar, varchar) to auth_service;
revoke insert, update, delete on account.creds to auth_service;