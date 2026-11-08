/*
    Run once, after 002 and before script.sql:

        psql "$OWNER_URL" -v ON_ERROR_STOP=1 -1 -f migrations/003_api_key_origins.sql
        psql "$OWNER_URL" -v ON_ERROR_STOP=1 -1 -f script.sql

    Users with Manager access can now own an API key; key_origins lists the
    portfolio origins allowed to use it from a browser.
*/

alter table account.creds add column if not exists key_origins text[] not null default '{}';
