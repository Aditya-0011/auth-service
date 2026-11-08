alter table account.creds add column if not exists key_hash text;
alter table account.creds add column if not exists key_hint text;

do $$
begin
    if exists (
        select 1 from information_schema.columns
        where table_schema = 'account' and table_name = 'creds' and column_name = 'key'
    ) then
        update account.creds
            set key_hash = encode(sha256(convert_to(key, 'UTF8')), 'hex'),
                key_hint = left(key, 7)
            where key_hash is null;
        alter table account.creds drop column key;
    end if;
end
$$;

alter table account.creds alter column key_hash set not null;
alter table account.creds alter column key_hint set not null;
alter table account.creds alter column updatedAt set not null;

do $$
begin
    if not exists (
        select 1 from pg_constraint where conname = 'creds_key_hash_key' and conrelid = 'account.creds'::regclass
    ) then
        alter table account.creds add constraint creds_key_hash_key unique (key_hash);
    end if;
end
$$;

drop function if exists account.create(varchar, bytea);
drop function if exists account.rotate_key(int);
drop function if exists account.generate_key();
